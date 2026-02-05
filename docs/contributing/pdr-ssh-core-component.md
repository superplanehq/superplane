# PDR: SSH as a Core Component

**Product Design Record**  
**Title:** Convert SSH from an Integration to a Core Component with Single Action and Secret-Based Authentication  
**Status:** Draft  
**Last updated:** 2026-02-05

---

## 1. Executive Summary

This document describes the design for converting the existing **SSH integration** (`pkg/integrations/ssh`) into a **core component** (`pkg/components/ssh`). The new component will:

- Expose **one action only**: run an SSH command on a remote host.
- Use **organization Secrets** for all authentication (no integration installation or per-install credentials).
- Support **two authentication modes** via secrets: **SSH key** (private key, optional passphrase) and **password**. The user explicitly chooses the auth method and which organization Secret and key name(s) to use for credentials.

The result is a simpler, more consistent UX: users add an "SSH Command" node to a workflow, configure host/connection details and the command, and reference a secret for credentials—no separate "SSH integration" to install or manage.

---

## 2. Background and Motivation

### 2.1 Current State: SSH Integration

Today, SSH is implemented as an **integration**:

- **Location:** `pkg/integrations/ssh/`
- **Registration:** `registry.RegisterIntegration("ssh", &SSH{})`
- **Configuration:** Host, port, username, private key, and optional passphrase are stored as **integration configuration** (encrypted per installation via `IntegrationContext.SetSecret` / `GetConfig`).
- **Components:** Three actions:
  1. **Execute Command** – run a single command
  2. **Execute Script** – run a multi-line script with interpreter
  3. **Host Metadata** – retrieve hostname, OS, kernel, disk, memory, etc.

Execution flow is integration-centric: user installs the SSH integration, configures one set of credentials per installation, then selects a "host" resource (e.g. `user@host:port`) from the integration when configuring a component. Credentials are read from `ctx.Integration.GetConfig("privateKey")` and `GetConfig("passphrase")` at runtime.

### 2.2 Why Move to a Core Component?

- **Consistency with other generic actions:** HTTP Request, Filter, Wait, etc. are core components. SSH is a generic "run something on a server" primitive, not a third-party SaaS product; it fits the core component model.
- **Simpler mental model:** No "SSH integration" to install. One node type: "SSH Command," configured with connection + command + secret reference.
- **Better secret hygiene:** Credentials live in organization Secrets (central store, audit, rotation) instead of integration config. Aligns with the direction of using secrets for sensitive values.
- **Reduced scope:** A single "run command" action covers the majority of use cases; script execution can be achieved by running a single command that invokes an interpreter (e.g. `bash -c '...'`). Host metadata is a niche feature and can be deprecation/out-of-scope for v1 of the core component.

### 2.3 Out of Scope for This PDR

- **Execute Script** and **Host Metadata** as separate actions are **not** in scope for the initial core component. They may be revisited later or left as advanced patterns (e.g. "run `bash -c '...'`" for scripts).
- **Migration path** for existing workflows using the SSH integration (e.g. automatic migration, or run both integration and core component in parallel) is a separate concern; this PDR focuses on the design of the new core component only.
- **Known hosts / host key verification** behavior (e.g. strict vs. insecure ignore) is noted in Security but not fully specified here; it can follow existing or updated security guidelines.

---

## 3. Goals and Non-Goals

| Goals | Non-Goals |
|-------|-----------|
| Single core component: "SSH Command" (run one command) | Multiple actions (script, host metadata) in this PDR |
| Authentication **only** via organization Secrets | Storing credentials in node/integration config |
| Support both SSH key and password auth via explicit auth method and secret+key references | Supporting other auth methods (e.g. agent forwarding) in v1 |
| Clear UX: host + command + secret reference | Backward compatibility with SSH integration in this doc |
| Reuse existing SSH client logic where possible | Changing how Secrets or expressions work platform-wide |

---

## 4. Product Requirements

### 4.1 User-Facing Behavior

1. **Component availability**  
   - In the component palette, users see a core component (e.g. "SSH Command" or "Run SSH Command") alongside HTTP Request, Filter, etc.  
   - No separate "SSH" integration to install.

