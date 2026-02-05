# PDR: Azure Integration (Base + Create Virtual Machine Component)

## Product Design Document

**Status:** Draft  
**Scope:** Azure integration base (identity, credentials, metadata) and a single component: **Create Virtual Machine** (compute).  
**Security:** First-class; no long-lived secrets, least privilege, and alignment with Azure security best practices.

---

## 1. Executive Summary

This document describes how to implement an Azure integration in SuperPlane that mirrors the security and architectural patterns of the existing AWS integration. The scope is limited to:

1. **Base integration** — How the Azure integration is registered, configured, and how it obtains and refreshes credentials without storing long-lived secrets.
2. **One component** — **Create Virtual Machine**, which provisions an Azure VM (compute resource) from a workflow, analogous to what "Run EC2 instance" would be for AWS.

The design follows AWS patterns: OIDC-based federation, short-lived tokens only, encrypted storage of runtime secrets, and minimal IAM. Triggers (e.g. Event Grid) and other components are out of scope for this PDR but are outlined so the base can support them later.

---

## 2. Reference: AWS Implementation (Summary)

### 2.1 Base (pkg/integrations/aws/)

- **Configuration (no secrets in config):** `region`, `sessionDurationSeconds`, `roleArn`, `tags`. No access keys.
- **Credentials:** SuperPlane OIDC token → AWS STS `AssumeRoleWithWebIdentity` → temporary AWS credentials (access key, secret key, session token). Stored as integration **secrets** (encrypted at rest). Refreshed at half of remaining TTL via `ScheduleResync`.
- **Browser action:** When `roleArn` is empty, shows instructions: create OIDC IdP in IAM (issuer = SuperPlane base URL, audience = integration ID), create IAM role with "Web identity" trust, attach needed policies, paste role ARN.
- **Sync flow:** Decode config → if no role ARN → show browser action; else → derive account ID from role ARN → `generateCredentials` (OIDC sign → STS assume) → store credentials as secrets, set session metadata, schedule resync → `configureRole` (IAM role for EventBridge) → `configureEventBridge` (connection + API destination, with a **separate** connection secret for webhook auth).
- **Cleanup:** Remove EventBridge rules/targets, API destinations, connections; remove IAM role policy and role.
- **HTTP handler:** `/events` validated by header `X-Superplane-Secret` against the EventBridge connection secret; then events dispatched to subscriptions.
- **Secrets:** (1) STS credentials: `accessKeyId`, `secretAccessKey`, `sessionToken`. (2) EventBridge: `eventbridge.connection.secret` (random 32-byte base64). No long-lived AWS keys anywhere.

### 2.2 Component Example: Lambda Run Function

- **Config:** `functionArn` (integration resource `lambda.function`), `payload` (object).
- **Setup:** Validate `functionArn`, store in node metadata.
- **Execute:** Load credentials from integration, resolve region (config or from ARN), build Lambda client (Sig V4), invoke, emit output (requestId, report, payload).
- **ListResources:** For `lambda.function`, call Lambda ListFunctions, return `IntegrationResource{Type, Name, ID=ARN}`.
- **Client:** Uses AWS credentials and Sig V4 for each request; no credentials in component config.

### 2.3 Security Patterns (to replicate for Azure)

- No long-lived cloud credentials; OIDC + token exchange only.
- Session/token lifetime bounded and configurable; refresh before expiry.
- All runtime secrets in integration secrets (encrypted by SuperPlane).
- Webhook/event ingestion protected by a dedicated secret (header check).
- Service-side IAM (e.g. EventBridge invoker role) uses minimal, scoped policies.
- User-supplied IAM (role ARN / Azure app) follows least privilege; docs guide scoping.

---

## 3. Azure Equivalents (High Level)

