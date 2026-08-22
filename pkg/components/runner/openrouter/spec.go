package openrouter

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

type RunOpenRouterSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Steps                   []runner.AgentStep            `mapstructure:"steps"`
	Credentials             runner.AgentCredentials       `mapstructure:"credentials"`
	Model                   string                        `mapstructure:"model"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"`
}

func decodeRunOpenRouterSpec(raw any) (RunOpenRouterSpec, error) {
	var spec RunOpenRouterSpec
	dec, err := runner.NewSpecDecoder(&spec)
	if err != nil {
		return RunOpenRouterSpec{}, fmt.Errorf("runnerOpenRouter spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunOpenRouterSpec{}, fmt.Errorf("decode runnerOpenRouter configuration: %w", err)
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
	}
	return spec, nil
}

func validateRunOpenRouterSpec(spec RunOpenRouterSpec) error {
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
	if err := runner.ValidateReservedEnvironmentName(spec.Environment, envOpenRouterAPIKey); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

func buildOpenRouterBrokerTask(spec RunOpenRouterSpec) ([]runner.BrokerCommand, []runner.BrokerTaskFile) {
	return runner.BuildAgentBrokerTask(
		"Prepare OpenRouter agent",
		runner.NodePrepareScript("", "", spec.WorkingDirectory),
		"run.js",
		runScript,
		spec.WorkingDirectory,
		spec.Steps,
		strings.TrimSpace(spec.Model),
		func(promptName, model string) string {
			return fmt.Sprintf(
				`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
				promptName,
				runner.ShellSingleQuote(model),
			)
		},
	)
}
