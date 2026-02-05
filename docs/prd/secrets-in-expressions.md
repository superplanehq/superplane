# PRD: Secrets in Expressions

## Overview

This document describes how to support referencing **organization secrets** inside node configuration expressions, so that sensitive values (API keys, tokens, etc.) can be used in workflow nodes without being stored in the workflow or in node execution records.

## Goals

- Allow users to reference secrets in expressions using the syntax `{{ secrets("secret-name").key }}`.
- Ensure **secret values are never persisted** in node execution configuration. They must be resolved only at **runtime** when the node actually runs.
- Reuse the existing expression engine and secrets model; scope resolution to the workflow’s organization.

## Non-Goals (out of scope for this PRD)

- Secret references in triggers’ queue-time configuration (e.g. webhook verification) — can be a follow-up.
- Workflow- or canvas-scoped secrets (only organization-scoped secrets are in scope).
- Exposing secret names/keys in UI autocomplete or validation (optional enhancement).

---

## User-Facing Syntax

- **Syntax:** `{{ secrets("name").key }}`
  - `"name"` is the **secret name** (organization-scoped).
  - `.key` is a **key** within that secret’s key-value data (e.g. `api_key`, `token`).
- **Examples:**
  - `{{ secrets("slack-credentials").token }}`
  - `{{ "Bearer " + secrets("api-keys").auth_token }}`
  - Mixed with other expressions: `{{ $.trigger.body.user + " " + secrets("internal").salt }}`

If the secret or key does not exist, expression evaluation should fail with a clear error so the execution can fail safely rather than proceed with empty or wrong data.

---

## Security Requirement: No Secrets in Node Execution

- **Node execution records** (e.g. `workflow_node_executions.configuration`) must **never** contain resolved secret values.
- Secrets must be **dynamically resolved only at runtime** when the node’s component (or trigger) is about to run.
- Stored configuration may contain the **expression string** (e.g. `{{ secrets("my-secret").key }}`); it must not contain the actual secret value.

This keeps logs, DB dumps, and execution history free of sensitive data and avoids leaking secrets through existing “show configuration” or debugging surfaces.

---

## Current State (Relevant Parts)

### Expressions