| Concept | AWS | Azure |
|--------|-----|--------|
| Identity / auth | IAM OIDC IdP + AssumeRoleWithWebIdentity | Azure AD (Entra ID) App Registration + Federated Credential (OIDC) |
| Temporary credentials | STS temporary credentials | Azure AD OAuth2 token (e.g. client credentials or OIDC token exchange) |
| Global/regional config | Region, role ARN | Tenant ID, subscription ID, (optionally) default location/region |
| Event ingestion | EventBridge API destination + connection | Event Grid (webhook subscription + validation) |
| Compute "create machine" | (N/A in current codebase; would be EC2 RunInstances) | ARM: Create or start a Virtual Machine (Microsoft.Compute/virtualMachines) |
| Listing compute | (e.g. Lambda list) | ARM: List VMs in resource group or subscription |

The Azure base must support: **OIDC federation → Azure AD token → use token for ARM (and optionally Microsoft Graph / Event Grid)**. The Create VM component will use ARM only.

---

## 4. Azure Base Integration

### 4.1 Goals

- **No long-lived Azure secrets** in config or in storage. Optional: allow client secret as fallback for environments where federated credential cannot be used; if supported, treat as secret and store encrypted, with clear deprecation path.
- **Primary path:** Azure AD App Registration with **Federated Credential** (OIDC) pointing at SuperPlane's OIDC issuer; audience = integration ID. SuperPlane signs a JWT; Azure AD validates and issues an access token for Azure APIs.
- **Short-lived tokens:** Request tokens with limited lifetime; refresh before expiry (e.g. at half-life) via `ScheduleResync`.
- **Secrets storage:** Store only tokens (or client secret if ever used) in integration secrets (encrypted at rest). No secrets in integration configuration (config is non-sensitive).

### 4.2 Configuration Schema (Non-Secret)

Proposed fields (aligned with AWS style):

| Field | Type | Required | Default | Description |
|-------|------|----------|--------|-------------|
| `tenantId` | string | yes | — | Azure AD tenant ID (directory ID). |
| `subscriptionId` | string | yes | — | Azure subscription ID for ARM (and billing). |
| `clientId` | string | yes | — | Application (client) ID of the App Registration. |
| `location` | string | no | e.g. `eastus` | Default location/region for resources (ARM "location"). |
| `tokenValidityMinutes` | number | no | 60 | Requested token lifetime in minutes (capped by Azure policy, e.g. 60–120). |
| `tags` | list of {key, value} | no | — | Tags to apply to Azure resources created by this integration. |

**Explicitly not in config:** client secret, certificates, or any long-lived secret. Those (if ever supported) would be stored via integration secrets only.

### 4.3 Identity and Credential Flow

**Recommended: OIDC Federated Credential (passwordless)**

1. **Customer setup (browser action when `clientId` or federated credential is missing):**
   - Create an **App Registration** in Azure AD (single tenant or multitenant as needed).
   - Under **Certificates & secrets:** do **not** create a client secret (goal: passwordless).
   - Under **Federated credentials:** add an **OIDC** credential:
     - **Issuer:** SuperPlane's OIDC issuer URL (e.g. `https://<superplane-base>/oidc` or the same base used for AWS).
     - **Subject:** Fixed pattern that includes the integration ID, e.g. `app-installation:<integration-id>` (must match what SuperPlane signs in the JWT `sub` claim).
     - **Audience:** Integration ID (UUID) so each installation has a distinct audience.
   - Grant the app the **minimum API permissions** required:
     - For ARM (Create VM, list VMs, etc.): `Microsoft.Compute/*` (or narrower, e.g. read/write Virtual Machines) scoped to the subscription or resource groups.
     - For Event Grid (future): appropriate Event Grid scope.
   - Admin consents if required.
   - Customer pastes **Tenant ID**, **Subscription ID**, and **Application (client) ID** into the integration form.

