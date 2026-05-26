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

func Test__DeleteLoadBalancer__Setup(t *testing.T) {
	component := &DeleteLoadBalancer{}

	t.Run("missing zoneId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"loadBalancer": "lb123",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "expected format zoneId/lbId")
	})

	t.Run("missing loadBalancer returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"loadBalancer": "",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "loadBalancer is required")
	})

	t.Run("expression loadBalancer is accepted without API call", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"loadBalancer": "{{ $.trigger.data.lbId }}",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid configuration resolves load balancer name", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"loadBalancer": "zone123/lb123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": {"id": "lb123", "name": "lb.example.com"}
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
		lbMeta, ok := meta.Metadata.(LoadBalancerNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "lb.example.com", lbMeta.LoadBalancerName)
	})
}

func Test__DeleteLoadBalancer__Execute(t *testing.T) {
	component := &DeleteLoadBalancer{}

	t.Run("successful delete emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "lb123"}}`)),
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
				"loadBalancer": "zone123/lb123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)

		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.loadBalancer.deleted", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/load_balancers/lb123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Load balancer not found"}]}`)),
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
				"loadBalancer": "zone123/lb123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete load balancer")
	})
}
