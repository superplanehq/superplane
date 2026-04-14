# HashiCorp Vault Base Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the base HashiCorp Vault integration with Token auth and a single "Get Secret" component that reads from KV v2.

**Architecture:** Follows the Perplexity integration pattern exactly — pure REST API, no webhooks, no triggers. The integration registers itself via `init()`, is verified in `Sync()` by calling `/v1/auth/token/lookup-self`, and exposes one component (`getSecret`) that calls `/v1/<mount>/data/<path>`.

**Tech Stack:** Go, `mapstructure` (config decoding), `encoding/json` (response parsing), `github.com/stretchr/testify` (test assertions), `test/support/contexts` (mock HTTP/integration contexts).

**Spec:** `docs/superpowers/specs/2026-04-14-hashicorp-vault-base-design.md`

---

## File Map

| Action | Path |
|--------|------|
| Create | `pkg/integrations/hashicorp_vault/vault.go` |
| Create | `pkg/integrations/hashicorp_vault/client.go` |
| Create | `pkg/integrations/hashicorp_vault/get_secret.go` |
| Create | `pkg/integrations/hashicorp_vault/example.go` |
| Create | `pkg/integrations/hashicorp_vault/example_output_get_secret.json` |
| Create | `pkg/integrations/hashicorp_vault/vault_test.go` |
| Create | `pkg/integrations/hashicorp_vault/get_secret_test.go` |
| Modify | `pkg/server/server.go` |

---

## Task 1: Integration struct, Sync, and HTTP client

**Files:**
- Create: `pkg/integrations/hashicorp_vault/vault_test.go`
- Create: `pkg/integrations/hashicorp_vault/vault.go`
- Create: `pkg/integrations/hashicorp_vault/client.go`

---

- [ ] **Step 1.1: Create `vault_test.go`**

```go
package hashicorp_vault

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSync_Success(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"accessor":"abc123"}}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "https://vault.example.com",
			"token":   "hvs.test",
		},
	}

	v := &HashicorpVault{}
	err := v.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"},
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)
	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/auth/token/lookup-self")
	assert.Equal(t, "hvs.test", httpCtx.Requests[0].Header.Get("X-Vault-Token"))
}

func TestSync_InvalidToken(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"errors":["permission denied"]}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "https://vault.example.com",
			"token":   "bad-token",
		},
	}

	v := &HashicorpVault{}
	err := v.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "bad-token"},
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.NotEqual(t, "ready", integrationCtx.State)
}

func TestSync_MissingToken(t *testing.T) {
	v := &HashicorpVault{}
	err := v.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": ""},
		HTTP:          &contexts.HTTPContext{},
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

func TestSync_MissingBaseURL(t *testing.T) {
	v := &HashicorpVault{}
	err := v.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"baseURL": "", "token": "hvs.test"},
		HTTP:          &contexts.HTTPContext{},
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "baseURL is required")
}

func TestSync_WithNamespace(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL":   "https://vault.example.com",
			"token":     "hvs.test",
			"namespace": "admin/team-a",
		},
	}

	v := &HashicorpVault{}
	err := v.Sync(core.SyncContext{
		Logger: logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{
			"baseURL":   "https://vault.example.com",
			"token":     "hvs.test",
			"namespace": "admin/team-a",
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "admin/team-a", httpCtx.Requests[0].Header.Get("X-Vault-Namespace"))
}
```

---

- [ ] **Step 1.2: Create stub `vault.go` and `client.go` (enough to compile)**

`pkg/integrations/hashicorp_vault/vault.go`:
```go
package hashicorp_vault

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("hashicorp_vault", &HashicorpVault{})
}

type HashicorpVault struct{}

func (v *HashicorpVault) Name() string         { return "hashicorp_vault" }
func (v *HashicorpVault) Label() string        { return "HashiCorp Vault" }
func (v *HashicorpVault) Icon() string         { return "vault" }
func (v *HashicorpVault) Description() string  { return "Securely read secrets from HashiCorp Vault" }
func (v *HashicorpVault) Instructions() string { return "" }

func (v *HashicorpVault) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (v *HashicorpVault) Components() []core.Component  { return []core.Component{} }
func (v *HashicorpVault) Triggers() []core.Trigger      { return []core.Trigger{} }
func (v *HashicorpVault) Actions() []core.Action        { return []core.Action{} }
func (v *HashicorpVault) HandleRequest(ctx core.HTTPRequestContext) {}

func (v *HashicorpVault) Cleanup(ctx core.IntegrationCleanupContext) error   { return nil }
func (v *HashicorpVault) HandleAction(ctx core.IntegrationActionContext) error { return nil }

func (v *HashicorpVault) Sync(ctx core.SyncContext) error { return nil }

func (v *HashicorpVault) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}
```

