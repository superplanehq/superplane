package runner

import (
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestNormalizeExecutionMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", ExecutionModeHost},
		{"host", ExecutionModeHost},
		{"HOST", ExecutionModeHost},
		{"docker", ExecutionModeDocker},
		{" Docker ", ExecutionModeDocker},
		{"vm", ExecutionModeHost},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := normalizeExecutionMode(tc.in); got != tc.want {
				t.Fatalf("normalizeExecutionMode(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

const testRunnerMachineType = MachineTypeE1LargeAMD64

func TestValidateRunnerSpec(t *testing.T) {
	t.Parallel()
	if err := validateRunnerSpec(Spec{MachineType: testRunnerMachineType, Commands: "echo hi", ExecutionMode: ExecutionModeHost}); err != nil {
		t.Fatalf("valid host spec: %v", err)
	}
	// Legacy persisted config: commands only (no execution_mode / execution_timeout_seconds keys).
	if err := validateRunnerSpec(Spec{MachineType: testRunnerMachineType, Commands: "echo hi"}); err != nil {
		t.Fatalf("valid legacy host spec (empty execution_mode): %v", err)
	}
	if err := validateRunnerSpec(Spec{
		MachineType:       testRunnerMachineType,
		Commands:          "echo hi",
		ExecutionMode:     ExecutionModeDocker,
		DockerImagePreset: "debian:bookworm-slim",
	}); err != nil {
		t.Fatalf("valid docker quick pick: %v", err)
	}
	if err := validateRunnerSpec(Spec{
		MachineType:       testRunnerMachineType,
		Commands:          "echo hi",
		ExecutionMode:     ExecutionModeDocker,
		DockerImagePreset: DockerImagePresetCustom,
		DockerImage:       "my.registry.example.com/app:1.0.0",
	}); err != nil {
		t.Fatalf("valid docker custom: %v", err)
	}
	// Legacy: no preset, only docker_image
	if err := validateRunnerSpec(Spec{
		MachineType:   testRunnerMachineType,
		Commands:      "echo hi",
		ExecutionMode: ExecutionModeDocker,
		DockerImage:   "debian:bookworm-slim",
	}); err != nil {
		t.Fatalf("valid docker legacy: %v", err)
	}
	if err := validateRunnerSpec(Spec{
		MachineType:             testRunnerMachineType,
		Commands:                "echo hi",
		ExecutionMode:           ExecutionModeHost,
		ExecutionTimeoutSeconds: 120,
	}); err != nil {
		t.Fatalf("valid timeout: %v", err)
	}

	if err := validateRunnerSpec(Spec{Commands: "echo hi", ExecutionMode: ExecutionModeHost}); err == nil {
		t.Fatal("expected error for missing machine type")
	}
	if err := validateRunnerSpec(Spec{MachineType: testRunnerMachineType, Commands: "", ExecutionMode: ExecutionModeHost}); err == nil {
		t.Fatal("expected error for empty commands")
	}
	if err := validateRunnerSpec(Spec{MachineType: testRunnerMachineType, Commands: "echo hi", ExecutionMode: ExecutionModeDocker}); err == nil {
		t.Fatal("expected error for docker without image")
	}
	if err := validateRunnerSpec(Spec{
		MachineType:       testRunnerMachineType,
		Commands:          "echo hi",
		ExecutionMode:     ExecutionModeDocker,
		DockerImagePreset: DockerImagePresetCustom,
		DockerImage:       "",
	}); err == nil {
		t.Fatal("expected error for docker custom without image")
	}
	longImage := strings.Repeat("a", maxDockerImageReferenceChars+1)
	if err := validateRunnerSpec(Spec{
		MachineType:       testRunnerMachineType,
		Commands:          "echo hi",
		ExecutionMode:     ExecutionModeDocker,
		DockerImagePreset: DockerImagePresetCustom,
		DockerImage:       longImage,
	}); err == nil {
		t.Fatal("expected error for docker image reference that is too long")
	}
	if err := validateRunnerSpec(Spec{
		MachineType:             testRunnerMachineType,
		Commands:                "echo hi",
		ExecutionMode:           ExecutionModeHost,
		ExecutionTimeoutSeconds: -1,
	}); err == nil {
		t.Fatal("expected error for negative timeout")
	}
	if err := validateRunnerSpec(Spec{
		MachineType:             testRunnerMachineType,
		Commands:                "echo hi",
		ExecutionMode:           ExecutionModeHost,
		ExecutionTimeoutSeconds: maxExecutionTimeoutSecondsRequest + 1,
	}); err == nil {
		t.Fatal("expected error for timeout above max")
	}
}

func TestDecodeRunnerSpec_WeakTypes(t *testing.T) {
	t.Parallel()
	raw := map[string]any{
		"machine_type":              testRunnerMachineType,
		"commands":                  "echo x",
		"execution_mode":            "docker",
		"docker_image_preset":       "debian:bookworm-slim",
		"docker_image":              " alpine:latest ",
		"execution_timeout_seconds": float64(90),
	}
	spec, err := decodeRunnerSpec(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if spec.Commands != "echo x" || spec.ExecutionMode != "docker" || spec.DockerImagePreset != "debian:bookworm-slim" || spec.DockerImage != " alpine:latest " {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if spec.ExecutionTimeoutSeconds != 90 {
		t.Fatalf("timeout: got %d want 90", spec.ExecutionTimeoutSeconds)
	}
	if err := validateRunnerSpec(spec); err != nil {
		t.Fatalf("validate after decode: %v", err)
	}
}

func TestValidateConfigurationRunnerLegacyPreExecutionFields(t *testing.T) {
	t.Parallel()
	r := &Runner{}
	legacy := map[string]any{
		"machine_type": testRunnerMachineType,
		"commands":     "echo hi",
	}
	if err := configuration.ValidateConfiguration(r.Configuration(), legacy); err != nil {
		t.Fatalf("ValidateConfiguration legacy runner: %v", err)
	}
	spec, err := decodeRunnerSpec(legacy)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if spec.ExecutionMode != ExecutionModeHost {
		t.Fatalf("execution_mode default: got %q want %q", spec.ExecutionMode, ExecutionModeHost)
	}
	if spec.ExecutionTimeoutSeconds != DefaultExecutionTimeoutSeconds {
		t.Fatalf("timeout default: got %d want %d", spec.ExecutionTimeoutSeconds, DefaultExecutionTimeoutSeconds)
	}
	if err := validateRunnerSpec(spec); err != nil {
		t.Fatalf("validateRunnerSpec legacy: %v", err)
	}
}

func TestValidateConfigurationRunnerLegacyDockerImageOnly(t *testing.T) {
	t.Parallel()
	r := &Runner{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":              testRunnerMachineType,
		"execution_mode":            ExecutionModeDocker,
		"commands":                  "echo hi",
		"execution_timeout_seconds": 0,
		"docker_image":              "debian:bookworm-slim",
	})
	if err != nil {
		t.Fatalf("legacy docker_image without preset: %v", err)
	}
}
