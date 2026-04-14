# HashiCorp Vault Base Integration — Design Spec

**Date:** 2026-04-14
**Issue:** [#3928](https://github.com/superplanehq/superplane/issues/3928)
**Scope:** Base integration — Token auth, KV v2, Get Secret component only.

---

## Decisions

- **Auth method:** Token only (simplest, covers the majority of real use cases; other methods deferred to follow-up).
- **KV version:** KV v2 only (modern Vault default with versioned secrets; v1 can be added later).
- **Components:** `Get Secret` only, as specified in the issue.
- **Triggers:** None (Vault is pull-only; no event push mechanism to integrate with).

---

## File Structure

```
pkg/integrations/hashicorp_vault/
  vault.go                          # integration struct, Configuration(), Sync(), registration
  client.go                         # HTTP client with token + namespace header
  get_secret.go                     # Get Secret component
  example.go                        # embeds example output JSON
  example_output_get_secret.json    # example payload for UI preview
  vault_test.go                     # Sync() unit tests
  get_secret_test.go                # Setup() and Execute() unit tests
```

One blank import added to `pkg/server/server.go` (alphabetically: after `harness`, before `hetzner`).

---

## Integration Config (`vault.go`)

### `Configuration()` fields

| Field       | Type   | Required | Sensitive | Description |
|-------------|--------|----------|-----------|-------------|
| `baseURL`   | string | yes      | no        | Vault server URL, e.g. `https://vault.example.com` |
| `namespace` | string | no       | no        | Vault Enterprise namespace; sent as `X-Vault-Namespace` header. Leave empty for community edition. |
| `token`     | string | yes      | yes       | Vault token (`hvs.…` or `s.…`) |

### `Sync()` behaviour

Calls `GET /v1/auth/token/lookup-self` with the configured token. On success, calls `ctx.Integration.Ready()`. On failure (401, network error, etc.), returns an error.

### Other methods

- `Triggers()` — returns empty slice.
- `HandleRequest()` — no-op (no webhooks).
- `Actions()` / `HandleAction()` — no-op.
- `Cleanup()` — no-op.

---

## Client (`client.go`)

```
type Client struct {
    BaseURL   string
    Token     string
    Namespace string
    http      core.HTTPContext
}
```

`NewClient(httpCtx, integrationCtx)` reads `baseURL`, `token`, `namespace` from integration config. Returns error if `baseURL` or `token` are empty.

`execRequest(method, path, body)` — builds the full URL (`BaseURL + path`), sets:
- `X-Vault-Token: <token>`
- `X-Vault-Namespace: <namespace>` (only when non-empty)
- `Content-Type: application/json`

Returns error for non-2xx status codes, including the response body in the error message.

`LookupSelf()` — calls `GET /v1/auth/token/lookup-self`, returns error on failure. Used by `Sync()` to verify credentials.

`GetKVSecret(mount, path string)` — calls `GET /v1/<mount>/data/<path>`, returns a parsed `KVSecret` struct.

---

## `Get Secret` Component (`get_secret.go`)

### Component identity

| Method | Value |
|--------|-------|
| `Name()` | `"hashicorp_vault.getSecret"` |
| `Label()` | `"Get Secret"` |
| `Icon()` | `"lock"` |
| `Color()` | `"gray"` |

### `Configuration()` fields

| Field   | Type   | Required | Default    | Description |
|---------|--------|----------|------------|-------------|
| `mount` | string | no       | `"secret"` | KV v2 mount path. The `Default` field in `Configuration()` must be set to `"secret"` so the UI pre-fills it. |
| `path`  | string | yes      | —          | Secret path, e.g. `myapp/db` |
| `key`   | string | no       | —          | If set, extract only this key from the secret data; populates `value` in the output |

### `Setup()` validation

- Returns error if `path` is empty.

### `Execute()` flow

1. Decode config into `getSecretSpec` struct.
2. Build `Client` via `NewClient`.
3. Call `client.GetKVSecret(spec.Mount, spec.Path)`.
4. Build output payload (see below).
5. Emit on `core.DefaultOutputChannel` with type `"hashicorp_vault.secret"`.

### Output payload shape

```json
{
  "mount": "secret",
  "path": "myapp/db",
  "data": { "username": "admin", "password": "s3cr3t" },
  "value": "admin",
  "metadata": {
    "version": 3,
    "created_time": "2025-01-01T00:00:00Z",
    "deletion_time": "",
    "destroyed": false
  }
}
```

- `data` — the full KV data map (always present).
- `value` — only populated when `key` config field is set; the string value of that key. If the key does not exist in `data`, returns an error.
- `metadata` — version info from the KV v2 response.

---

## Tests

### `vault_test.go`

- `TestSync_Success` — mock returns 200 from `/v1/auth/token/lookup-self`; assert `integrationCtx.State == "ready"`.
- `TestSync_InvalidToken` — mock returns 403; assert error returned, state not ready.
- `TestSync_MissingToken` — empty token in config; assert error without making HTTP request.
- `TestSync_MissingBaseURL` — empty baseURL; assert error without making HTTP request.

### `get_secret_test.go`

- `TestGetSecret_Execute_AllData` — `key` not set; assert full `data` map in emitted payload, `value` empty.
- `TestGetSecret_Execute_SpecificKey` — `key` set; assert `value` populated with correct string.
- `TestGetSecret_Execute_KeyNotFound` — `key` set to a key not present in `data`; assert error.
- `TestGetSecret_Execute_APIError` — mock returns 403; assert error propagated.
- `TestGetSecret_Setup_MissingPath` — empty `path`; assert error from `Setup()`.
- `TestGetSecret_Execute_DefaultMount` — `mount` not set; assert request uses `"secret"` as mount.