2. **Configuration**  
   - **Connection (required):**
     - **Host** – hostname or IP (string or expression).
     - **Port** – number, default 22.
     - **Username** – string or expression.
   - **Authentication (required):**  
     - **Auth method** – user selects **SSH key** or **Password**. The chosen method determines which fields are shown (see 4.2).
     - For **SSH key:** user specifies which organization Secret and which **key name** within that secret holds the private key (e.g. Secret "prod-ssh", key "private_key"). Optionally, a second secret+key for the passphrase if the key is encrypted.
     - For **Password:** user specifies which organization Secret and which **key name** holds the password (e.g. Secret "prod-ssh", key "password").
   - **Action:**
     - **Command** – the command to run (string, supports expressions).
   - **Optional:**
     - **Working directory** – if set, run `cd <dir> && <command>`.
     - **Timeout (seconds)** – 0 = no timeout.

3. **Authentication: explicit method and secret/key references**  
   - The user **explicitly chooses** the auth method (SSH key or Password). There is no inference from the contents of a secret.
   - For **SSH key:** the user selects a Secret and the key name that contains the private key value; optionally, a Secret and key name for the passphrase. Any secret key names are valid—the user tells the component which key to use.
   - For **Password:** the user selects a Secret and the key name that contains the password. Again, the user specifies which key to use.
   - At runtime the platform resolves the referenced secret(s) and key(s) and passes the resolved values to the component. The component uses them according to the selected auth method only.

4. **Output**  
   - Same as current Execute Command: stdout, stderr, exit code.  
   - **Success** channel when exit code is 0, **Failed** channel when non-zero (per component-design guidelines: "failed" = executed but outcome failure).

5. **Errors**  
   - Connection failures, auth failures, missing/invalid secret, timeout → **error** state with clear, actionable messages (no raw stack traces or internal details).

### 4.2 Auth method and secret/key references

Authentication is **explicit**: the user selects an auth method, then points to the exact secret and key name(s) to use. The component does not inspect or infer anything from the secret’s keys.

| Auth method | User configures | Required |
|-------------|-----------------|----------|
| **SSH key** | Secret + key name for **private key** | Yes |
|             | Secret + key name for **passphrase** (if key is encrypted) | No |
| **Password**| Secret + key name for **password** | Yes |

- **Auth method** is a single choice: "SSH key" or "Password". Only the fields relevant to that method are shown (e.g. when "Password" is selected, private key / passphrase fields are hidden).
- **Secret** is selected from the organization’s Secrets (by name or ID).
- **Key name** is the name of the key within that secret whose value should be used (e.g. `private_key`, `password`, or any user-defined key name). The UI can offer a dropdown of the secret’s keys when a secret is selected, or a free-form key name.
- The same secret can be used for multiple keys (e.g. one secret "prod-ssh" with keys `private_key` and `passphrase`), or different secrets per value; the user explicitly binds each value to a secret+key.

---

## 5. Technical Design

### 5.1 Component Identity and Registration

- **Name:** `ssh` (core component name; node type could be "ssh" or "ssh.command" for consistency with other core components that use a single name).
- **Registration:** `registry.RegisterComponent("ssh", &SSHCommand{})` in `pkg/components/ssh/` (no integration registration).
- **Label / UX:** "SSH Command" or "Run SSH Command."
- **Icon:** Lucide icon (e.g. `terminal` or `server`) per component-design guidelines for core components.

### 5.2 File and Package Layout

- **Package:** `pkg/components/ssh/`
- **Files (suggested):**
  - `ssh.go` – component definition, `Name()`, `Label()`, `Description()`, `Documentation()`, `Configuration()`, `OutputChannels()`, `Setup()`, `Execute()`, etc.
  - `client.go` – SSH client (connect, run command). Can be adapted from `pkg/integrations/ssh/client.go` with auth abstracted (signer vs. password).
  - `auth.go` – resolve auth from configuration: given resolved config (with secret reference resolved to key-value map), build `ssh.AuthMethod` (PublicKeys or Password).
  - `example.go` / `example_output.json` – for docs and tests.
  - `*_test.go` – unit tests.

The existing integration’s `Client` uses `PrivateKey` and `Passphrase` bytes. The core component will obtain these from the **resolved configuration** (secret values injected by the platform when resolving expressions or secret references).

