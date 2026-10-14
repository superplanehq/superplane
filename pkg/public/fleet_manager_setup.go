package public

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultStableRunnerVersion       = "v0.1.0"
	defaultStableFleetManagerVersion = "v0.1.0"
	fleetManagerChartRepository      = "oci://ghcr.io/superplanehq/fleet-manager-chart"
	fleetManagerImageRepository      = "ghcr.io/superplanehq/fleet-manager"
	runnerReleaseURLTemplate         = "https://github.com/superplanehq/superplane/releases/download/runner%2F{version}"
)

type fleetManagerSetupResponse struct {
	DefaultRunnerVersion       string                    `json:"defaultRunnerVersion"`
	DefaultFleetManagerVersion string                    `json:"defaultFleetManagerVersion"`
	DefaultChartVersion        string                    `json:"defaultChartVersion"`
	RunnerReleaseURLTemplate   string                    `json:"runnerReleaseUrlTemplate"`
	ChartRepository            string                    `json:"chartRepository"`
	ImageRepository            string                    `json:"imageRepository"`
	FleetPresets               []fleetManagerFleetPreset `json:"fleetPresets"`
}

type fleetManagerFleetPreset struct {
	ID   string                `json:"id"`
	Spec fleetManagerFleetSpec `json:"spec"`
}

type fleetManagerFleetSpec struct {
	OperatingSystem string   `json:"operatingSystem"`
	Architecture    string   `json:"architecture"`
	CPUMillicores   int32    `json:"cpuMillicores"`
	MemoryMB        int32    `json:"memoryMb"`
	DiskGB          int32    `json:"diskGb"`
	Capabilities    []string `json:"capabilities"`
}

func (s *Server) adminGetFleetManagerSetup(w http.ResponseWriter, _ *http.Request) {
	runnerVersion := configuredVersion("RUNNER_STABLE_VERSION", defaultStableRunnerVersion)
	fleetManagerVersion := configuredVersion(
		"FLEET_MANAGER_STABLE_VERSION",
		defaultStableFleetManagerVersion,
	)
	defaults := models.DefaultInstallationRunnerFleets(runnerVersion)
	presets := make([]fleetManagerFleetPreset, 0, len(defaults))
	for _, fleet := range defaults {
		spec := fleet.Spec.Data()
		presets = append(presets, fleetManagerFleetPreset{
			ID: fleet.Slug,
			Spec: fleetManagerFleetSpec{
				OperatingSystem: spec.OperatingSystem,
				Architecture:    spec.Architecture,
				CPUMillicores:   spec.CPUMillicores,
				MemoryMB:        spec.MemoryMB,
				DiskGB:          spec.DiskGB,
				Capabilities:    spec.Capabilities,
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fleetManagerSetupResponse{
		DefaultRunnerVersion:       runnerVersion,
		DefaultFleetManagerVersion: fleetManagerVersion,
		DefaultChartVersion:        strings.TrimPrefix(fleetManagerVersion, "v"),
		RunnerReleaseURLTemplate:   runnerReleaseURLTemplate,
		ChartRepository:            fleetManagerChartRepository,
		ImageRepository:            fleetManagerImageRepository,
		FleetPresets:               presets,
	})
}

func configuredVersion(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
