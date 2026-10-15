package public

import (
	"encoding/json"
	"errors"
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
	DefaultRunnerVersion       string                     `json:"defaultRunnerVersion"`
	DefaultFleetManagerVersion string                     `json:"defaultFleetManagerVersion"`
	DefaultChartVersion        string                     `json:"defaultChartVersion"`
	RunnerReleaseURLTemplate   string                     `json:"runnerReleaseUrlTemplate"`
	ChartRepository            string                     `json:"chartRepository"`
	ImageRepository            string                     `json:"imageRepository"`
	FleetPresets               []fleetManagerFleetPreset  `json:"fleetPresets"`
	InfrastructureDefaults     fleetManagerInfrastructure `json:"infrastructureDefaults"`
	Credentials                fleetManagerCredentials    `json:"credentials"`
}

type fleetManagerInfrastructure struct {
	Source             string         `json:"source"`
	Provider           *string        `json:"provider"`
	ManagerID          string         `json:"managerId"`
	Namespace          string         `json:"namespace"`
	ServiceAccountName string         `json:"serviceAccountName"`
	Fields             map[string]any `json:"fields"`
}

type fleetManagerCredentials struct {
	ExistingSecret  string `json:"existingSecret"`
	TokenKey        string `json:"tokenKey"`
	AutoProvisioned bool   `json:"autoProvisioned"`
}

type fleetManagerSetupConfig struct {
	Source             string         `json:"source"`
	Provider           *string        `json:"provider"`
	ManagerID          string         `json:"managerId"`
	Namespace          string         `json:"namespace"`
	ServiceAccountName string         `json:"serviceAccountName"`
	Infrastructure     map[string]any `json:"infrastructure"`
	Credentials        struct {
		ExistingSecret string `json:"existingSecret"`
		TokenKey       string `json:"tokenKey"`
		Provision      bool   `json:"provision"`
	} `json:"credentials"`
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
	setup, err := loadFleetManagerSetupConfig(os.Getenv("FLEET_MANAGER_SETUP_CONFIG_PATH"))
	if err != nil {
		http.Error(w, "Could not load Fleet Manager setup configuration", http.StatusInternalServerError)
		return
	}

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
		InfrastructureDefaults: fleetManagerInfrastructure{
			Source:             setup.Source,
			Provider:           setup.Provider,
			ManagerID:          setup.ManagerID,
			Namespace:          setup.Namespace,
			ServiceAccountName: setup.ServiceAccountName,
			Fields:             setup.Infrastructure,
		},
		Credentials: fleetManagerCredentials{
			ExistingSecret:  setup.Credentials.ExistingSecret,
			TokenKey:        setup.Credentials.TokenKey,
			AutoProvisioned: setup.Credentials.Provision,
		},
	})
}

func loadFleetManagerSetupConfig(path string) (fleetManagerSetupConfig, error) {
	setup := defaultFleetManagerSetupConfig()
	if strings.TrimSpace(path) == "" {
		return setup, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fleetManagerSetupConfig{}, err
	}
	if err := json.Unmarshal(contents, &setup); err != nil {
		return fleetManagerSetupConfig{}, err
	}
	if setup.Source != "terraform" && setup.Source != "manual" {
		return fleetManagerSetupConfig{}, errors.New("invalid Fleet Manager setup source")
	}
	if setup.Provider != nil && !validFleetManagerProvider(*setup.Provider) {
		return fleetManagerSetupConfig{}, errors.New("invalid Fleet Manager provider")
	}
	if setup.Infrastructure == nil {
		setup.Infrastructure = map[string]any{}
	}
	return setup, nil
}

func defaultFleetManagerSetupConfig() fleetManagerSetupConfig {
	setup := fleetManagerSetupConfig{
		Source:             "manual",
		ManagerID:          "my-cluster",
		Namespace:          "superplane",
		ServiceAccountName: "superplane-fleet-manager",
		Infrastructure:     map[string]any{},
	}
	setup.Credentials.ExistingSecret = "superplane-fleet-manager-token"
	setup.Credentials.TokenKey = "INSTALLATION_ADMIN_TOKEN"
	return setup
}

func validFleetManagerProvider(provider string) bool {
	return provider == "aws" || provider == "gcp" || provider == "azure"
}

func configuredVersion(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
