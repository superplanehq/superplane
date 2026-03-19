package digitalocean

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

func Test__GetAlertPolicy__Setup(t *testing.T) {
	component := &GetAlertPolicy{}

	t.Run("missing alertPolicy returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "alertPolicy is required")
	})

	t.Run("empty alertPolicy returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"alertPolicy": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "alertPolicy is required")
	})

	t.Run("expression alertPolicy is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"alertPolicy": "{{ $.trigger.data.policyUuid }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid alertPolicy -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"alertPolicy": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"policy": {
								"uuid": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
								"description": "High CPU alert"
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__GetAlertPolicy__Execute(t *testing.T) {
	component := &GetAlertPolicy{}

	t.Run("successful fetch -> emits policy data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"policy": {
							"uuid": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
							"description": "High CPU alert",
							"type": "v1/insights/droplet/cpu",
							"compare": "GreaterThan",
							"value": 75,
							"window": "5m",
							"entities": ["557784760"],
							"tags": [],
							"alerts": {"slack": [], "email": ["ops@example.com"]},
							"enabled": true
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"alertPolicy": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.alertpolicy.fetched", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("policy not found (404) -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"The resource you requested could not be found."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"alertPolicy": "00000000-0000-0000-0000-000000000000",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get alert policy")
		assert.False(t, executionState.Passed)
	})
}