2. **SuperPlane Sync (each sync or resync):**
   - Build a JWT signed by SuperPlane's OIDC key:
     - `iss`: SuperPlane issuer
     - `sub`: `app-installation:<integration-id>`
     - `aud`: integration ID (UUID)
     - `iat` / `nbf` / `exp`: short window (e.g. 5 minutes for the request).
   - Call Azure AD **OAuth2 token endpoint** with grant type **client_credentials** and:
     - `client_id`, `tenant_id`
     - `client_assertion_type`: `urn:ietf:params:oauth:client-assertion-type:jwt-bearer`
     - `client_assertion`: the signed JWT
     - `scope`: e.g. `https://management.azure.com/.default`
   - Azure AD validates the federated credential (issuer, subject, audience) and returns an **access_token** (and optionally refresh_token depending on flow; for client_credentials typically no refresh).
   - Store in integration **secrets**: e.g. `accessToken` (and if applicable `expiresOn` as a string). Do **not** store in configuration.
   - Set integration **metadata**: e.g. `session` with `tenantId`, `subscriptionId`, `clientId`, `expiresAt` (token expiry).
   - Call `ScheduleResync(refreshAfter)` so that refresh runs at half of remaining TTL (similar to AWS), ensuring tokens are never used near expiry.

3. **Component execution (e.g. Create VM):**
   - Load credentials from integration secrets (`accessToken`).
   - If token is missing or expired, fail fast and surface "integration needs resync" (or trigger resync).
   - Use the access token as Bearer token for ARM API requests.

**Security notes:**

- Token lifetime should be kept short (e.g. 60 minutes) and refreshed proactively.
- Federated credential subject must be stable and unique per integration (e.g. `app-installation:<id>`).
- No client secret means no secret rotation in Azure AD for this app; compromise of SuperPlane's OIDC key is the main concern (same as AWS).

### 4.4 Metadata and Secrets (Storage)

**Metadata (integration metadata, non-secret):**

- `session`: `tenantId`, `subscriptionId`, `clientId`, `expiresAt` (ISO8601), optional `location`.
- `tags`: normalized list of `{key, value}` for tagging resources.
- Future: `eventGrid` (or similar) for webhook subscriptions; leave structure undefined for this PDR.

**Secrets (integration secrets, encrypted at rest):**

- `accessToken`: current Azure AD access token for `https://management.azure.com/.default` (or the scopes actually used).
- Optional future: `eventgrid.webhook.secret` for Event Grid delivery validation (constant-time compare).

Never store tokens or secrets in configuration or in logs.

### 4.5 Sync Flow (Detailed)

1. Decode configuration; validate required fields (`tenantId`, `subscriptionId`, `clientId`). If missing, show **browser action** with step-by-step instructions (create App Registration, add federated credential, grant permissions, paste IDs).
2. Normalize tags; store in metadata.
3. **Obtain token:** Sign OIDC JWT (subject `app-installation:<integration-id>`, audience = integration ID); POST to `https://login.microsoftonline.com/<tenantId>/oauth2/v2.0/token` with `client_credentials` and `client_assertion`; parse `access_token` and `expires_in`.
4. Store `accessToken` in secrets; set `session` in metadata with `expiresAt` = now + expires_in.
5. Schedule resync at half of remaining TTL (e.g. `expires_in/2` seconds), with a minimum (e.g. 1 minute).
6. Mark integration ready; remove browser action.

No Event Grid or IAM provisioning in this PDR; keep Sync limited to credential + metadata.

### 4.6 Cleanup

- On integration deletion: clear metadata and secrets (handled by platform). No need to call Azure to "delete" the app or federated credential; customer manages that. Optionally: document that deleting the integration does not remove the App Registration, so customers can revoke access by deleting the federated credential or the app.

### 4.7 HTTP Handler (Future: Event Grid)

- For Event Grid webhook delivery, Azure sends a validation request (with a code) and later events with a secret in a header (e.g. `aeg-event-type`). Validation: respond with the code. Event delivery: verify header (e.g. `aeg-signature` or custom header with shared secret) using constant-time compare. Store the webhook secret in integration secrets (`eventgrid.webhook.secret`). Out of scope for initial base; design the base so this can be added without changing credential flow.

### 4.8 Error Handling and Resilience

