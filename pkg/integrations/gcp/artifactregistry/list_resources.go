package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeLocation   = "artifactregistry.location"
	ResourceTypeRepository = "artifactregistry.repository"
)

type repositoryListResponse struct {
	Repositories  []repositoryItem `json:"repositories"`
	NextPageToken string           `json:"nextPageToken"`
}

type repositoryItem struct {
	Name        string `json:"name"`
	Format      string `json:"format"`
	Description string `json:"description"`
}

func ListLocationResources(ctx context.Context, client *gcpcommon.Client) ([]core.IntegrationResource, error) {
	url := fmt.Sprintf(
		"%s/projects/%s/locations",
		artifactRegistryBaseURL,
		client.ProjectID(),
	)

	body, err := client.GetURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("list Artifact Registry locations: %w", err)
	}

	var resp struct {
		Locations []struct {
			LocationID string `json:"locationId"`
		} `json:"locations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse locations response: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(resp.Locations))
	for _, loc := range resp.Locations {
		resources = append(resources, core.IntegrationResource{
			ID:   loc.LocationID,
			Name: loc.LocationID,
		})
	}
	return resources, nil
}

func ListRepositoryResources(ctx context.Context, client *gcpcommon.Client, location string) ([]core.IntegrationResource, error) {
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	var allRepos []repositoryItem
	pageToken := ""

	for {
		url := fmt.Sprintf(
			"%s/projects/%s/locations/%s/repositories",
			artifactRegistryBaseURL,
			client.ProjectID(),
			location,
		)
		if pageToken != "" {
			url += "?pageToken=" + pageToken
		}

		body, err := client.GetURL(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("list Artifact Registry repositories: %w", err)
		}

		var resp repositoryListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse repositories response: %w", err)
		}

		allRepos = append(allRepos, resp.Repositories...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	resources := make([]core.IntegrationResource, 0, len(allRepos))
	for _, repo := range allRepos {
		parts := strings.Split(repo.Name, "/")
		repoName := parts[len(parts)-1]

		resources = append(resources, core.IntegrationResource{
			ID:   repo.Name,
			Name: repoName,
		})
	}
	return resources, nil
}