`pkg/integrations/hashicorp_vault/client.go`:
```go
package hashicorp_vault

import (
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL   string
	Token     string
	Namespace string
	http      core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseURL")
	if err != nil {
		return nil, fmt.Errorf("failed to get baseURL: %w", err)
	}

	token, err := ctx.GetConfig("token")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	namespace, _ := ctx.GetConfig("namespace")

	return &Client{
		BaseURL:   string(baseURL),
		Token:     string(token),
		Namespace: string(namespace),
		http:      httpCtx,
	}, nil
}

func (c *Client) LookupSelf() error {
	return nil
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	return nil, nil
}
```

---

- [ ] **Step 1.3: Run tests — expect failures**

```bash
make test PKG_TEST_PACKAGES=./pkg/integrations/hashicorp_vault
```

Expected: multiple FAIL — `TestSync_Success` (state not "ready"), `TestSync_InvalidToken` (no error returned), `TestSync_WithNamespace` (no header sent), etc.

---

- [ ] **Step 1.4: Implement full `vault.go`**

Replace `vault.go` with the complete implementation:

```go
package hashicorp_vault

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("hashicorp_vault", &HashicorpVault{})
}

type HashicorpVault struct{}

type vaultConfig struct {
	BaseURL   string `mapstructure:"baseURL"`
	Namespace string `mapstructure:"namespace"`
	Token     string `mapstructure:"token"`
}

func (v *HashicorpVault) Name() string         { return "hashicorp_vault" }
func (v *HashicorpVault) Label() string        { return "HashiCorp Vault" }
func (v *HashicorpVault) Icon() string         { return "vault" }
func (v *HashicorpVault) Description() string  { return "Securely read secrets from HashiCorp Vault" }
func (v *HashicorpVault) Instructions() string { return "" }

func (v *HashicorpVault) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Vault server URL, e.g. https://vault.example.com",
		},
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Vault Enterprise namespace. Leave empty for community edition.",
		},
		{
			Name:        "token",
			Label:       "Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Vault token (hvs.… or s.…)",
		},
	}
}

func (v *HashicorpVault) Components() []core.Component  { return []core.Component{} }
func (v *HashicorpVault) Triggers() []core.Trigger      { return []core.Trigger{} }
func (v *HashicorpVault) Actions() []core.Action        { return []core.Action{} }
func (v *HashicorpVault) HandleRequest(ctx core.HTTPRequestContext) {}

func (v *HashicorpVault) Cleanup(ctx core.IntegrationCleanupContext) error    { return nil }
func (v *HashicorpVault) HandleAction(ctx core.IntegrationActionContext) error { return nil }

func (v *HashicorpVault) Sync(ctx core.SyncContext) error {
	cfg := vaultConfig{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if cfg.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	if cfg.Token == "" {
		return fmt.Errorf("token is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.LookupSelf(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (v *HashicorpVault) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}
```

---

- [ ] **Step 1.5: Implement full `client.go`**

Replace `client.go` with the complete implementation:

```go
package hashicorp_vault

import (
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL   string
	Token     string
	Namespace string
	http      core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseURL")
	if err != nil {
		return nil, fmt.Errorf("failed to get baseURL: %w", err)
	}
	if len(baseURL) == 0 {
		return nil, fmt.Errorf("baseURL is required")
	}

	token, err := ctx.GetConfig("token")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if len(token) == 0 {
		return nil, fmt.Errorf("token is required")
	}

	namespace, _ := ctx.GetConfig("namespace")

	return &Client{
		BaseURL:   string(baseURL),
		Token:     string(token),
		Namespace: string(namespace),
		http:      httpCtx,
	}, nil
}

func (c *Client) LookupSelf() error {
	_, err := c.execRequest(http.MethodGet, "/v1/auth/token/lookup-self", nil)
	return err
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("X-Vault-Token", c.Token)
	req.Header.Set("Content-Type", "application/json")
	if c.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.Namespace)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
```