- Expressions use the form `{{ <expr> }}` and are evaluated with the [expr](https://github.com/expr-lang/expr) engine.
- Resolution is implemented in `pkg/workers/contexts/node_configuration_builder.go`:
  - `ResolveExpression` / `resolveExpression` replace `{{ ... }}` with evaluated values.
  - Environment includes `$` (message chain), `config` (blueprint node config), and functions such as `root()` and `previous()`.
- Configuration is **resolved once** when building the context for a node (e.g. in the process queue or when creating a child execution). The **resolved** configuration is then stored on the **node execution** and passed to the component as `ExecutionContext.Configuration`.

### Configuration storage

- **Canvas node:** `CanvasNode.Configuration` holds the user-defined configuration (may contain expression strings).
- **Node execution:** `CanvasNodeExecution.Configuration` holds the **resolved** configuration used for that run. This is what gets passed to `Execute(ctx)` as `ctx.Configuration` and is the record we must keep free of secret values.

### Secrets

- **Model:** `pkg/models/secret.go` — secrets are key-value stores with `DomainType` and `DomainID` (e.g. organization).
- **Lookup:** `FindSecretByNameInTransaction(tx, domainType, domainID, name)`.
- **Provider:** `pkg/secrets` (e.g. `LocalProvider`) loads and decrypts secret data; `Load()` returns `map[string]string` (key → value).
- Today, secrets are organization-scoped (`DomainTypeOrganization`); no workflow- or canvas-level secrets.

---

## Proposed Design

### 1. Two-phase resolution

- **Phase 1 — Build-time (existing flow):** When building configuration to be **stored** on a node execution (process queue, blueprint child execution, etc.):
  - Detect expressions that reference `secrets(...)` (e.g. via AST or a simple heuristic).
  - For those expressions, **do not resolve** — leave the expression string in the config (e.g. `"{{ secrets(\"name\").key }}"`). All other expressions (e.g. `$`, `previous()`, `root()`) are resolved as today.
- **Phase 2 — Runtime (new):** Immediately before running the component (e.g. in `NodeExecutor.executeComponentNode`), run a **runtime resolution** step:
  - Take the execution’s stored configuration.
  - Walk the config (recursively for nested objects/arrays). For any **string** value that looks like an expression (e.g. contains `{{` and `}}`), **completely re-evaluate** it with full runtime env (including `secrets()`, `$`, `root()`, `previous()`, etc.) and substitute the result.
  - Pass this **fully resolved** configuration to the component as `ctx.Configuration`. This resolved config is **not** written back to the DB.

So: **stored config** = no secret values (only expression text where we deferred); **runtime config** = fully resolved in memory. If it looks like an expression, re-evaluate it at runtime.

### 2. `secrets()` in the expression engine

- Add a built-in (or env) function `secrets(name string)` that:
  - Is only available during **runtime** resolution (not during build-time resolution for storage).
  - Uses execution context: `tx`, `workflow.OrganizationID`, encryptor.
  - Looks up the secret by name for the org (`DomainTypeOrganization`, `domainID = workflow.OrganizationID`), loads key-value map via existing provider, and returns an object/map so that `.key` returns the value.
- If the secret or key is missing, return a clear error so the expression fails and the execution can fail with a safe message (no secret value in the error).

### 3. Where to plug in runtime resolution

- **Component execution:** In `pkg/workers/node_executor.go`, in `executeComponentNode`, **before** building `ExecutionContext` and calling `component.Execute(ctx)`:
  - Run a “runtime config resolver” that takes `execution.Configuration.Data()`, organization ID, tx, and encryptor.
  - Resolver walks the config (recursively for nested objects/arrays). For any string value that looks like an expression (e.g. contains `{{` and `}}`), **completely re-evaluate** it with full env (including `secrets()`); substitute the result.
  - Set `ctx.Configuration` to this fully resolved config (never persist it).
- **Blueprint nodes:** When building config for a child execution, the same rule applies: any expression containing `secrets()` is left as the expression string. When that child execution later runs, the runtime resolution step re-evaluates those values.
- **Process queue / CreateExecution:** The config that is stored in the new execution record must be the result of Phase 1 only (no secret values).

### 4. Build-time detection of `secrets()` usage

- During Phase 1, when resolving an expression string:
  - If the expression (or any sub-expression in a `{{ ... }}` block) contains a call to `secrets(...)` (e.g. detected by parsing the expr AST or a regex), do **not** resolve it — leave the full expression string (e.g. `"{{ secrets(\"name\").key }}"`) in the stored config.

### 5. Scoping and authorization

- Secrets are resolved in the context of the **workflow’s organization** (already available when executing a node).
- No new authorization is required for “reading” the secret at runtime if the execution is already allowed to run in that org (execution is already org-scoped). If the product later adds “secret use” permissions, that can be enforced in the runtime resolver.

---

## Edge Cases and Open Questions

- **Missing secret or key:** Fail the expression evaluation and thus the execution, with a message like “secret not found: &lt;name&gt;” or “secret key not found: &lt;key&gt;”. Do not expose the secret value in the error.
- **Nested config (objects/arrays):** Runtime resolver recurses into nested maps and arrays; any string value that looks like an expression is re-evaluated.
- **Trigger and process-queue path:** Triggers that build config at queue time (e.g. webhook) currently also use `NodeConfigurationBuilder.Build()`. The same rule applies: do not resolve `secrets()` at queue time; leave it for runtime. If the “runtime” for a trigger is the process-queue execution context, then the runtime resolution step must run there too before the trigger/component runs (so that when an execution is created and later run, its stored config still has no secrets, and the first time secrets are injected is when the node is executed).
- **Logging and debugging:** Ensure no logging of `ctx.Configuration` (or the resolved config) in a way that would print secret values. Prefer logging only “config keys” or redacting known sensitive keys if config is ever logged.
- **UI:** Optionally, in the expression editor, support autocomplete or validation for `secrets("...")` (e.g. list org secret names). Not required for the first version.

---

## Success Criteria

- Users can use `{{ secrets("name").key }}` in node configuration fields that support expressions.
- No secret value is ever written to `workflow_node_executions.configuration` (or any other persisted execution/config store).
- Secret values are only computed at the moment of node execution and only passed in memory to the component.
- Missing secret or key results in a clear execution failure without leaking secret data.

---

## References

- Expression resolution: `pkg/workers/contexts/node_configuration_builder.go`
- Node execution and config flow: `pkg/workers/node_executor.go`, `pkg/workers/contexts/process_queue_context.go`
- Secrets model and provider: `pkg/models/secret.go`, `pkg/secrets/provider.go`, `pkg/secrets/local_provider.go`
- Execution context: `pkg/core/component.go` (`ExecutionContext.Configuration`)
