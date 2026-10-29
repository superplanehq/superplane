package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
)

const (
	// Fallback when the chart does not set a version and GitHub lookup fails.
	// Keep these equal to componentReleases in the SuperPlane Helm values.
	defaultStableRunnerVersion       = "v0.2.1"
	defaultStableFleetManagerVersion = "v0.1.7"
	fleetManagerReleaseTagPrefix     = "fleet-manager/"
	fleetManagerChartRepositoryName  = "superplanehq/fleet-manager-chart"
	fleetManagerChartRepository      = "oci://ghcr.io/" + fleetManagerChartRepositoryName
	fleetManagerImageRepository      = "ghcr.io/superplanehq/fleet-manager"
	runnerReleaseURLTemplate         = "https://github.com/superplanehq/superplane/releases/download/runner%2F{version}"
	runnerReleaseLookupTimeout       = 3 * time.Second
	runnerReleaseCacheTTL            = 5 * time.Minute
	runnerReleaseFailureBackoff      = 15 * time.Second
	maximumRunnerTagPages            = 10
	runnerReleaseTagPrefix           = "runner/"
	runnerTagRefPrefix               = "refs/tags/" + runnerReleaseTagPrefix
	fleetManagerTagRefPrefix         = "refs/tags/" + fleetManagerReleaseTagPrefix
	runnerTagPageSize                = 100
)

var (
	runnerReleaseAPIBase          = "https://api.github.com/repos/superplanehq/superplane"
	fleetManagerChartRegistryBase = "https://ghcr.io"
	stableRunnerVersionPattern    = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)
	runnerReleaseCacheMu          sync.Mutex
	runnerReleaseRefreshMu        sync.Mutex
	cachedRunnerReleaseVersions   []string
	cachedRunnerReleaseFetchedAt  time.Time
	cachedRunnerReleaseOK         bool
	cachedRunnerReleaseFailedAt   time.Time
	cachedRunnerReleaseFailure    error

	fleetManagerReleaseCacheMu        sync.Mutex
	cachedFleetManagerVersions        []string
	cachedFleetManagerFetchedAt       time.Time
	cachedFleetManagerOK              bool
	cachedFleetManagerReleaseFailedAt time.Time
	cachedFleetManagerReleaseFailure  error
	cachedFleetManagerCharts          = map[string]fleetManagerChartResult{}
)

type fleetManagerChartResult struct {
	exists    bool
	err       error
	fetchedAt time.Time
}

type githubRunnerRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type githubMatchingRef struct {
	Ref string `json:"ref"`
}

