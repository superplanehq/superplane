package runner

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRunBashExecuteSendsBashPayloadToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-bash-1"}`))},
		},
	}

	component := &RunBash{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"script": `num=$(jq -r '."GitHub PR".data.number' "$SUPERPLANE_PAYLOAD_FILE")
printf '{"pr":%s}\n' "$num" > "$SUPERPLANE_RESULT_FILE"`,
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
	assert.Equal(t, RunModeBash, req.RunMode)
	assert.Contains(t, req.Script, "jq")
	assert.Contains(t, req.Script, "SUPERPLANE_PAYLOAD_FILE")
	assert.Contains(t, req.Script, "SUPERPLANE_RESULT_FILE")
	assert.NotContains(t, string(body), `"commands"`)
	assert.Empty(t, req.SetupCommands)
	require.True(t, json.Valid(req.MessageChain))
	assert.Contains(t, string(req.MessageChain), "GitHub PR")
}

func TestRunBashExecuteSendsSetupCommandsWhenEnabled(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-bash-setup-1"}`))},
		},
	}

	component := &RunBash{}
	setupCommands := `if ! command -v aws >/dev/null 2>&1; then
  apt-get update
  apt-get install -y awscli
fi
echo ready`

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type":          testRunnerMachineType,
			"enable_setup_commands": true,
			"setup_commands":        setupCommands,
			"script":                `echo '{"ok":true}' > "$SUPERPLANE_RESULT_FILE"`,
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

	assert.Equal(t, []string{`if ! command -v aws >/dev/null 2>&1; then
apt-get update
apt-get install -y awscli
fi`, "echo ready"}, req.SetupCommands)
}

func TestValidateRunBashSpecRequiresSetupCommandsWhenEnabled(t *testing.T) {
	t.Parallel()

	spec := RunBashSpec{
		MachineType:         testRunnerMachineType,
		Script:              `echo ok > "$SUPERPLANE_RESULT_FILE"`,
		EnableSetupCommands: true,
		SetupCommands:       "   \n  ",
	}
	require.Error(t, validateRunBashSpec(spec))
}

func TestValidateRunBashSpec(t *testing.T) {
	t.Parallel()

	spec := RunBashSpec{
		MachineType: testRunnerMachineType,
		Script:      `echo ok > "$SUPERPLANE_RESULT_FILE"`,
	}
	require.NoError(t, validateRunBashSpec(spec))

	spec.Script = "  "
	require.Error(t, validateRunBashSpec(spec))
}

func TestRunBashProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"ok":true}`),
	}
	require.NoError(t, processBrokerTaskStatus(state, task, RunBashFinishedEventType))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, RunBashFinishedEventType, wrapped["type"])
}
