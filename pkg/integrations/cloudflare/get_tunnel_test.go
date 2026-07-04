package cloudflare

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

func Test__GetTunnel__Setup(t *testing.T) {
	component := &GetTunnel{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"tunnel":    "tun123",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "accountId is required")
	})

	t.Run("missing tunnel returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"tunnel":    "",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "tunnel is required")
	})

	t.Run("expression tunnel id is accepted without API call", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"tunnel":    "{{ $.trigger.data.tunnelId }}",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid configuration resolves tunnel name", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"tunnel":    "tun123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": {"id": "tun123", "name": "edge-tunnel"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "token123"},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)

		meta, ok := ctx.Metadata.(*contexts.MetadataContext)
		require.True(t, ok)
		tunnelMeta, ok := meta.Metadata.(TunnelNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "edge-tunnel", tunnelMeta.TunnelName)
	})
}

func Test__GetTunnel__Execute(t *testing.T) {
	component := &GetTunnel{}

	t.Run("successful get emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "tun123",
							"name": "edge-tunnel",
							"status": "healthy",
							"config_src": "cloudflare"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"tunnel":    "tun123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.tunnel.fetched", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/cfd_tunnel/tun123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	})
}

func Test__CreateTunnel__Execute(t *testing.T) {
	component := &CreateTunnel{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": {
						"id": "new-tun",
						"name": "my-tunnel",
						"status": "inactive",
						"config_src": "cloudflare"
					}
				}`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"accountId": "acc123",
			"name":      "my-tunnel",
			"configSrc": "cloudflare",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "tok"},
		},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.Equal(t, CreateTunnelPayloadType, execState.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/cfd_tunnel", httpContext.Requests[0].URL.String())
}

func Test__DeleteTunnel__Execute(t *testing.T) {
	component := &DeleteTunnel{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"accountId": "acc123",
			"tunnel":    "tun123",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "tok"},
		},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.Equal(t, DeleteTunnelPayloadType, execState.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/cfd_tunnel/tun123")
}
