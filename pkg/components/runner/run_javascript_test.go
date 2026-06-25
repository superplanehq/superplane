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

type stubMessageChainBuilder struct {
	chain map[string]any
	err   error
}

func (s *stubMessageChainBuilder) Run(string) (any, error) { return nil, nil }
func (s *stubMessageChainBuilder) RunWithExtraVariables(string, map[string]any) (any, error) {
	return nil, nil
}
func (s *stubMessageChainBuilder) BuildExecutionMessageChain() (map[string]any, error) {
	return s.chain, s.err
}

func TestRunJSExecuteSendsJavaScriptPayloadToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-js-1"}`))},
		},
	}

	component := &RunJS{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"script": `function main() {
  return { pr: $('GitHub PR').data.number };
}`,
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
	assert.Equal(t, RunModeJavaScript, req.RunMode)
	assert.Contains(t, req.Script, "function main()")
	assert.NotContains(t, string(body), `"commands"`)
	assert.Empty(t, req.SetupCommands)
	require.True(t, json.Valid(req.MessageChain))
	assert.Contains(t, string(req.MessageChain), "GitHub PR")
}

func TestRunJSExecuteSendsSetupCommandsWhenEnabled(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-js-setup-1"}`))},
		},
	}

	component := &RunJS{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type":          testRunnerMachineType,
			"enable_setup_commands": true,
			"setup_commands":        "npm ci\necho ready",
			"script":                "function main() { return { ok: true }; }",
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

	assert.Equal(t, []string{"npm ci\necho ready"}, req.SetupCommands)
}

func TestValidateRunJSSpecRequiresSetupCommandsWhenEnabled(t *testing.T) {
	t.Parallel()

	spec := RunJSSpec{
		MachineType:         testRunnerMachineType,
		Script:              "function main() { return 1; }",
		EnableSetupCommands: true,
		SetupCommands:       "   \n  ",
	}
	require.Error(t, validateRunJSSpec(spec))
}

func TestValidateRunJSSpec(t *testing.T) {
	t.Parallel()

	spec := RunJSSpec{
		MachineType: testRunnerMachineType,
		Script:      "function main() { return 1; }",
	}
	require.NoError(t, validateRunJSSpec(spec))

	spec.Script = "  "
	require.Error(t, validateRunJSSpec(spec))
}

func TestRunJSProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"ok":true}`),
	}
	require.NoError(t, processBrokerTaskStatus(state, task, RunJSFinishedEventType))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, RunJSFinishedEventType, wrapped["type"])
}