type fleetManagerSetupResponse struct {
	DefaultRunnerVersion       string                     `json:"defaultRunnerVersion"`
	DefaultFleetManagerVersion string                     `json:"defaultFleetManagerVersion"`
	DefaultChartVersion        string                     `json:"defaultChartVersion"`
	RunnerReleaseURLTemplate   string                     `json:"runnerReleaseUrlTemplate"`
	ChartRepository            string                     `json:"chartRepository"`
	ImageRepository            string                     `json:"imageRepository"`
	RunnerVersions             []string                   `json:"runnerVersions"`
	StableRunnerVersion        string                     `json:"stableRunnerVersion"`
	InstallationType           string                     `json:"installationType"`
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

	stableRunnerVersion := configuredVersion("RUNNER_STABLE_VERSION", defaultStableRunnerVersion)
	runnerVersion := stableRunnerVersion
	fleetManagerVersion := installationFleetManagerVersion(configuredVersion(
		"FLEET_MANAGER_STABLE_VERSION",
		defaultStableFleetManagerVersion,
	))
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

	installVersion, runnerVersions := runnerVersionsForSetup(runnerVersion)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fleetManagerSetupResponse{
		DefaultRunnerVersion:       installVersion,
		DefaultFleetManagerVersion: fleetManagerVersion,
		DefaultChartVersion:        strings.TrimPrefix(fleetManagerVersion, "v"),
		RunnerReleaseURLTemplate:   runnerReleaseURLTemplate,
		ChartRepository:            fleetManagerChartRepository,
		ImageRepository:            fleetManagerImageRepository,
		RunnerVersions:             runnerVersions,
		StableRunnerVersion:        stableRunnerVersion,
		InstallationType:           os.Getenv("SUPERPLANE_INSTALLATION_TYPE"),
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

func runnerVersionsForSetup(stableVersion string) (string, []string) {
	configured := configuredRunnerVersions(stableVersion)
	published, err := publishedRunnerVersions()
	if err != nil {
		return stableVersion, configured
	}
	return installationRunnerVersion(stableVersion, published), mergeRunnerVersions(configured, published)
}

func installationFleetManagerVersion(configured string) string {
	published, err := publishedFleetManagerVersions()
	if err != nil {
		return configured
	}

	candidates := append([]string(nil), published...)
	if !slices.Contains(candidates, configured) {
		exists, chartErr := fleetManagerChartExists(configured)
		if chartErr == nil && exists {
			candidates = append(candidates, configured)
		}
	}
	if len(candidates) == 0 {
		return configured
	}
	return installationRunnerVersion(candidates[0], candidates)
}

func publishedFleetManagerVersions() ([]string, error) {
	fleetManagerReleaseCacheMu.Lock()
	if cachedFleetManagerOK && time.Since(cachedFleetManagerFetchedAt) < runnerReleaseCacheTTL {
		versions := append([]string(nil), cachedFleetManagerVersions...)
		fleetManagerReleaseCacheMu.Unlock()
		return versions, nil
	}
	if cachedFleetManagerReleaseFailure != nil && time.Since(cachedFleetManagerReleaseFailedAt) < runnerReleaseFailureBackoff {
		err := cachedFleetManagerReleaseFailure
		fleetManagerReleaseCacheMu.Unlock()
		return nil, err
	}
	fleetManagerReleaseCacheMu.Unlock()

	versions, err := fetchPublishedFleetManagerVersions()
	fleetManagerReleaseCacheMu.Lock()
	defer fleetManagerReleaseCacheMu.Unlock()
	if err != nil {
		if cachedFleetManagerOK {
			cachedFleetManagerFetchedAt = time.Now()
			return append([]string(nil), cachedFleetManagerVersions...), nil
		}
		cachedFleetManagerReleaseFailure = err
		cachedFleetManagerReleaseFailedAt = time.Now()
		return nil, err
	}
	cachedFleetManagerVersions = append([]string(nil), versions...)
	cachedFleetManagerFetchedAt = time.Now()
	cachedFleetManagerOK = true
	cachedFleetManagerReleaseFailure = nil
	cachedFleetManagerReleaseFailedAt = time.Time{}
	return versions, nil
}

func fetchPublishedFleetManagerVersions() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runnerReleaseLookupTimeout)
	defer cancel()

	client := &http.Client{Timeout: runnerReleaseLookupTimeout}
	candidates, err := listStableTagVersions(ctx, client, fleetManagerReleaseTagPrefix, fleetManagerTagRefPrefix)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return runnerVersionNewer(candidates[i], candidates[j])
	})

	token, err := fetchFleetManagerRegistryToken(ctx, client)
	if err != nil {
		return nil, err
	}

	var verificationErr error
	for _, version := range candidates {
		release, found, releaseErr := fetchRunnerReleaseByTag(ctx, client, fleetManagerReleaseTagPrefix+version)
		if releaseErr != nil {
			verificationErr = releaseErr
			continue
		}
		if !found || release.Draft || release.Prerelease {
			continue
		}
		confirmed, ok := strings.CutPrefix(release.TagName, fleetManagerReleaseTagPrefix)
		if !ok || !stableRunnerVersionPattern.MatchString(confirmed) || confirmed != version {
			continue
		}
		exists, chartErr := fleetManagerChartPublished(ctx, client, token, version)
		storeFleetManagerChart(version, exists, chartErr)
		if chartErr != nil {
			verificationErr = chartErr
			continue
		}
		if !exists {
			continue
		}
		return []string{version}, nil
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return nil, nil
}

func fleetManagerChartExists(version string) (bool, error) {
	if exists, ok, err := freshFleetManagerChart(version); ok {
		return exists, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), runnerReleaseLookupTimeout)
	defer cancel()

	client := &http.Client{Timeout: runnerReleaseLookupTimeout}
	token, err := fetchFleetManagerRegistryToken(ctx, client)
	if err != nil {
		storeFleetManagerChart(version, false, err)
		return false, err
	}
	exists, err := fleetManagerChartPublished(ctx, client, token, version)
	storeFleetManagerChart(version, exists, err)
	return exists, err
}

func freshFleetManagerChart(version string) (bool, bool, error) {
	fleetManagerReleaseCacheMu.Lock()
	defer fleetManagerReleaseCacheMu.Unlock()

	result, ok := cachedFleetManagerCharts[version]
	if !ok {
		return false, false, nil
	}
	if result.err != nil {
		if time.Since(result.fetchedAt) < runnerReleaseFailureBackoff {
			return false, true, result.err
		}
		return false, false, nil
	}
	if time.Since(result.fetchedAt) < runnerReleaseCacheTTL {
		return result.exists, true, nil
	}
	return false, false, nil
}