### 5.3 Configuration Schema (Backend)

**Spec struct (strongly typed, per component-implementations):**

```go
type Spec struct {
    Host             string `json:"host"`
    Port             int    `json:"port"`
    Username         string `json:"username"`
    AuthMethod       string `json:"authMethod"`       // "ssh_key" | "password"
    // For SSH key auth (resolved at runtime from secret+key):
    PrivateKeySecretRef string `json:"privateKeySecretRef,omitempty"`
    PrivateKeyKeyName  string `json:"privateKeyKeyName,omitempty"`
    PassphraseSecretRef string `json:"passphraseSecretRef,omitempty"`
    PassphraseKeyName  string `json:"passphraseKeyName,omitempty"`
    // For password auth:
    PasswordSecretRef string `json:"passwordSecretRef,omitempty"`
    PasswordKeyName  string `json:"passwordKeyName,omitempty"`
    Command          string `json:"command"`
    WorkingDirectory string `json:"workingDirectory,omitempty"`
    Timeout          int    `json:"timeout,omitempty"`
}
```

**Authentication:** The user chooses `authMethod`; the UI shows only the fields for that method. The platform resolves the relevant secret ref + key name at runtime to the actual value(s) and passes them in the resolved configuration (e.g. `ResolvedPrivateKey`, `ResolvedPassphrase`, or `ResolvedPassword`). The component never sees raw config for the other method and does not infer auth from secret contents.

**Configuration fields** exposed to the UI:

- `host` – string (or expression).
- `port` – number, default 22.
- `username` – string (or expression).
- `authMethod` – select: "SSH key" or "Password". Visibility conditions show/hide the auth fields below.
- For **SSH key:** `privateKeySecretRef`, `privateKeyKeyName`; optionally `passphraseSecretRef`, `passphraseKeyName`.
- For **Password:** `passwordSecretRef`, `passwordKeyName`.
- `command` – string (expression allowed).
- `workingDirectory` – optional string.
- `timeout` – optional number.

Sensitive data (private key, passphrase, password) **must not** be stored in the node configuration; they are only ever read from the resolved secret values at execution time.

### 5.4 Configuration Resolution and Secrets at Runtime

Today, **core components** receive `Configuration` that has been resolved by `NodeConfigurationBuilder.Build()` using the **schema** of the component’s `Configuration()` fields. Expression strings (`{{ ... }}`) are evaluated with an env that includes `$` (message chain), `root()`, `previous()`, etc. **Organization Secrets are not currently injected into that expression environment.**

To support "use secrets for authentication" we need one of:

- **Option A – Secret reference field type:**  
  - A new field type (e.g. `secret` or `secretRef`) where the user selects an organization Secret (by name or ID).  
  - At execution time, the builder (or executor) resolves this field specially: load the secret by ID/name, decrypt, and pass the **key-value map** into the configuration under a key like `secret` or `resolvedSecret`.  
  - The component then reads `resolvedSecret["private_key"]`, `resolvedSecret["password"]`, etc.  
  - No expressions needed for the secret content; only the reference is stored.

- **Option B – Expressions that reference secrets:**  
  - Extend the expression language with a function or variable, e.g. `secrets("MySecret").private_key` or `secrets.MySecret.private_key`, so that when the expression is evaluated, the runtime fetches the org secret, decrypts, and returns the value (or struct).  
  - User would put in the "Authentication" field something like `{{ secrets("prod-ssh").private_key }}` for key or `{{ secrets("prod-ssh").password }}` for password.  
  - Then the component would need **two** fields (e.g. "Private key (from secret)" and "Password (from secret)") or one expression that points to the whole secret object.  
  - Requires expression env to have access to org secrets (scoped to workflow’s organization).

**Recommendation:** **Option A** is clearer for "select a secret" and keeps secret content out of the expression string. Option B is more flexible for mixing secret values with other expressions. The PDR assumes **Option A** for the SSH component: a dedicated **Secret** (or secret reference) field, resolved by the platform to a key-value map, with fixed key names (`private_key`, `passphrase`, `password`) for auth. If the platform later adds `secrets()` in expressions, the component could still use Option A for simplicity.