---

- [ ] **Step 1.6: Run tests — expect all to pass**

```bash
make test PKG_TEST_PACKAGES=./pkg/integrations/hashicorp_vault
```

Expected: `ok  github.com/superplanehq/superplane/pkg/integrations/hashicorp_vault`

---

- [ ] **Step 1.7: Format and commit**

```bash
make format.go
git add pkg/integrations/hashicorp_vault/vault.go \
        pkg/integrations/hashicorp_vault/client.go \
        pkg/integrations/hashicorp_vault/vault_test.go
git commit -m "feat: add HashiCorp Vault integration with Token auth and Sync verification"
```

---

## Task 2: Get Secret component

**Files:**
- Create: `pkg/integrations/hashicorp_vault/get_secret_test.go`
- Create: `pkg/integrations/hashicorp_vault/get_secret.go`
- Create: `pkg/integrations/hashicorp_vault/example.go`
- Create: `pkg/integrations/hashicorp_vault/example_output_get_secret.json`
- Modify: `pkg/integrations/hashicorp_vault/client.go` (add `GetKVSecret`)
- Modify: `pkg/integrations/hashicorp_vault/vault.go` (update `Components()`)

---

- [ ] **Step 2.1: Create `get_secret_test.go`**

```go
package hashicorp_vault

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func kvSecretJSON(dataJSON string) string {
	return `{"data":{"data":` + dataJSON + `,"metadata":{"version":3,"created_time":"2025-01-01T00:00:00Z","deletion_time":"","destroyed":false}}}`
}

func TestGetSecret_Execute_AllData(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(kvSecretJSON(`{"username":"admin","password":"s3cr3t"}`))),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"},
	}

	c := &getSecret{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"mount": "secret", "path": "myapp/db"},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, SecretPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)
	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(secretPayload)
	assert.Equal(t, "secret", payload.Mount)
	assert.Equal(t, "myapp/db", payload.Path)
	assert.Equal(t, "admin", payload.Data["username"])
	assert.Empty(t, payload.Value)
}

func TestGetSecret_Execute_SpecificKey(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(kvSecretJSON(`{"username":"admin","password":"s3cr3t"}`))),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	c := &getSecret{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"mount": "secret", "path": "myapp/db", "key": "username"},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"}},
	})

	require.NoError(t, err)
	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(secretPayload)
	assert.Equal(t, "admin", payload.Value)
}

func TestGetSecret_Execute_KeyNotFound(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(kvSecretJSON(`{"username":"admin"}`))),
		}},
	}

	c := &getSecret{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"mount": "secret", "path": "myapp/db", "key": "password"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `"password" not found`)
}

func TestGetSecret_Execute_APIError(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"errors":["permission denied"]}`)),
		}},
	}

	c := &getSecret{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"path": "myapp/db"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestGetSecret_Setup_MissingPath(t *testing.T) {
	c := &getSecret{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{"path": ""},
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		Metadata:      &contexts.MetadataContext{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestGetSecret_Execute_DefaultMount(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(kvSecretJSON(`{"key":"val"}`))),
		}},
	}

	c := &getSecret{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"path": "myapp/config"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://vault.example.com", "token": "hvs.test"}},
	})

	require.NoError(t, err)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/secret/data/myapp/config")
}
```

---

- [ ] **Step 2.2: Create stub `get_secret.go` (compiles, returns nil/empty)**

```go
package hashicorp_vault

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SecretPayloadType = "hashicorp_vault.secret"

type getSecret struct{}

type getSecretSpec struct {
	Mount string `mapstructure:"mount"`
	Path  string `mapstructure:"path"`
	Key   string `mapstructure:"key"`
}

type secretPayload struct {
	Mount    string           `json:"mount"`
	Path     string           `json:"path"`
	Data     map[string]any   `json:"data"`
	Value    string           `json:"value,omitempty"`
	Metadata KVSecretMetadata `json:"metadata"`
}

func (c *getSecret) Name() string         { return "hashicorp_vault.getSecret" }
func (c *getSecret) Label() string        { return "Get Secret" }
func (c *getSecret) Description() string  { return "Read a secret from HashiCorp Vault KV v2" }
func (c *getSecret) Documentation() string { return "" }
func (c *getSecret) Icon() string         { return "lock" }
func (c *getSecret) Color() string        { return "gray" }

