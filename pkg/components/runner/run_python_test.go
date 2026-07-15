package runner

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRunPythonExecuteSendsPythonPayloadToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-py-1"}`))},
		},
	}

	component := &RunPython{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"script": `def main(payload):
    return {"pr": payload["GitHub PR"]["data"]["number"]}`,
		},
		HTTP:           httpContext,
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
		Expressions: &stubMessageChainBuilder{
			chain: map[string]any{
				"GitHub PR": map[string]any{"data": map[string]any{"number": 42}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Equal(t, RunModePython, req.RunMode)
	assert.Equal(t, config.MaxWebhookPayloadSize, req.WebhookPayloadSizeLimit)
	assert.Contains(t, req.Script, "def main(payload)")
	assert.NotContains(t, string(body), `"commands"`)
	assert.Empty(t, req.SetupCommands)
	require.True(t, json.Valid(req.MessageChain))
	assert.Contains(t, string(req.MessageChain), "GitHub PR")
}

func TestRunPythonExecuteSendsSetupCommandsWhenEnabled(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-py-setup-1"}`))},
		},
	}

	component := &RunPython{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type":          testRunnerMachineType,
			"enable_setup_commands": true,
			"setup_commands":        "pip install requests\necho ready",
			"script":                "def main(payload):\n    return {\"ok\": True}",
		},
		HTTP:           httpContext,
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
		Expressions:    &stubMessageChainBuilder{chain: map[string]any{}},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, []string{"pip install requests", "echo ready"}, req.SetupCommands)
}

func TestValidateRunPythonSpecRequiresSetupCommandsWhenEnabled(t *testing.T) {
	t.Parallel()

	spec := RunPythonSpec{
		MachineType:         testRunnerMachineType,
		Script:              "def main(payload):\n    return 1",
		EnableSetupCommands: true,
		SetupCommands:       "   \n  ",
	}
	require.Error(t, validateRunPythonSpec(spec))
}

func TestValidateRunPythonSpec(t *testing.T) {
	t.Parallel()

	spec := RunPythonSpec{
		MachineType: testRunnerMachineType,
		Script:      "def main(payload):\n    return 1",
	}
	require.NoError(t, validateRunPythonSpec(spec))

	spec.Script = "  "
	require.Error(t, validateRunPythonSpec(spec))
}

func TestRunPythonProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"ok":true}`),
	}
	require.NoError(t, processBrokerTaskStatus(state, task, TaskOutcome{FinishedEventType: RunPythonFinishedEventType}))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, RunPythonFinishedEventType, wrapped["type"])
}
