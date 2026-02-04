# Secret resolution at execution time (design outline)

## Goal

Resolve `secret("name", "key")` only when a component actually runs, and **never** persist the resolved secret value in the execution’s `configuration` column. Stored config should keep a placeholder; the real value is injected when building the runtime config for the component.

## Current behavior (problem)

1. **Config build** (queue worker or executor): `NodeConfigurationBuilder.Build(node.Configuration.Data())` resolves all expressions, including `secret("name", "key")`, and returns a fully resolved map (with plain secret strings).
2. **Persistence**: That resolved map is stored as `execution.Configuration` (e.g. `CreatePendingChildExecution(..., config)`, `ctx.CreateExecution()` with `Configuration: datatypes.NewJSONType(config)`).
3. **Result**: Every execution row contains the secret in its `configuration` JSON.

## Target behavior

1. **Config build**: Resolve everything **except** `secret(...)`. For fields that contain `secret("name", "key")`, keep a **placeholder** in the stored config (e.g. a small struct or marker that encodes secret name + key, not the value).
2. **Persistence**: Store this “partially resolved” config (with placeholders) in `execution.Configuration`.
3. **Execution time**: When the component is about to run, take `execution.Configuration.Data()`, replace each placeholder by calling the secret provider, and pass the **fully resolved** config only into the component’s `Execute(ctx)` (never write it back to the DB).

So: **resolve secrets only in the path that builds the config handed to `Execute()`**, and ensure that path never persists the result.

## Design outline

### 1. Placeholder representation

- **Option A – Inline marker in string**: Keep the field as a string and use a sentinel that still parses as expr, e.g. `"{{ secret(\"my-api\", \"api_key\") }}"` is **not** resolved during config build; stored config literally contains that string. At execution time, a single “resolve secrets in config” step re-evaluates only those expressions (or scans for `secret(` and replaces).
- **Option B – Structured placeholder**: During config build, when we encounter `secret("name", "key")`, we don’t resolve it; instead we store a small structure, e.g. `{"__secret": true, "name": "my-api", "key": "api_key"}`. At execution time, walk the config and replace any `__secret` object by loading the secret.

Choice affects: how we detect “this value is a secret reference” when we build config, and how we replace it at execution time. Option A reuses expr; Option B is explicit and easy to redact in APIs (strip or mask `__secret` objects).

### 2. Where to “not resolve” secrets

- **NodeConfigurationBuilder**: When resolving an expression (inside `resolveExpression` or the `{{ ... }}` replacement), either:
  - **Option A**: Detect `secret(...)` in the expression and, instead of calling `resolveSecret`, return a placeholder (string or struct).
  - **Option B**: Don’t add `secret()` to the expr env used for **config build**; only add it for a separate “execution-time only” resolution step. Then any `{{ secret("n","k") }}` would stay literal in stored config (or be represented as a placeholder).
- So: **config build** must either (1) return placeholders for secret references, or (2) leave them as literal strings / structured placeholders that are only expanded later.

### 3. Where to resolve placeholders (execution time)

- **Single choke point**: The place where we currently pass `execution.Configuration.Data()` into the component (e.g. `ctx.Configuration` in `core.ExecutionContext`) should instead receive a **copy** of that config with all secret placeholders resolved.
- **Locations** (today):
  - **Node executor** (`executeComponentNode`): builds `core.ExecutionContext{ ..., Configuration: execution.Configuration.Data(), ... }`. Here we’d set `Configuration: resolveSecretsInConfig(tx, execution.WorkflowID, execution.Configuration.Data(), orgID, encryptor)` (or equivalent).
  - **Process queue context**: `ctx.Configuration` is the built config; today it’s the same config that gets stored in `CreateExecution`. We need to split:
    - **Stored**: config with placeholders (no secret values).
    - **Runtime**: resolve placeholders when creating `ProcessQueueContext` / when the component runs, and pass only the resolved map into the component.
- So: **one helper** like `ResolveSecretsInConfig(tx, workflowID, config, orgID, encryptor) (map[string]any, error)` that walks the config, finds placeholders (or `secret(...)` strings), calls the secret provider, and returns a new map with secrets filled in. Call it only when constructing the context that is passed to `Execute()`, never when writing to `execution.Configuration`.

### 4. Schema / backward compatibility

- **Existing executions**: Already have resolved secrets in `configuration`. Options: (1) Leave as-is (old runs keep secrets in DB); (2) One-time migration to replace known secret values with placeholders (complex and error-prone). Recommend (1) unless there’s a hard compliance requirement.
- **New executions**: Always store config with placeholders (or literal `secret(...)` strings / structured placeholders). No change to DB schema; only the **content** of the JSON changes (placeholders instead of values).

### 5. Edge cases and constraints

- **Blueprint nodes**: First child’s config is built in `executeBlueprintNode` and stored via `CreatePendingChildExecution`. That config must use the same placeholder strategy so we never store resolved secrets. When that child execution later runs, we resolve placeholders in its config at execution time as above.
- **ExpressionEnv / lazy expr**: If any code path builds an expression env and evaluates expressions that reference `secret()`, that path must either (1) receive already-resolved config, or (2) have access to orgID + encryptor and resolve secrets on the fly. So “resolve at execution time” really means “resolve when we need the value for execution,” not “resolve when we build the execution record.”
- **Sensitive fields**: If we use structured placeholders (e.g. `__secret`), we can add a convention: when returning execution config via API or logs, replace `__secret` placeholders with a redacted value (e.g. `"***"`) so we never leak secret names/keys if desired (or allow names for debugging, per policy).

### 6. Implementation order (when you implement)

1. Define placeholder format (string vs struct) and add a small `resolveSecretsInConfig(...)` that walks a config map and replaces placeholders using existing secret provider.
2. Change `NodeConfigurationBuilder` so that during `Build()` it does **not** resolve `secret(...)` (either by returning placeholders or by leaving literal expr strings).
3. Ensure every path that **persists** config (e.g. `CreatePendingChildExecution`, `ctx.CreateExecution`) uses the output of `Build()` without an extra resolution step.
4. In the **single place** where we pass config into the component (executor + process queue context), call `resolveSecretsInConfig()` and pass the result as `Configuration`; never persist that result.
5. Tests: (1) New execution rows have no plain secret in `configuration`. (2) Component still receives the correct resolved config and runs successfully.

---

**Summary**: Keep `secret("name", "key")` as a placeholder in stored config; resolve it only in the path that builds the runtime config for `Execute()`, and never write that resolved config back to the execution row.