func (c *getSecret) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *getSecret) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *getSecret) Setup(ctx core.SetupContext) error   { return nil }
func (c *getSecret) Execute(ctx core.ExecutionContext) error { return nil }
func (c *getSecret) Cancel(ctx core.ExecutionContext) error  { return nil }
func (c *getSecret) Cleanup(ctx core.SetupContext) error     { return nil }

func (c *getSecret) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *getSecret) Actions() []core.Action                    { return []core.Action{} }
func (c *getSecret) HandleAction(ctx core.ActionContext) error  { return nil }

func (c *getSecret) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
```

Note: `KVSecretMetadata` is referenced here but defined in `client.go` in Step 2.4.

---

- [ ] **Step 2.3: Add `GetKVSecret` and KV types to `client.go`** *(do this before running tests — stub references `KVSecretMetadata`)*

Add these types and method to the end of `client.go` (before the closing of the file):

```go
// KVSecretMetadata holds version info returned by the KV v2 API.
type KVSecretMetadata struct {
	Version      int    `json:"version"`
	CreatedTime  string `json:"created_time"`
	DeletionTime string `json:"deletion_time"`
	Destroyed    bool   `json:"destroyed"`
}

// KVSecret is the parsed result of a KV v2 GET request.
type KVSecret struct {
	Data     map[string]any   `json:"data"`
	Metadata KVSecretMetadata `json:"metadata"`
}

type kvSecretResponse struct {
	Data struct {
		Data     map[string]any   `json:"data"`
		Metadata KVSecretMetadata `json:"metadata"`
	} `json:"data"`
}

func (c *Client) GetKVSecret(mount, path string) (*KVSecret, error) {
	endpoint := fmt.Sprintf("/v1/%s/data/%s", mount, path)
	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response kvSecretResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &KVSecret{
		Data:     response.Data.Data,
		Metadata: response.Data.Metadata,
	}, nil
}
```

Also add `"encoding/json"` to the import block at the top of `client.go`:

```go
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)
```

---

- [ ] **Step 2.4: Run tests — expect failures on the new tests**

```bash
make test PKG_TEST_PACKAGES=./pkg/integrations/hashicorp_vault
```

Expected: Task 1 tests still pass. New tests fail — `TestGetSecret_Execute_AllData` (stub Execute returns nil, no payload emitted), etc.

---

- [ ] **Step 2.5: Implement full `get_secret.go`**

Replace `get_secret.go` with the complete implementation:

```go
package hashicorp_vault

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SecretPayloadType = "hashicorp_vault.secret"

type getSecret struct{}

type getSecretSpec struct {
	Mount string `mapstructure:"mount"`
	Path  string `mapstructure:"path"`
	Key   string `mapstructure:"key"`
}

type secretPayload struct {
	Mount    string           `json:"mount"`
	Path     string           `json:"path"`
	Data     map[string]any   `json:"data"`
	Value    string           `json:"value,omitempty"`
	Metadata KVSecretMetadata `json:"metadata"`
}

func (c *getSecret) Name() string        { return "hashicorp_vault.getSecret" }
func (c *getSecret) Label() string       { return "Get Secret" }
func (c *getSecret) Description() string { return "Read a secret from HashiCorp Vault KV v2" }
func (c *getSecret) Icon() string        { return "lock" }
func (c *getSecret) Color() string       { return "gray" }

func (c *getSecret) Documentation() string {
	return `The Get Secret component reads a secret from a HashiCorp Vault KV v2 secrets engine.

## Use Cases

- **Inject secrets into workflows**: Read credentials, API keys, or certificates at runtime
- **Secret rotation checks**: Read the latest version of a secret after rotation
- **Conditional workflows**: Branch based on secret values

## Configuration

- **Mount**: The KV v2 secrets engine mount path (default: "secret")
- **Path**: The secret path within the mount, e.g. "myapp/db"
- **Key**: Optional. If set, extracts a single key from the secret data. Available as "value" in the output.

## Output

Returns the full secret data map, optional extracted value, and version metadata.`
}

func (c *getSecret) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *getSecret) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "mount",
			Label:       "Mount",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "secret",
			Description: "KV v2 mount path",
		},
		{
			Name:        "path",
			Label:       "Secret Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "myapp/db",
			Description: "Path to the secret, e.g. myapp/db",
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "password",
			Description: "Optional. Extract a specific key from the secret data.",
		},
	}
}

