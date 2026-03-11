package artifactregistry

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	artifactRegistryBaseURL  = "https://artifactregistry.googleapis.com/v1"
	containerAnalysisBaseURL = "https://containeranalysis.googleapis.com/v1"
)

// Client is the interface used by Artifact Registry components to call the API.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)
)

func SetClientFactory(fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp artifactregistry: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}

func listLocationsURL(projectID string) string {
	return fmt.Sprintf("%s/projects/%s/locations?pageSize=100", artifactRegistryBaseURL, projectID)
}

func listRepositoriesURL(projectID, location string) string {
	return fmt.Sprintf("%s/projects/%s/locations/%s/repositories?pageSize=100", artifactRegistryBaseURL, projectID, location)
}

func listPackagesURL(projectID, location, repository string) string {
	return fmt.Sprintf("%s/projects/%s/locations/%s/repositories/%s/packages?pageSize=100", artifactRegistryBaseURL, projectID, location, repository)
}

func listVersionsURL(packageName string) string {
	return fmt.Sprintf("%s/%s/versions?pageSize=100&orderBy=updateTime+desc", artifactRegistryBaseURL, packageName)
}

func getVersionURL(packageName, version string) string {
	return fmt.Sprintf("%s/%s/versions/%s", artifactRegistryBaseURL, packageName, version)
}

func listOccurrencesURL(projectID, resourceFilter string) string {
	base := fmt.Sprintf("%s/projects/%s/occurrences?pageSize=100", containerAnalysisBaseURL, projectID)
	if resourceFilter != "" {
		return base + "&filter=" + url.QueryEscape(resourceFilter)
	}
	return base
}

// packageShortName extracts the package name from a full resource name.
func packageShortName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

// versionShortName extracts the version from a full resource name.
func versionShortName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}

// parseArtifactResourceURL parses a GCP Artifact Registry resource URL into its components.
// Accepts URLs in the form:
//
//	https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/PACKAGE@sha256:DIGEST
//	https://LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/PACKAGE:TAG
func parseArtifactResourceURL(rawURL string) (location, repository, pkg, version string, err error) {
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")

	host, path, ok := strings.Cut(rawURL, "/")
	if !ok {
		return "", "", "", "", fmt.Errorf("invalid resource URL: missing path")
	}

	const dockerSuffix = "-docker.pkg.dev"
	if !strings.HasSuffix(host, dockerSuffix) {
		return "", "", "", "", fmt.Errorf("invalid resource URL: expected host ending in -docker.pkg.dev, got %q", host)
	}
	location = strings.TrimSuffix(host, dockerSuffix)

	// path: PROJECT/REPOSITORY/PACKAGE@sha256:DIGEST
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("invalid resource URL: expected PROJECT/REPOSITORY/PACKAGE in path")
	}

	repository = parts[1]
	packageRef := parts[2]

	if idx := strings.LastIndex(packageRef, "@"); idx >= 0 {
		pkg = packageRef[:idx]
		version = packageRef[idx+1:] // sha256:DIGEST
	} else if idx := strings.LastIndex(packageRef, ":"); idx >= 0 {
		pkg = packageRef[:idx]
		version = packageRef[idx+1:] // TAG
	} else {
		return "", "", "", "", fmt.Errorf("invalid resource URL: package reference must include @digest or :tag")
	}

	if location == "" || repository == "" || pkg == "" || version == "" {
		return "", "", "", "", fmt.Errorf("invalid resource URL: could not extract all components")
	}

	return location, repository, pkg, version, nil
}