func storeFleetManagerChart(version string, exists bool, err error) {
	fleetManagerReleaseCacheMu.Lock()
	defer fleetManagerReleaseCacheMu.Unlock()
	cachedFleetManagerCharts[version] = fleetManagerChartResult{
		exists:    exists,
		err:       err,
		fetchedAt: time.Now(),
	}
}

func fetchFleetManagerRegistryToken(ctx context.Context, client *http.Client) (string, error) {
	tokenURL := fleetManagerChartRegistryBase + "/token?scope=repository:" + fleetManagerChartRepositoryName + ":pull"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SuperPlane")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected registry token status %d", resp.StatusCode)
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Token == "" {
		return "", errors.New("registry token is empty")
	}
	return payload.Token, nil
}

func fleetManagerChartPublished(ctx context.Context, client *http.Client, token, version string) (bool, error) {
	manifestURL := fleetManagerChartRegistryBase + "/v2/" + fleetManagerChartRepositoryName + "/manifests/" + url.PathEscape(strings.TrimPrefix(version, "v"))
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Set("User-Agent", "SuperPlane")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected chart status %d", resp.StatusCode)
	}
}

func listStableTagVersions(ctx context.Context, client *http.Client, tagPrefix, refPrefix string) ([]string, error) {
	nextURL := runnerReleaseAPIBase + "/git/matching-refs/tags/" + tagPrefix + "?per_page=" + fmt.Sprint(runnerTagPageSize)
	seen := map[string]struct{}{}
	versions := make([]string, 0)

	for page := 0; page < maximumRunnerTagPages && nextURL != ""; page++ {
		refs, nextPageURL, err := fetchRunnerTagPage(ctx, client, nextURL)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			version, ok := strings.CutPrefix(ref.Ref, refPrefix)
			if !ok || !stableRunnerVersionPattern.MatchString(version) {
				continue
			}
			if _, exists := seen[version]; exists {
				continue
			}
			seen[version] = struct{}{}
			versions = append(versions, version)
		}
		if nextPageURL == "" {
			return versions, nil
		}
		nextURL = nextPageURL
	}
	if nextURL != "" {
		return nil, fmt.Errorf("tag list for %s exceeded %d pages", tagPrefix, maximumRunnerTagPages)
	}
	return versions, nil
}

func resetFleetManagerReleaseCache() {
	fleetManagerReleaseCacheMu.Lock()
	defer fleetManagerReleaseCacheMu.Unlock()
	cachedFleetManagerVersions = nil
	cachedFleetManagerFetchedAt = time.Time{}
	cachedFleetManagerOK = false
	cachedFleetManagerReleaseFailure = nil
	cachedFleetManagerReleaseFailedAt = time.Time{}
	cachedFleetManagerCharts = map[string]fleetManagerChartResult{}
}

func installationRunnerVersion(stableVersion string, published []string) string {
	candidates := make([]string, 0, len(published)+1)
	candidates = append(candidates, published...)
	candidates = append(candidates, stableVersion)
	newest := mergeRunnerVersions(nil, candidates)
	if len(newest) == 0 {
		return stableVersion
	}
	return newest[0]
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
	if versions, ok, err := reusableRunnerReleaseResult(); ok {
		return versions, err
	}

	runnerReleaseRefreshMu.Lock()
	defer runnerReleaseRefreshMu.Unlock()
	if versions, ok, err := reusableRunnerReleaseResult(); ok {
		return versions, err
	}

	versions, err := fetchPublishedRunnerVersions()
	if err != nil {
		if stale, ok := cachedRunnerVersions(); ok {
			touchRunnerReleaseCache()
			return stale, nil
		}
		rememberRunnerReleaseFailure(err)
		return nil, err
	}
	storeCachedRunnerVersions(versions)
	return versions, nil
}

func reusableRunnerReleaseResult() ([]string, bool, error) {
	if versions, ok := freshCachedRunnerVersions(); ok {
		return versions, true, nil
	}
	if err := freshRunnerReleaseFailure(); err != nil {
		return nil, true, err
	}
	return nil, false, nil
}

func fetchPublishedRunnerVersions() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runnerReleaseLookupTimeout)
	defer cancel()

	client := &http.Client{Timeout: runnerReleaseLookupTimeout}
	candidates, err := listStableRunnerTagVersions(ctx, client)
	if err != nil {
		return nil, err
	}

	published := make([]string, 0, len(candidates))
	for _, version := range candidates {
		release, found, err := fetchRunnerReleaseByTag(ctx, client, runnerReleaseTagPrefix+version)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		confirmed, ok := stableRunnerVersionFromRelease(release)
		if !ok || confirmed != version {
			continue
		}
		published = append(published, version)
	}
	return published, nil
}

