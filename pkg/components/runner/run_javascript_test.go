package runner

import (
	"encoding/json"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/config"
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

func TestRunJSExecuteSendsJavaScriptPayloadToRunner(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}

	httpContext := &contexts.HTTPContext{}

	component := &RunJS{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks: tasks,
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
	require.Len(t, httpContext.Requests, 0)

	body := tasks.Payload
	require.NotEmpty(t, body)

	var req runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, tasks.FleetID)
	assert.Equal(t, RunModeJavaScript, req.RunMode)
	assert.Equal(t, config.MaxWebhookPayloadSize, req.WebhookPayloadSizeLimit)
	assert.Contains(t, req.Script, "function main()")
	assert.NotContains(t, string(body), `"commands"`)
	assert.Empty(t, req.SetupCommands)
	require.True(t, json.Valid(req.MessageChain))
	assert.Contains(t, string(req.MessageChain), "GitHub PR")
}

func TestRunJSExecuteSendsSetupCommandsWhenEnabled(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}

	httpContext := &contexts.HTTPContext{}

	component := &RunJS{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks: tasks,
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
	require.Len(t, httpContext.Requests, 0)

	body := tasks.Payload
	require.NotEmpty(t, body)

	var req runnerapi.TaskPayload
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, []string{"npm ci", "echo ready"}, req.SetupCommands)
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
	task := &core.RunnerTask{
		State:    "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"ok":true}`),
	}
	require.NoError(t, processTaskStatus(state, task, RunJSFinishedEventType, "", nil, nil, nil))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, RunJSFinishedEventType, wrapped["type"])
}
