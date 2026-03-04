package cloudbuild

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

const (
	ResourceTypeTrigger    = "cloudbuild.trigger"
	ResourceTypeBuild      = "cloudbuild.build"
	ResourceTypeLocation   = "cloudbuild.location"
	ResourceTypeConnection = "cloudbuild.connection"
	ResourceTypeRepository = "cloudbuild.repository"
	ResourceTypeBranch     = "cloudbuild.branch"
	ResourceTypeTag        = "cloudbuild.tag"
)

// Cloud Build Triggers

type triggerListResponse struct {
	Triggers      []triggerItem `json:"triggers"`
	NextPageToken string        `json:"nextPageToken"`
}

type triggerItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ListTriggerResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	baseURL := fmt.Sprintf("%s/projects/%s/triggers", cloudBuildBaseURL, projectID)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed to list triggers: %w", err)
		}

		var resp triggerListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse triggers response: %w", err)
		}

		for _, trigger := range resp.Triggers {
			displayName := trigger.Name
			if displayName == "" {
				displayName = trigger.ID
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTrigger,
				Name: displayName,
				ID:   trigger.ID,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Cloud Build builds

type buildListResponse struct {
	Builds        []buildResourceItem `json:"builds"`
	NextPageToken string              `json:"nextPageToken"`
}

type buildResourceItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CreateTime string `json:"createTime"`
}

func ListBuildResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf("%s/projects/%s/builds?pageSize=50", cloudBuildBaseURL, projectID)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list builds: %w", err)
		}

		var resp buildListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse builds response: %w", err)
		}

		for _, build := range resp.Builds {
			buildID := strings.TrimSpace(build.ID)
			if buildID == "" {
				buildID = buildIDFromName(build.Name)
			}
			if buildID == "" {
				continue
			}

			resourceID := strings.TrimSpace(build.Name)
			if resourceID == "" {
				resourceID = buildID
			}

			displayName := buildResourceLabel(build)
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeBuild,
				Name: displayName,
				ID:   resourceID,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Cloud Build 2nd-gen locations

type locationListResponse struct {
	Locations     []locationResourceItem `json:"locations"`
	NextPageToken string                 `json:"nextPageToken"`
}

type locationResourceItem struct {
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

	baseURL := fmt.Sprintf("https://cloudbuild.googleapis.com/v2/projects/%s/locations?pageSize=100", projectID)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list locations: %w", err)
		}

		var resp locationListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse locations response: %w", err)
		}

		for _, location := range resp.Locations {
			locationID := strings.TrimSpace(location.LocationID)
			if locationID == "" {
				locationID = locationIDFromName(location.Name)
			}
			if locationID == "" {
				continue
			}

			displayName := strings.TrimSpace(location.DisplayName)
			if displayName == "" {
				displayName = locationID
			} else if displayName != locationID {
				displayName = fmt.Sprintf("%s (%s)", displayName, locationID)
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeLocation,
				Name: displayName,
				ID:   locationID,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Cloud Build 2nd-gen connections

type connectionListResponse struct {
	Connections   []connectionResourceItem `json:"connections"`
	NextPageToken string                   `json:"nextPageToken"`
}

type connectionResourceItem struct {
	Name                      string         `json:"name"`
	GitHubConfig              map[string]any `json:"githubConfig"`
	GitHubEnterpriseConfig    map[string]any `json:"githubEnterpriseConfig"`
	GitLabConfig              map[string]any `json:"gitlabConfig"`
	BitbucketCloudConfig      map[string]any `json:"bitbucketCloudConfig"`
	BitbucketDataCenterConfig map[string]any `json:"bitbucketDataCenterConfig"`
	Disabled                  bool           `json:"disabled"`
}

func ListConnectionResources(ctx context.Context, client Client, projectID string, location string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	location = strings.TrimSpace(location)
	if projectID == "" || location == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf(
		"https://cloudbuild.googleapis.com/v2/projects/%s/locations/%s/connections?pageSize=50",
		projectID,
		location,
	)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list connections: %w", err)
		}

		var resp connectionListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse connections response: %w", err)
		}

		for _, connection := range resp.Connections {
			connectionName := strings.TrimSpace(connection.Name)
			connectionID := connectionIDFromName(connectionName)
			if connectionID == "" {
				continue
			}

			displayName := connectionID
			provider := connectionProvider(connection)
			if provider != "" {
				displayName = fmt.Sprintf("%s · %s", displayName, provider)
			}
			if connection.Disabled {
				displayName = displayName + " · disabled"
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeConnection,
				Name: displayName,
				ID:   connectionName,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Cloud Build 2nd-gen repositories

type repositoryListResponse struct {
	Repositories  []repositoryResourceItem `json:"repositories"`
	NextPageToken string                   `json:"nextPageToken"`
}

type repositoryResourceItem struct {
	Name      string `json:"name"`
	RemoteURI string `json:"remoteUri"`
}

func ListRepositoryResources(ctx context.Context, client Client, connectionName string) ([]core.IntegrationResource, error) {
	connectionName = strings.TrimSpace(connectionName)
	if connectionName == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf("https://cloudbuild.googleapis.com/v2/%s/repositories?pageSize=50", connectionName)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		var resp repositoryListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse repositories response: %w", err)
		}

		for _, repository := range resp.Repositories {
			repositoryName := strings.TrimSpace(repository.Name)
			if repositoryName == "" {
				continue
			}

			displayName := strings.TrimSpace(repository.RemoteURI)
			if displayName == "" {
				displayName = repositoryIDFromName(repositoryName)
			}
			if displayName == "" {
				displayName = repositoryName
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeRepository,
				Name: displayName,
				ID:   repositoryName,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

// Cloud Build 2nd-gen git refs

type gitRefsResponse struct {
	RefNames      []string `json:"refNames"`
	NextPageToken string   `json:"nextPageToken"`
}

func ListBranchResources(ctx context.Context, client Client, repositoryName string) ([]core.IntegrationResource, error) {
	return listGitRefResources(ctx, client, repositoryName, "BRANCH", ResourceTypeBranch)
}

func ListTagResources(ctx context.Context, client Client, repositoryName string) ([]core.IntegrationResource, error) {
	return listGitRefResources(ctx, client, repositoryName, "TAG", ResourceTypeTag)
}

func listGitRefResources(
	ctx context.Context,
	client Client,
	repositoryName string,
	refType string,
	resourceType string,
) ([]core.IntegrationResource, error) {
	repositoryName = strings.TrimSpace(repositoryName)
	if repositoryName == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf(
		"https://cloudbuild.googleapis.com/v2/%s:fetchGitRefs?refType=%s&pageSize=100",
		repositoryName,
		refType,
	)
	url := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, url)
		if err != nil {
			if isUnavailable(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to list %s refs: %w", strings.ToLower(refType), err)
		}

		var resp gitRefsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse %s refs response: %w", strings.ToLower(refType), err)
		}

		for _, refName := range resp.RefNames {
			refName = strings.TrimSpace(refName)
			if refName == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: gitRefDisplayName(refName),
				ID:   refName,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		url = withPageToken(baseURL, resp.NextPageToken)
	}

	return resources, nil
}

func buildIDFromName(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func locationIDFromName(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return ""
	}
	return parts[len(parts)-1]
}

func connectionIDFromName(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "connections" {
		return ""
	}
	return parts[len(parts)-1]
}

func repositoryIDFromName(name string) string {
	_, _, _, repositoryID := parseCloudBuildRepositoryName(name)
	return repositoryID
}

func connectionProvider(connection connectionResourceItem) string {
	switch {
	case len(connection.GitHubConfig) > 0:
		return "GitHub"
	case len(connection.GitHubEnterpriseConfig) > 0:
		return "GitHub Enterprise"
	case len(connection.GitLabConfig) > 0:
		return "GitLab"
	case len(connection.BitbucketCloudConfig) > 0:
		return "Bitbucket Cloud"
	case len(connection.BitbucketDataCenterConfig) > 0:
		return "Bitbucket Data Center"
	default:
		return ""
	}
}

func gitRefDisplayName(refName string) string {
	switch {
	case strings.HasPrefix(refName, "refs/heads/"):
		return strings.TrimPrefix(refName, "refs/heads/")
	case strings.HasPrefix(refName, "refs/tags/"):
		return strings.TrimPrefix(refName, "refs/tags/")
	default:
		return refName
	}
}

func buildResourceLabel(build buildResourceItem) string {
	buildID := strings.TrimSpace(build.ID)
	if buildID == "" {
		buildID = buildIDFromName(build.Name)
	}
	if buildID == "" {
		return "Unnamed build"
	}
	return buildID
}
