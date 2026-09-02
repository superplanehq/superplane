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
	if err := runner.ValidateHostedAgentSpec(spec.Credentials, spec.Model, spec.Environment, envOpenAIBaseURL); err != nil {
		return err
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

func buildCodexBrokerTask(spec RunCodexSpec, usage string, setups []runner.IntegrationSetup) ([]runner.BrokerCommand, []runner.BrokerTaskFile) {
	return runner.BuildAgentBrokerTask(runner.AgentBrokerTaskInput{
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
}
