package openrouter

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRunOpenRouterExecuteUsesFactoryDefaultWhenStepModelIsEmpty(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-openrouter-default"}`))},
		},
	}
	err := (&RunOpenRouter{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": runner.MachineTypeE1LargeAMD64,
			"steps": []map[string]any{
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
			},
			"credentials": map[string]any{
				"source": "secret",
				"secret": map[string]any{"secret": "openrouter", "key": "api_key"},
			},
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{"openrouter/api_key": []byte("sk-or-test")},
		},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
		Factory: &contexts.FactoryContext{Default: core.FactoryDefaultAgentModel{
			Model: "moonshotai/kimi-k2.6",
		}},
		HostedLLM: &contexts.HostedLLMContext{},
	})
	require.NoError(t, err)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `'moonshotai/kimi-k2.6'`)
}

func TestRunOpenRouterExecuteRequiresModelWhenFactoryDefaultIsMissing(t *testing.T) {
	err := (&RunOpenRouter{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": runner.MachineTypeE1LargeAMD64,
			"steps": []map[string]any{
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
			},
			"credentials": map[string]any{
				"source": "secret",
				"secret": map[string]any{"secret": "openrouter", "key": "api_key"},
			},
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}