- **Azure AD errors:** Map HTTP 4xx/5xx and `error` / `error_description` from token endpoint to a stable error type; do not log tokens or assertions. Retry with backoff for 5xx and network errors.
- **Token expiry:** If a component gets 401 from ARM, fail the execution and optionally trigger a resync so the next run gets a fresh token.
- Use a small **common** package (e.g. `pkg/integrations/azure/common`) for: credential loading from integration, token expiry check, normalized tags, and error type (e.g. `common.Error`) for idempotency (e.g. AlreadyExists, NotFound) to align with AWS patterns.

### 4.9 File Layout (Base)

- `pkg/integrations/azure/azure.go` — Integration struct, Configuration, Name/Label/Icon/Description/Instructions, Configuration(), Components(), Triggers(), Sync(), Cleanup(), HandleRequest(), CompareWebhookConfig(), SetupWebhook(), CleanupWebhook(), ListResources(), Actions(), HandleAction(). Register in `init()` with `registry.RegisterIntegration("azure", &Azure{})`.
- `pkg/integrations/azure/token.go` — Obtain token: build JWT (use existing OIDC provider), POST to Azure AD token endpoint, return access token + expiry. No storage here; caller (Sync) stores in secrets.
- `pkg/integrations/azure/common/common.go` — IntegrationMetadata (Session, Tags, optional future EventGrid), CredentialsFromInstallation (read accessToken from secrets), LocationFromInstallation, NormalizeTags, optional SubscriptionIDFromInstallation.
- `pkg/integrations/azure/common/error.go` — Parse Azure ARM/Azure AD error payloads; helpers like IsAlreadyExistsErr, IsNotFoundErr (e.g. from `error.code` or `Code`).
- `pkg/server/server.go` — Add blank import: `_ "github.com/superplanehq/superplane/pkg/integrations/azure"`.

Dependencies: use standard library HTTP + JWT signing (existing OIDC); no need to ship an Azure SDK in the first iteration if a minimal REST client for token + ARM is sufficient (reduces supply-chain and keeps control over signing and headers).

---

## 5. Component: Create Virtual Machine

### 5.1 Purpose

- Allow a workflow to create a single Azure Virtual Machine (compute resource). This is the "one component" in scope: the Azure analogue of "create a compute instance" (e.g. EC2 RunInstances).

### 5.2 Component Contract

- **Name:** `azure.compute.createVirtualMachine` (or `azure.compute.createVM`).
- **Label:** "Azure Compute • Create Virtual Machine".
- **Description:** "Create an Azure Virtual Machine in the specified resource group with the given size, image, and optional network/disk configuration."
- **Output:** Single default output channel; payload includes at least: VM resource ID, name, provisioning state, and optionally location, size, OS type.

### 5.3 Configuration Fields

All fields should be strongly typed in a spec struct and decoded with `mapstructure` (see component-implementations.md). Proposed fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `resourceGroup` | string | yes | Name of the resource group (must exist). |
| `name` | string | yes | VM name (Azure naming rules). |
| `location` | string | no | Override integration default location. |
| `size` | string | yes | VM size (e.g. `Standard_B2s`, `Standard_D2s_v3`). |
| `image` | object | yes | OS image: `publisher`, `offer`, `sku`, `version` (e.g. Canonical, 0001-com-ubuntu-server-jammy, 22_04-lts, latest). |
| `adminUsername` | string | yes | Admin user name. |
| `adminPassword` | string or secret ref | no* | Password (sensitive). Prefer reference to SuperPlane secret; if inline, mark sensitive and never log. |
| `sshPublicKey` | string | no | SSH public key (alternative to password for Linux). |
| `osDisk` | object | no | Size (GB), type (e.g. Premium_LRS), optional caching. |
| `tags` | list of {key, value} | no | Tags for the VM (merged with integration tags if desired). |
| `networkInterface` | object | no | Existing subnet ID / NIC, or "create new" with VNet/subnet params; keep minimal for v1 (e.g. subnet resource ID). |