func listStableRunnerTagVersions(ctx context.Context, client *http.Client) ([]string, error) {
	nextURL := runnerTagRefsURL()
	seen := map[string]struct{}{}
	versions := make([]string, 0)

	for page := 0; page < maximumRunnerTagPages && nextURL != ""; page++ {
		refs, nextPageURL, err := fetchRunnerTagPage(ctx, client, nextURL)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			version, ok := stableRunnerVersionFromRef(ref.Ref)
			if !ok {
				continue
			}
			if _, exists := seen[version]; exists {
				continue
			}
			seen[version] = struct{}{}
			versions = append(versions, version)
		}
		if nextPageURL == "" {
			return versions, nil
		}
		nextURL = nextPageURL
	}
	if nextURL != "" {
		return nil, fmt.Errorf("runner tag list exceeded %d pages", maximumRunnerTagPages)
	}
	return versions, nil
}

func runnerTagRefsURL() string {
	return runnerReleaseAPIBase + "/git/matching-refs/tags/" + runnerReleaseTagPrefix + "?per_page=" + fmt.Sprint(runnerTagPageSize)
}

func fetchRunnerTagPage(ctx context.Context, client *http.Client, pageURL string) ([]githubMatchingRef, string, error) {
	body, nextPageURL, err := fetchGitHubJSON(ctx, client, pageURL)
	if err != nil {
		return nil, "", err
	}
	var refs []githubMatchingRef
	if err := json.Unmarshal(body, &refs); err != nil {
		return nil, "", err
	}
	return refs, nextPageURL, nil
}

func fetchRunnerReleaseByTag(ctx context.Context, client *http.Client, tag string) (githubRunnerRelease, bool, error) {
	pageURL := runnerReleaseAPIBase + "/releases/tags/" + url.PathEscape(tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return githubRunnerRelease{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "SuperPlane")

	resp, err := client.Do(req)
	if err != nil {
		return githubRunnerRelease{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return githubRunnerRelease{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return githubRunnerRelease{}, false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release githubRunnerRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRunnerRelease{}, false, err
	}
	return release, true, nil
}

func fetchGitHubJSON(ctx context.Context, client *http.Client, pageURL string) ([]byte, string, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return body, nextRunnerReleasePageURL(resp.Header.Get("Link")), nil
}

func freshCachedRunnerVersions() ([]string, bool) {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	if !cachedRunnerReleaseOK || time.Since(cachedRunnerReleaseFetchedAt) >= runnerReleaseCacheTTL {
		return nil, false
	}
	return append([]string(nil), cachedRunnerReleaseVersions...), true
}

func cachedRunnerVersions() ([]string, bool) {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	if !cachedRunnerReleaseOK {
		return nil, false
	}
	return append([]string(nil), cachedRunnerReleaseVersions...), true
}

func freshRunnerReleaseFailure() error {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	if cachedRunnerReleaseFailure == nil || time.Since(cachedRunnerReleaseFailedAt) >= runnerReleaseFailureBackoff {
		return nil
	}
	return cachedRunnerReleaseFailure
}

func rememberRunnerReleaseFailure(err error) {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	cachedRunnerReleaseFailure = err
	cachedRunnerReleaseFailedAt = time.Now()
}

func storeCachedRunnerVersions(versions []string) {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	cachedRunnerReleaseVersions = append([]string(nil), versions...)
	cachedRunnerReleaseFetchedAt = time.Now()
	cachedRunnerReleaseOK = true
	cachedRunnerReleaseFailure = nil
	cachedRunnerReleaseFailedAt = time.Time{}
}

func touchRunnerReleaseCache() {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	if !cachedRunnerReleaseOK {
		return
	}
	cachedRunnerReleaseFetchedAt = time.Now()
}

func resetRunnerReleaseCache() {
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	cachedRunnerReleaseVersions = nil
	cachedRunnerReleaseFetchedAt = time.Time{}
	cachedRunnerReleaseOK = false
	cachedRunnerReleaseFailure = nil
	cachedRunnerReleaseFailedAt = time.Time{}
}

func stableRunnerVersionFromRef(ref string) (string, bool) {
	version, ok := strings.CutPrefix(ref, runnerTagRefPrefix)
	if !ok || !stableRunnerVersionPattern.MatchString(version) {
		return "", false
	}
	return version, true
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