func (c *getSecret) Setup(ctx core.SetupContext) error {
	spec := getSecretSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}

func (c *getSecret) Execute(ctx core.ExecutionContext) error {
	spec := getSecretSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	mount := spec.Mount
	if mount == "" {
		mount = "secret"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	secret, err := client.GetKVSecret(mount, spec.Path)
	if err != nil {
		return err
	}

	payload := secretPayload{
		Mount:    mount,
		Path:     spec.Path,
		Data:     secret.Data,
		Metadata: secret.Metadata,
	}

	if spec.Key != "" {
		val, ok := secret.Data[spec.Key]
		if !ok {
			return fmt.Errorf("key %q not found in secret data", spec.Key)
		}

		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("key %q has non-string value", spec.Key)
		}

		payload.Value = strVal
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		SecretPayloadType,
		[]any{payload},
	)
}

func (c *getSecret) Cancel(ctx core.ExecutionContext) error  { return nil }
func (c *getSecret) Cleanup(ctx core.SetupContext) error     { return nil }

func (c *getSecret) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *getSecret) Actions() []core.Action                   { return []core.Action{} }
func (c *getSecret) HandleAction(ctx core.ActionContext) error { return nil }

func (c *getSecret) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
```

---

- [ ] **Step 2.6: Create `example_output_get_secret.json`**

`pkg/integrations/hashicorp_vault/example_output_get_secret.json`:
```json
{
  "mount": "secret",
  "path": "myapp/db",
  "data": {
    "username": "admin",
    "password": "s3cr3t"
  },
  "value": "",
  "metadata": {
    "version": 3,
    "created_time": "2025-01-01T00:00:00Z",
    "deletion_time": "",
    "destroyed": false
  }
}
```

---

- [ ] **Step 2.7: Create `example.go`**

`pkg/integrations/hashicorp_vault/example.go`:
```go
package hashicorp_vault

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_secret.json
var exampleOutputGetSecretBytes []byte

var exampleOutputGetSecretOnce sync.Once
var exampleOutputGetSecret map[string]any

func (c *getSecret) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSecretOnce, exampleOutputGetSecretBytes, &exampleOutputGetSecret)
}
```

---

- [ ] **Step 2.8: Update `Components()` in `vault.go`**

Change the `Components()` method in `vault.go` from:
```go
func (v *HashicorpVault) Components() []core.Component { return []core.Component{} }
```
to:
```go
func (v *HashicorpVault) Components() []core.Component {
	return []core.Component{
		&getSecret{},
	}
}
```

---

- [ ] **Step 2.9: Run tests — expect all to pass**

```bash
make test PKG_TEST_PACKAGES=./pkg/integrations/hashicorp_vault
```

Expected: `ok  github.com/superplanehq/superplane/pkg/integrations/hashicorp_vault`

All 11 tests pass (5 Sync + 6 GetSecret).

---

- [ ] **Step 2.10: Format and commit**

```bash
make format.go
git add pkg/integrations/hashicorp_vault/
git commit -m "feat: add HashiCorp Vault Get Secret component (KV v2)"
```

---

## Task 3: Wire up integration into the server

**Files:**
- Modify: `pkg/server/server.go`

---

- [ ] **Step 3.1: Add blank import to `pkg/server/server.go`**

In `pkg/server/server.go`, find the import block containing other integration blank imports. They are alphabetically sorted. Add the new line between `harness` and `hetzner`:

```go
	_ "github.com/superplanehq/superplane/pkg/integrations/harness"
	_ "github.com/superplanehq/superplane/pkg/integrations/hashicorp_vault"
	_ "github.com/superplanehq/superplane/pkg/integrations/hetzner"
```

---

- [ ] **Step 3.2: Build and lint check**

```bash
make check.build.app
make lint
```

Expected: no errors.

---

- [ ] **Step 3.3: Format and commit**

```bash
make format.go
git add pkg/server/server.go
git commit -m "feat: register HashiCorp Vault integration in server"
```

---

## Done

At this point:
- All 11 unit tests pass
- The integration compiles and is registered in the server
- `pkg/integrations/hashicorp_vault/` contains all 7 files
- One line added to `pkg/server/server.go`

The PR is ready to open against `superplanehq/superplane` referencing issue #3928.
