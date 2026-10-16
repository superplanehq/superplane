package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultStableRunnerVersion       = "v0.1.0"
	defaultStableFleetManagerVersion = "v0.1.0"
	fleetManagerChartRepository      = "oci://ghcr.io/superplanehq/fleet-manager-chart"
	fleetManagerImageRepository      = "ghcr.io/superplanehq/fleet-manager"
	runnerReleaseURLTemplate         = "https://github.com/superplanehq/superplane/releases/download/runner%2F{version}"
	runnerReleaseLookupTimeout       = 3 * time.Second
	maximumRunnerReleasePages        = 10
	runnerReleaseTagPrefix           = "runner/"
)

var (
	runnerReleasesURL          = "https://api.github.com/repos/superplanehq/superplane/releases?per_page=100"
	stableRunnerVersionPattern = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)
)

type githubRunnerRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type fleetManagerSetupResponse struct {
	DefaultRunnerVersion       string                     `json:"defaultRunnerVersion"`
	DefaultFleetManagerVersion string                     `json:"defaultFleetManagerVersion"`
	DefaultChartVersion        string                     `json:"defaultChartVersion"`
	RunnerReleaseURLTemplate   string                     `json:"runnerReleaseUrlTemplate"`
	ChartRepository            string                     `json:"chartRepository"`
	ImageRepository            string                     `json:"imageRepository"`
	RunnerVersions             []string                   `json:"runnerVersions"`
	FleetPresets               []fleetManagerFleetPreset  `json:"fleetPresets"`
	InfrastructureDefaults     fleetManagerInfrastructure `json:"infrastructureDefaults"`
}

type fleetManagerInfrastructure struct {
	Source             string         `json:"source"`
	Provider           *string        `json:"provider"`
	ManagerID          string         `json:"managerId"`
	Namespace          string         `json:"namespace"`
	ServiceAccountName string         `json:"serviceAccountName"`
	Fields             map[string]any `json:"fields"`
}

type fleetManagerSetupConfig struct {
	Source             string         `json:"source"`
	Provider           *string        `json:"provider"`
	ManagerID          string         `json:"managerId"`
	Namespace          string         `json:"namespace"`
	ServiceAccountName string         `json:"serviceAccountName"`
	Infrastructure     map[string]any `json:"infrastructure"`
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
		RunnerVersions:             runnerVersionsForSetup(runnerVersion),
		FleetPresets:               presets,
		InfrastructureDefaults: fleetManagerInfrastructure{
			Source:             setup.Source,
			Provider:           setup.Provider,
			ManagerID:          setup.ManagerID,
			Namespace:          setup.Namespace,
			ServiceAccountName: setup.ServiceAccountName,
			Fields:             setup.Infrastructure,
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
	return setup
}

func runnerVersionsForSetup(stableVersion string) []string {
	configured := configuredRunnerVersions(stableVersion)
	published, err := publishedRunnerVersions()
	if err != nil {
		return configured
	}
	return mergeRunnerVersions(configured, published)
}

func configuredRunnerVersions(stableVersion string) []string {
	versions := make([]string, 0, 1)
	seen := map[string]bool{}
	add := func(version string) {
		version = strings.TrimSpace(version)
		if version == "" || seen[version] {
			return
		}
		seen[version] = true
		versions = append(versions, version)
	}
	add(stableVersion)
	for _, version := range strings.Split(os.Getenv("RUNNER_SUPPORTED_VERSIONS"), ",") {
		add(version)
	}
	return versions
}

func publishedRunnerVersions() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runnerReleaseLookupTimeout)
	defer cancel()

	client := &http.Client{Timeout: runnerReleaseLookupTimeout}
	nextURL := runnerReleasesURL
	seen := map[string]struct{}{}
	versions := make([]string, 0)

	for page := 0; page < maximumRunnerReleasePages && nextURL != ""; page++ {
		releases, nextPageURL, err := fetchRunnerReleasePage(ctx, client, nextURL)
		if err != nil {
			return nil, err
		}
		for _, release := range releases {
			version, ok := stableRunnerVersionFromRelease(release)
			if !ok {
				continue
			}
			if _, exists := seen[version]; exists {
				continue
			}
			seen[version] = struct{}{}
			versions = append(versions, version)
		}
		nextURL = nextPageURL
	}

	return versions, nil
}

func fetchRunnerReleasePage(ctx context.Context, client *http.Client, pageURL string) ([]githubRunnerRelease, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SuperPlane")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var releases []githubRunnerRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, "", err
	}
	return releases, nextRunnerReleasePageURL(resp.Header.Get("Link")), nil
}

func nextRunnerReleasePageURL(linkHeader string) string {
	for link := range strings.SplitSeq(linkHeader, ",") {
		if !strings.Contains(link, `rel="next"`) {
			continue
		}
		start := strings.IndexByte(link, '<')
		end := strings.IndexByte(link, '>')
		if start >= 0 && end > start {
			return link[start+1 : end]
		}
	}
	return ""
}

func stableRunnerVersionFromRelease(release githubRunnerRelease) (string, bool) {
	if release.Draft || release.Prerelease {
		return "", false
	}
	version, ok := strings.CutPrefix(release.TagName, runnerReleaseTagPrefix)
	if !ok || !stableRunnerVersionPattern.MatchString(version) {
		return "", false
	}
	return version, true
}

func mergeRunnerVersions(configured, published []string) []string {
	seen := map[string]struct{}{}
	versions := make([]string, 0, len(configured)+len(published))
	for _, version := range append(append([]string{}, published...), configured...) {
		version = strings.TrimSpace(version)
		if version == "" {
			continue
		}
		if _, exists := seen[version]; exists {
			continue
		}
		seen[version] = struct{}{}
		versions = append(versions, version)
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return runnerVersionNewer(versions[i], versions[j])
	})
	return versions
}

func runnerVersionNewer(left, right string) bool {
	leftMajor, leftMinor, leftPatch, leftOK := parseStableRunnerVersion(left)
	rightMajor, rightMinor, rightPatch, rightOK := parseStableRunnerVersion(right)
	if leftOK != rightOK {
		return leftOK
	}
	if !leftOK {
		return left > right
	}
	if leftMajor != rightMajor {
		return leftMajor > rightMajor
	}
	if leftMinor != rightMinor {
		return leftMinor > rightMinor
	}
	return leftPatch > rightPatch
}

func parseStableRunnerVersion(version string) (int, int, int, bool) {
	if !stableRunnerVersionPattern.MatchString(version) {
		return 0, 0, 0, false
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "v%d.%d.%d", &major, &minor, &patch); err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
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
