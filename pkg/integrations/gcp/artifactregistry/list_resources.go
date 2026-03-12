package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	ResourceTypeLocation   = "artifactregistry.location"
	ResourceTypeRepository = "artifactregistry.repository"
	ResourceTypePackage    = "artifactregistry.package"
	ResourceTypeVersion    = "artifactregistry.version"
)

func isUnavailable(err error) bool {
	apiErr, ok := err.(*common.GCPAPIError)
	if !ok {
		return false
	}
	return apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound
}

func withPageToken(baseURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return baseURL
	}
	encoded := url.Values{"pageToken": {token}}.Encode()
	if strings.Contains(baseURL, "?") {
		return baseURL + "&" + encoded
	}
	return baseURL + "?" + encoded
}

func packageIDFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	const marker = "/packages/"
	if idx := strings.Index(name, marker); idx >= 0 {
		id := strings.TrimSpace(name[idx+len(marker):])
		if id != "" {
			return id
		}
	}

	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

// Locations

type locationListResponse struct {
	Locations     []locationItem `json:"locations"`
	NextPageToken string         `json:"nextPageToken"`
}

type locationItem struct {
	Name        string `json:"name"`
	LocationID  string `json:"locationId"`
	DisplayName string `json:"displayName"`
}

func ListLocationResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return nil, nil
	}

	baseURL := listLocationsURL(projectID)
	reqURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, reqURL)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("list locations: %w", err)
		}

		var resp locationListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse locations: %w", err)
		}

		for _, loc := range resp.Locations {
			id := strings.TrimSpace(loc.LocationID)
			if id == "" {
				parts := strings.Split(loc.Name, "/")
				if len(parts) > 0 {
					id = parts[len(parts)-1]
				}
			}
			if id == "" {
				continue
			}
			displayName := strings.TrimSpace(loc.DisplayName)
			if displayName == "" || displayName == id {
				displayName = id
			} else {
				displayName = fmt.Sprintf("%s (%s)", displayName, id)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeLocation,
				Name: displayName,
				ID:   id,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		reqURL = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Repositories

type repositoryListResponse struct {
	Repositories  []repositoryItem `json:"repositories"`
	NextPageToken string           `json:"nextPageToken"`
}

type repositoryItem struct {
	Name        string `json:"name"`
	Format      string `json:"format"`
	Description string `json:"description"`
}

func ListRepositoryResources(ctx context.Context, client Client, projectID, location string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	location = strings.TrimSpace(location)
	if projectID == "" || location == "" {
		return nil, nil
	}

	baseURL := listRepositoriesURL(projectID, location)
	reqURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, reqURL)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("list repositories: %w", err)
		}

		var resp repositoryListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse repositories: %w", err)
		}

		for _, repo := range resp.Repositories {
			repoName := strings.TrimSpace(repo.Name)
			if repoName == "" {
				continue
			}
			parts := strings.Split(repoName, "/")
			repoID := parts[len(parts)-1]
			displayName := repoID
			if repo.Format != "" {
				displayName = fmt.Sprintf("%s · %s", repoID, repo.Format)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeRepository,
				Name: displayName,
				ID:   repoID,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		reqURL = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Packages

type packageListResponse struct {
	Packages      []packageItem `json:"packages"`
	NextPageToken string        `json:"nextPageToken"`
}

type packageItem struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

func ListPackageResources(ctx context.Context, client Client, projectID, location, repository string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	location = strings.TrimSpace(location)
	repository = strings.TrimSpace(repository)
	if projectID == "" || location == "" || repository == "" {
		return nil, nil
	}

	baseURL := listPackagesURL(projectID, location, repository)
	reqURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, reqURL)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("list packages: %w", err)
		}

		var resp packageListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse packages: %w", err)
		}

		for _, pkg := range resp.Packages {
			pkgName := strings.TrimSpace(pkg.Name)
			if pkgName == "" {
				continue
			}
			packageID := packageIDFromName(pkgName)
			if packageID == "" {
				continue
			}

			shortName := packageShortName(packageID)
			displayName := strings.TrimSpace(pkg.DisplayName)
			if displayName == "" {
				displayName = shortName
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypePackage,
				Name: displayName,
				ID:   packageID,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		reqURL = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Versions

type versionListResponse struct {
	Versions      []versionItem `json:"versions"`
	NextPageToken string        `json:"nextPageToken"`
}

type versionItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UpdateTime  string `json:"updateTime"`
}

func ListVersionResources(ctx context.Context, client Client, projectID, location, repository, pkg string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	location = strings.TrimSpace(location)
	repository = strings.TrimSpace(repository)
	pkg = strings.TrimSpace(pkg)
	if projectID == "" || location == "" || repository == "" || pkg == "" {
		return nil, nil
	}

	packageName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s", projectID, location, repository, pkg)
	baseURL := listVersionsURL(packageName)
	reqURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, reqURL)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("list versions: %w", err)
		}

		var resp versionListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse versions: %w", err)
		}

		for _, v := range resp.Versions {
			vName := strings.TrimSpace(v.Name)
			if vName == "" {
				continue
			}
			shortName := versionShortName(vName)
			displayName := shortName
			if v.Description != "" {
				displayName = fmt.Sprintf("%s · %s", shortName, v.Description)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeVersion,
				Name: displayName,
				ID:   shortName,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		reqURL = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}
