package runner

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// RunBashSpec is persisted runnerBash node configuration.
type RunBashSpec struct {
	MachineType             string                `mapstructure:"machine_type"`
	Script                  string                `mapstructure:"script"`
	EnableSetupCommands     bool                  `mapstructure:"enable_setup_commands"`
	SetupCommands           string                `mapstructure:"setup_commands"`
	Environment             []EnvironmentVariable `mapstructure:"environment"`
	ExecutionMode           string                `mapstructure:"execution_mode"`
	DockerImagePreset       string                `mapstructure:"docker_image_preset"`
	DockerImage             string                `mapstructure:"docker_image"`
	ExecutionTimeoutSeconds int                   `mapstructure:"execution_timeout_seconds"` // 0 = DefaultExecutionTimeoutSeconds
}

func decodeRunBashSpec(raw any) (RunBashSpec, error) {
	var spec RunBashSpec
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return RunBashSpec{}, fmt.Errorf("runnerBash spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunBashSpec{}, fmt.Errorf("decode runnerBash configuration: %w", err)
	}
	applyRunBashSpecDefaults(&spec)
	return spec, nil
}

func applyRunBashSpecDefaults(spec *RunBashSpec) {
	if strings.TrimSpace(spec.ExecutionMode) == "" {
		spec.ExecutionMode = ExecutionModeHost
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = DefaultExecutionTimeoutSeconds
	}
	if strings.TrimSpace(spec.DockerImagePreset) == "" {
		spec.DockerImagePreset = runBashDefaultDockerPreset
	}
}

func resolvedRunBashDockerImageRef(spec RunBashSpec) string {
	if normalizeExecutionMode(spec.ExecutionMode) != ExecutionModeDocker {
		return ""
	}
	preset := strings.TrimSpace(spec.DockerImagePreset)
	custom := strings.TrimSpace(spec.DockerImage)
	if preset == "" {
		return custom
	}
	if preset == DockerImagePresetCustom {
		return custom
	}
	return preset
}

func validateRunBashSpec(spec RunBashSpec) error {
	if err := validateScript(spec.Script); err != nil {
		return err
	}

	if err := validateEnvironment(spec.Environment); err != nil {
		return err
	}

	if spec.EnableSetupCommands {
		if err := validateCommands(spec.SetupCommands); err != nil {
			return fmt.Errorf("setup commands: %w", err)
		}
	}

	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}

	ref := strings.TrimSpace(resolvedRunBashDockerImageRef(spec))
	mode := normalizeExecutionMode(spec.ExecutionMode)

	if ref != "" && len(ref) > maxDockerImageReferenceChars {
		return fmt.Errorf("container image reference must be at most %d characters", maxDockerImageReferenceChars)
	}
	if mode == ExecutionModeDocker && ref == "" {
		return fmt.Errorf("container image is required when execution mode is Docker")
	}

	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > maxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", maxExecutionTimeoutSecondsRequest, DefaultExecutionTimeoutSeconds)
		}
	}

	return nil
}
