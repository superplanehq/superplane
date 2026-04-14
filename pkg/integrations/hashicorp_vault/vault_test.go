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
