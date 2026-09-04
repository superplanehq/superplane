package codex

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

type RunCodexSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Steps                   []runner.AgentStep            `mapstructure:"steps"`
	Credentials             runner.AgentCredentials       `mapstructure:"credentials"`
	Model                   string                        `mapstructure:"model"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"`
}

func decodeRunCodexSpec(raw any) (RunCodexSpec, error) {
	var spec RunCodexSpec
	dec, err := runner.NewSpecDecoder(&spec)
	if err != nil {
		return RunCodexSpec{}, fmt.Errorf("runnerCodex spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunCodexSpec{}, fmt.Errorf("decode runnerCodex configuration: %w", err)
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
	}
	return spec, nil
}

func validateRunCodexSpec(spec RunCodexSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if err := runner.ValidateAgentSteps(spec.Steps); err != nil {
		return err
	}
	if err := runner.RejectHostedCredentials(spec.Credentials); err != nil {
		return err
	}
	if err := runner.ValidateAgentCredentials(spec.Credentials, true); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	if err := runner.ValidateReservedEnvironmentName(spec.Environment, envOpenAIAPIKey); err != nil {
		return err
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

// CodexBrokerTask is the ordered broker commands and task files for a run.
type CodexBrokerTask struct {
	Commands []runner.BrokerCommand
	Files    []runner.BrokerTaskFile
}

func buildCodexBrokerTask(spec RunCodexSpec, usage string, setups []runner.IntegrationSetup) CodexBrokerTask {
	commands, files := runner.BuildAgentBrokerTask(runner.AgentBrokerTaskInput{
		PrepareName:      "Prepare Codex",
		PrepareScript:    runner.NodePrepareScript("codex", "codex CLI not found on PATH; install Codex on the runner", spec.WorkingDirectory),
		RunScriptName:    "run.js",
		RunScript:        runScript,
		WorkingDirectory: spec.WorkingDirectory,
		Steps:            spec.Steps,
		Usage:            usage,
		Setups:           setups,
		Model:            strings.TrimSpace(spec.Model),
		PromptCommand: func(promptName, model string) string {
			return fmt.Sprintf(
				`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
				promptName,
				runner.ShellSingleQuote(model),
			)
		},
	})
	return CodexBrokerTask{Commands: commands, Files: files}
}

func BuildBrokerTask(spec RunCodexSpec, usage string, setups []runner.IntegrationSetup) CodexBrokerTask {
	return buildCodexBrokerTask(spec, usage, setups)
}

func ApplyPlanningFollowUp(task CodexBrokerTask, environment []runner.BrokerEnvironmentVariable, spec RunCodexSpec) CodexBrokerTask {
	return applyPlanningFollowUp(task, environment, spec)
}

// applyPlanningFollowUp keeps the machine on after canvas steps when this run
// is a planning session. Line apps never attach a planning token, so they
// keep the default step list and finish.
func applyPlanningFollowUp(task CodexBrokerTask, environment []runner.BrokerEnvironmentVariable, spec RunCodexSpec) CodexBrokerTask {
	if !runner.HasPlanningSessionToken(environment) {
		return task
	}
	task.Files = append(task.Files, runner.FollowUpLoopFile())
	task.Commands = append(task.Commands, planningFollowUpCommand(spec))
	return task
}

func planningFollowUpCommand(spec RunCodexSpec) runner.BrokerCommand {
	workdir := planningFollowUpWorkingDirectory(spec)
	model := strings.TrimSpace(spec.Model)
	return runner.BrokerCommand{
		Name: "Wait for the next message",
		Command: runner.WrapAgentStepCommand(
			runner.WrapCommandInWorkingDirectory(
				workdir,
				fmt.Sprintf(`node "$SUPERPLANE_TASK_DIR/follow_up_loop.js" %s`, runner.ShellSingleQuote(model)),
			),
		),
		Kind:    runner.LiveLogKindPrompt,
		Preview: "Wait for the next user message",
	}
}

func planningFollowUpWorkingDirectory(spec RunCodexSpec) string {
	for i := len(spec.Steps) - 1; i >= 0; i-- {
		if runner.NormalizeAgentStepType(spec.Steps[i].Type) == runner.AgentStepPrompt {
			return runner.EffectiveWorkingDirectory(spec.WorkingDirectory, spec.Steps[i].WorkingDirectory)
		}
	}
	return strings.TrimSpace(spec.WorkingDirectory)
}