**Implementation note:** Where exactly "resolve secret ref + key name → value" happens is a platform concern. The SSH component’s contract is: at `Execute()`, `ctx.Configuration` (after resolution) contains the resolved credential value(s) for the chosen auth method only—e.g. `ResolvedPrivateKey` and optionally `ResolvedPassphrase` for SSH key, or `ResolvedPassword` for password. The component does not receive a full secret map or infer auth from key names.

### 5.5 Execute Flow

1. Decode `ctx.Configuration` into `Spec`. The platform has already resolved secret refs into the actual values (e.g. `ResolvedPrivateKey`, `ResolvedPassphrase`, or `ResolvedPassword`) according to the chosen `authMethod`; the component does not resolve secrets itself.
2. **Build auth:** From `Spec.AuthMethod` and the resolved value(s), build `ssh.AuthMethod`—`ssh.PublicKeys(signer)` for SSH key, `ssh.Password(...)` for password (see Section 6).
3. **Validate:** Host, username, command non-empty; port valid; for the selected auth method, the corresponding resolved value(s) must be present.
4. Build SSH client (reuse or adapt `pkg/integrations/ssh/client.go`), connect, run command (with optional working directory and timeout).
5. Map result to metadata (stdout, stderr, exit code), set execution metadata, emit to **success** or **failed** channel; on connection/execution error, call `ctx.ExecutionState.Fail(..., reasonError, message)`.

**Important:** Core components do **not** have `ctx.Integration`; it is nil. So no `GetConfig("privateKey")`. All credential data must come from the resolved configuration (secret) provided by the platform.

### 5.6 Output Channels and Payload

- **Channels:** `success` (exit code 0), `failed` (exit code != 0), plus implicit **error** state for execution failures (see component-design).
- **Payload:** Same as current Execute Command, e.g.:

```json
{
  "result": {
    "stdout": "...",
    "stderr": "...",
    "exitCode": 0
  }
}
```

Event type e.g. `ssh.command.executed` / `ssh.command.failed`. Structure should remain expression-friendly (flat, ≤3 levels for `$['Node Name'].result.stdout`).

### 5.7 Host Key Verification

Current integration uses `ssh.InsecureIgnoreHostKey()`. The PDR does not mandate a specific policy; the implementation can keep this for v1 and add a configuration option (e.g. "Strict host key checking" with optional `known_hosts` from a secret or file) in a later iteration. Security section below calls this out.

---

## 6. Authentication Modes in Detail

Auth is driven **only** by the user-selected `authMethod` and the resolved value(s) for that method. No inference from secret keys.

### 6.1 SSH Key

- **Configured by user:** Secret + key name for private key; optionally secret + key name for passphrase.
- **At Execute():** Component receives resolved private key value (and optionally passphrase). No key-name logic in the component.
- **Private key format:** PEM or OpenSSH (existing `normalizePrivateKey` and `parseSigner` in `pkg/integrations/ssh/client.go` can be reused). Base64-wrapped keys can be supported as today.
- **Passphrase:** Only used when the user configured passphrase secret+key and the platform resolved a non-empty value; otherwise use empty passphrase for parsing the key.
- **Auth method:** `ssh.PublicKeys(signer)`.

### 6.2 Password

- **Configured by user:** Secret + key name for password.
- **At Execute():** Component receives resolved password value.
- **Auth method:** `ssh.Password(password)`.

### 6.3 Validation

- **Setup():** For the selected auth method, the corresponding secret ref(s) and key name(s) must be present. Optionally validate that the secret exists (e.g. via platform API).
- **Execute():** After the platform has resolved secrets, if the resolved value for the chosen auth method is missing or empty, fail with a clear error: e.g. "Private key value could not be resolved from the selected secret and key." or "Password could not be resolved from the selected secret and key."

---

## 7. Security Considerations