*Either `adminPassword` or `sshPublicKey` required for Linux; for Windows, password typically required. Validation in Setup().

### 5.4 Setup()

- Decode configuration; validate required fields and naming rules (length, allowed characters).
- Validate image and size (e.g. non-empty; optional: allowlist of known sizes to avoid silent ARM errors).
- Store in node metadata only what is needed at execution (e.g. resource group, name, location, size, image, admin credentials reference, OS type). Do **not** store raw password in metadata if it can be avoided; pass through at Execute from config or from secrets resolution.

### 5.5 Execute()

1. Load Azure credentials from integration (common.CredentialsFromInstallation → access token). If missing or expired, fail with a clear "integration sync required" message.
2. Resolve location (node config override or integration default).
3. Build ARM request body for **Create or Update Virtual Machine** (PUT `https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{name}?api-version=...`). Use a stable API version (e.g. 2024-07-01 or current).
4. Map configuration to ARM JSON: properties (hardwareProfile, storageProfile, osProfile, networkProfile), tags. Ensure admin password is taken from secrets or expression-resolved secret reference, not logged.
5. Send PUT with Bearer token; handle 200/201 and error responses.
6. Parse response: id, name, provisioningState, location, etc. Emit to default output channel (e.g. payload type `azure.compute.vmCreated` with body containing resource ID, name, provisioningState, location, size).

### 5.6 ListResources

- **Resource type:** `compute.virtualMachine` (or `compute.vm`).
- **Implementation:** Call ARM List VMs (e.g. by resource group or by subscription). Endpoint: `GET .../resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines?api-version=...` or subscription-scoped list. Return `[]core.IntegrationResource` with Type, Name, ID (ARM resource ID). Support either "list by resource group" (if integration has a default resource group) or "list by subscription" with optional filter; document that listing may be large and consider pagination (ARM uses nextLink).

### 5.7 Client and API

- **Client:** `pkg/integrations/azure/compute/client.go` (or under `compute/`). NewClient(http, credentials, subscriptionID, baseURL). Methods: CreateOrUpdateVM(ctx, resourceGroup, name, params), ListVMs(ctx, resourceGroup or subscriptionID, options). Use Bearer token for Authorization header; no Sig V4.
- **API version:** Pin one ARM API version for Microsoft.Compute (e.g. 2024-07-01) for predictability.
- **Errors:** Parse ARM error body (`error.code`, `error.message`); map to common.Error; use IsAlreadyExistsErr / IsNotFoundErr where applicable so callers can react (e.g. idempotent create).

### 5.8 Security (Component)

- **Secrets:** Admin password must not appear in logs, traces, or non-secret metadata. Use SuperPlane secrets or expression that resolves to a secret; component receives value in execution context without logging.
- **Least privilege:** Documentation should state that the Azure app only needs Compute (and if creating NICs, Network) permissions scoped to the intended resource groups or subscription.
- **Idempotency:** Create VM is PUT create/update; same name + resource group is idempotent. Document behavior when VM already exists (update vs error) per ARM semantics.
- **Input validation:** Reject obviously invalid sizes or image refs; validate VM name and resource group name against Azure rules to avoid confusing ARM errors.

### 5.9 Example Output and Tests

- **Example output JSON:** Include `id`, `name`, `provisioningState`, `location`, `vmSize` in example_output_create_virtual_machine.json.
- **Unit tests:** Setup() with valid/invalid config; Execute() with mocked HTTP (success 201, error 4xx, conflict). Use mapstructure and common.CredentialsFromInstallation patterns from AWS Lambda component.

### 5.10 File Layout (Component)

