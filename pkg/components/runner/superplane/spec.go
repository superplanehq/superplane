package superplane

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

const (
	envAnthropicAPIKey   = "ANTHROPIC_API_KEY"
	envAnthropicBaseURL  = "ANTHROPIC_BASE_URL"
	envOpenAIAPIKey      = "OPENAI_API_KEY"
	envOpenAIBaseURL     = "OPENAI_BASE_URL"
	envOpenRouterAPIKey  = "OPENROUTER_API_KEY"
	envOpenRouterBaseURL = "OPENROUTER_BASE_URL"
)

var reservedHostedEnvNames = []string{
	envAnthropicAPIKey,
	envAnthropicBaseURL,
	envOpenAIAPIKey,
	envOpenAIBaseURL,
	envOpenRouterAPIKey,
	envOpenRouterBaseURL,
}

var unsupportedSuperPlaneFields = []string{"credentials", "model", "maxTurns", "hostedProvider"}

type RunSuperPlaneSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Steps                   []runner.AgentStep            `mapstructure:"steps"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"`
}

func decodeRunSuperPlaneSpec(raw any) (RunSuperPlaneSpec, error) {
	return decodeRunSuperPlaneSpecWithSidecar(raw, false)
}

func decodeRunSuperPlaneSpecForExecute(raw any) (RunSuperPlaneSpec, error) {
	return decodeRunSuperPlaneSpecWithSidecar(raw, true)
}

func decodeRunSuperPlaneSpecWithSidecar(raw any, stripSidecar bool) (RunSuperPlaneSpec, error) {
	if stripSidecar {
		stripSuperPlaneSidecarFields(raw)
	} else if err := rejectUnsupportedSuperPlaneFields(raw); err != nil {
		return RunSuperPlaneSpec{}, err
	}

	var spec RunSuperPlaneSpec
	dec, err := runner.NewSpecDecoder(&spec)
	if err != nil {
		return RunSuperPlaneSpec{}, fmt.Errorf("runnerSuperPlane spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunSuperPlaneSpec{}, fmt.Errorf("decode runnerSuperPlane configuration: %w", err)
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
	}
	return spec, nil
}

func validateRunSuperPlaneSpec(spec RunSuperPlaneSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if err := runner.ValidateAgentSteps(spec.Steps); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	if err := runner.ValidateReservedEnvironmentNames(spec.Environment, reservedHostedEnvNames...); err != nil {
		return err
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

func rejectUnsupportedSuperPlaneFields(raw any) error {
	cfg, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range unsupportedSuperPlaneFields {
		if _, exists := cfg[key]; exists {
			return fmt.Errorf("%s is not supported on Run SuperPlane Agent", key)
		}
	}
	return nil
}

func stripSuperPlaneSidecarFields(raw any) {
	cfg, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for _, key := range unsupportedSuperPlaneFields {
		delete(cfg, key)
	}
}