- **Secrets storage:** Private keys and passwords are stored only in organization Secrets (encrypted at rest, access-controlled). Node configuration must never persist raw key or password values; only a reference (and optionally resolved at runtime in memory).
- **Resolution scope:** Secret resolution must be scoped to the workflow’s organization so that one org cannot access another org’s secrets.
- **Logging and errors:** Do not log or expose secret content (key material, passphrase, password) in error messages or logs. Use generic messages (e.g. "SSH authentication failed").
- **Host key verification:** Current integration disables host key checking. For production, consider offering an option to enable it and/or supply known_hosts (e.g. from a secret key) in a future revision.
- **Network:** SSH runs over the network; consider timeouts and resource limits (already in current design via timeout).

---

## 8. User Experience and Documentation

### 8.1 In-UI Guidance

- **Setup instructions:** Short copy in the component’s description or docs: "Choose SSH key or Password auth, then select the organization Secret and the key name within that secret that holds the credential (e.g. private key or password)."
- **Placeholders:** Host "e.g. example.com or 192.168.1.100", Username "e.g. root, ubuntu", Command "e.g. ls -la /tmp".
- **Main content (node card):** Per component-design, show 0–3 items. Suggested: host (or truncated), command (truncated), e.g. "prod-server · ls -la".

### 8.2 Documentation

- **Core.mdx:** Add "SSH Command" under Actions with a short description, link to a section that explains:
  - Configuration (host, port, username, secret, command, working directory, timeout).
  - How to choose auth method and bind secret+key for SSH key or password.
  - Output (success/failed, result shape).
  - Example payload.
- **Contributing:** Reference this PDR from the component-implementations or integration docs where SSH is mentioned.

---

## 9. Testing Strategy

- **Unit tests:**  
  - Auth resolution: given a map with `private_key` / `passphrase` or `password`, correct `ssh.AuthMethod` is built.  
  - Validation: missing host, missing auth, invalid port, etc.  
  - Command building (with working directory, timeout).  
  - Reuse existing SSH client tests where possible (e.g. `execute_command_test.go`, `client_test.go`) and adapt to the new auth source (secret map instead of integration config).
- **Integration/E2E:**  
  - Run a real SSH command against a test host using a secret that contains key or password (in a safe test environment).  
  - Verify success/failed channels and payload shape.

---

## 10. Migration and Deprecation (Brief)

- This PDR does **not** define the migration path for the existing SSH integration. Options include: (1) keep both integration and core component indefinitely; (2) deprecate the integration and provide a migration path (e.g. "Replace with SSH Command node and create a secret from your integration credentials"); (3) auto-migrate workflows. To be decided in a separate migration plan.
- If the integration is deprecated, its three components (Execute Command, Execute Script, Host Metadata) would be removed or redirected; Execute Command’s behavior is the one that the new core component replaces.

---

## 11. Open Questions and Future Work

1. **Secret + key reference fields:** Does the platform support selecting an organization Secret and a key name within it (e.g. a "secret reference" field that stores secret ID/name + key name and is resolved to the single value at runtime)? The SSH component needs that for private key, passphrase, and password. If not, this PDR assumes such a mechanism will be added.
2. **Key name input:** UI can show a dropdown of the secret’s keys when a secret is selected, or a free-form key name; product preference TBD.
3. **Execute Script:** Later enhancement could add "Run script" (multi-line) as a second action or as a single command that runs `bash -c '...'` with script in an expression.
4. **Host key verification:** When to add strict checking and how to supply known_hosts (e.g. another secret key).

---

## 12. Summary Checklist

- [ ] SSH is a **core component** registered as `ssh` in `pkg/components/ssh/`.
- [ ] **Single action:** Run one SSH command (host, port, username, command, optional working dir, optional timeout).
- [ ] **Authentication** is **only** via organization Secrets; no integration config for credentials.
- [ ] User **explicitly selects** auth method (SSH key or Password) and, for that method, the Secret and key name(s) to use; no inference from secret contents.
- [ ] Configuration includes **auth method** and **secret+key references** for the chosen method; platform resolves them at runtime; component never stores raw credentials.
- [ ] Output: **success** / **failed** channels and **error** state; payload with stdout, stderr, exit code.
- [ ] UX and docs explain how to create and select a secret for SSH.
- [ ] Security: no logging of secrets; resolution scoped to org; host key policy documented or deferred.
- [ ] Execute Script and Host Metadata are out of scope for this PDR; migration of existing SSH integration is a separate concern.

---

*End of PDR.*
