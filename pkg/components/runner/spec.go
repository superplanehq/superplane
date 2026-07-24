package runner

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
)

// Execution modes accepted by the task broker / fleet manager (lowercase JSON).
const (
	ExecutionModeHost   = "host"
	ExecutionModeDocker = "docker"

	// DockerImagePresetCustom selects the free-text docker_image field instead of a quick-pick ref.
	DockerImagePresetCustom = "custom"

	maxExecutionTimeoutSecondsRequest = 86400
	// maxDockerImageReferenceChars caps OCI image references (name:tag, registry/repo@digest, etc.).
	maxDockerImageReferenceChars = 2048

	// DefaultExecutionTimeoutSeconds is the wall-clock limit when a node omits execution_timeout_seconds.
	DefaultExecutionTimeoutSeconds = 3600 // 1 hour
)

// EnvironmentVariable is one row in the Runner "Environment variables" list.
type EnvironmentVariable struct {
	Name        string                     `json:"name" mapstructure:"name"`
	ValueSource string                     `json:"valueSource" mapstructure:"valueSource"`
	Value       *string                    `json:"value,omitempty" mapstructure:"value"`
	Secret      configuration.SecretKeyRef `json:"secret,omitempty" mapstructure:"secret"`
}

// Spec is persisted Runner node configuration.
type Spec struct {
	MachineType             string                 `mapstructure:"machine_type"`
	Commands                string                 `mapstructure:"commands"`
	EnvironmentFrom         []EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []EnvironmentVariable  `mapstructure:"environment"`
	ExecutionMode           string                 `mapstructure:"execution_mode"`
	DockerImagePreset       string                 `mapstructure:"docker_image_preset"`
	DockerImage             string                 `mapstructure:"docker_image"`
	ExecutionTimeoutSeconds int                    `mapstructure:"execution_timeout_seconds"` // 0 = DefaultExecutionTimeoutSeconds
}

func NewSpecDecoder(result any) (*mapstructure.Decoder, error) {
	return mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           result,
		WeaklyTypedInput: true,
	})
}

func decodeRunnerSpec(raw any) (Spec, error) {
	var spec Spec
	dec, err := NewSpecDecoder(&spec)
	if err != nil {
		return Spec{}, fmt.Errorf("runner spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return Spec{}, fmt.Errorf("decode runner configuration: %w", err)
	}
	applyRunnerSpecDefaults(&spec)
	return spec, nil
}

// applyRunnerSpecDefaults fills in values for nodes created before newer Runner fields existed.
// configuration.ValidateConfiguration does not apply Field.Default, so those fields stay optional at the schema level.
func applyRunnerSpecDefaults(spec *Spec) {
	if strings.TrimSpace(spec.ExecutionMode) == "" {
		spec.ExecutionMode = ExecutionModeHost
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = DefaultExecutionTimeoutSeconds
	}
}

func normalizeExecutionMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case ExecutionModeDocker:
		return ExecutionModeDocker
	default:
		return ExecutionModeHost
	}
}

// resolvedDockerImageRef returns the OCI reference sent to the broker in Docker mode.
// Legacy configs omit docker_image_preset and only set docker_image.
func resolvedDockerImageRef(spec Spec) string {
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

func validateRunnerSpec(spec Spec) error {
	if err := validateCommands(spec.Commands); err != nil {
		return err
	}

	if err := ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}

	if err := ValidateEnvironment(spec.Environment); err != nil {
		return err
	}

	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}

	ref := strings.TrimSpace(resolvedDockerImageRef(spec))
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
