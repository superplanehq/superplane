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