- `pkg/integrations/azure/compute/create_virtual_machine.go` — Component struct, Name/Label/Description/Documentation/Icon/Color, Configuration(), OutputChannels(), Setup(), Execute(), ProcessQueueItem(), Cancel(), Cleanup(), ExampleOutput(), Actions(), HandleAction(), HandleWebhook().
- `pkg/integrations/azure/compute/client.go` — ARM client: CreateOrUpdateVM, ListVMs (and any shared helpers).
- `pkg/integrations/azure/compute/resources.go` — ListVirtualMachines(ctx, resourceType) for integration ListResources.
- `pkg/integrations/azure/compute/example_output_create_virtual_machine.json` — Example output payload.
- Tests: create_virtual_machine_test.go, client_test.go with mocked HTTP.

---

## 6. Security Best Practices (Checklist)

- **No long-lived secrets in config:** Only tenant ID, subscription ID, client ID, location, token validity, tags. No client secret or certificate in config.
- **Federated credential (OIDC):** Primary auth path; subject and audience bound to integration ID; issuer under SuperPlane control.
- **Short-lived tokens:** Request and refresh with half-life; store only in integration secrets (encrypted).
- **Secrets storage:** All tokens and any future webhook secrets in integration secrets only; never in configuration or logs.
- **Webhook validation (future):** Constant-time comparison for Event Grid (or any) webhook secret.
- **Least privilege:** Document minimal Azure AD app permissions (e.g. Compute read/write for specific scope); avoid subscription-wide Owner.
- **Component secrets:** Passwords and keys via secret references or resolved values; never log or emit in payloads.
- **Error handling:** No token or password in error messages or logs; use stable error codes for idempotency and debugging.
- **HTTPS only:** All calls to login.microsoftonline.com and management.azure.com over TLS; no custom endpoints without explicit justification.
- **Dependencies:** Prefer minimal dependencies (net/http, JWT); add Azure SDK only if necessary and with version pinning and review.

---

## 7. Out of Scope (This PDR)

- Event Grid triggers (design base so they can be added later).
- Other components (e.g. scale set, blob, AKS).
- Azure Government / sovereign clouds (endpoint and tenant differences; can be a follow-up).
- Managed identity for SuperPlane itself (this PDR is about per-integration Azure AD app + federated credential).
- UI/UX details (exact copy and form layout); only configuration schema and behavior are specified.

---

## 8. Acceptance Criteria (Summary)

**Base integration**

- [ ] Integration registers as `azure`; configuration schema as above; no secrets in config.
- [ ] Sync uses OIDC JWT + Azure AD token endpoint; stores access token in secrets; sets session metadata; schedules resync at half TTL.
- [ ] Browser action guides user to create App Registration, federated credential (OIDC), and paste tenant/subscription/client IDs.
- [ ] Cleanup clears state; no long-lived Azure secrets left in SuperPlane.
- [ ] Common package provides CredentialsFromInstallation, error parsing, NormalizeTags.

**Create Virtual Machine component**

- [ ] Component creates a VM via ARM with configurable resource group, name, location, size, image, admin credentials (or SSH key), and optional disk/network.
- [ ] Admin password handled as secret (no logs); output includes resource ID, name, provisioning state.
- [ ] ListResources for `compute.virtualMachine` returns VMs (by resource group or subscription) as IntegrationResource list.
- [ ] Unit tests for Setup and Execute (with mocked HTTP); example output JSON included.

**Security**

- [ ] No client secret or long-lived credential in config or in code paths by default.
- [ ] Tokens only in integration secrets; refresh before expiry.
- [ ] Webhook path (if stubbed) designed for future constant-time secret check.

---

## 9. References

- AWS integration: `pkg/integrations/aws/aws.go`, `sts.go`, `common/`, `lambda/run_function.go`, `lambda/client.go`.
- SuperPlane OIDC: `pkg/oidc/provider.go` (Sign with subject, audience, duration).
- Core interfaces: `pkg/core/integration.go`, `pkg/core/component.go`.
- Component implementation guide: `docs/contributing/component-implementations.md`.
- Azure AD: [Federated credentials](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation), [OAuth2 token endpoint](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-client-creds-grant-flow).
- ARM: [Virtual Machines - Create Or Update](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/create-or-update), [List](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/list).
