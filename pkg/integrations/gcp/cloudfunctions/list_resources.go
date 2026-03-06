package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeLocation = "cloudfunctions.location"
	ResourceTypeFunction = "cloudfunctions.function"
)

// Locations

type locationListResponse struct {
	Locations     []locationItem `json:"locations"`
	NextPageToken string         `json:"nextPageToken"`
}

type locationItem struct {
	LocationId  string `json:"locationId"`
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

	baseURL := fmt.Sprintf("%s/v2/projects/%s/locations?pageSize=100", cloudFunctionsBaseURL, projectID)
	pageURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list locations: %w", err)
		}

		var resp locationListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse locations response: %w", err)
		}

		for _, loc := range resp.Locations {
			locationID := loc.LocationId
			if locationID == "" {
				continue
			}
			displayName := loc.DisplayName
			if displayName == "" {
				displayName = locationID
			} else if displayName != locationID {
				displayName = fmt.Sprintf("%s (%s)", displayName, locationID)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeLocation,
				ID:   locationID,
				Name: displayName,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageURL = baseURL + "&pageToken=" + url.QueryEscape(resp.NextPageToken)
	}

	return resources, nil
}

// Functions (Cloud Functions v2 + Cloud Run services)

type functionListResponse struct {
	Functions     []functionItem `json:"functions"`
	NextPageToken string         `json:"nextPageToken"`
}

type functionItem struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type serviceListResponse struct {
	Services      []serviceItem `json:"services"`
	NextPageToken string        `json:"nextPageToken"`
}

type serviceItem struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// ListFunctionResources lists Cloud Functions v2 and Cloud Run services for a given location.
// Both are shown because Gen 2 Cloud Functions ("Cloud Run functions") may be deployed
// directly as Cloud Run services and won't appear in the Cloud Functions API.
func ListFunctionResources(ctx context.Context, client Client, projectID string, location string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	location = strings.TrimSpace(location)
	if projectID == "" || location == "" {
		return nil, nil
	}

	var resources []core.IntegrationResource

	// Cloud Functions v2
	cfResources, err := listCloudFunctions(ctx, client, projectID, location)
	if err != nil {
		return nil, err
	}
	resources = append(resources, cfResources...)

	// Cloud Run services
	crResources, err := listCloudRunServices(ctx, client, projectID, location)
	if err != nil {
		return nil, err
	}
	resources = append(resources, crResources...)

	return resources, nil
}

func listCloudFunctions(ctx context.Context, client Client, projectID, location string) ([]core.IntegrationResource, error) {
	baseURL := fmt.Sprintf("%s/v2/projects/%s/locations/%s/functions?pageSize=100", cloudFunctionsBaseURL, projectID, location)
	pageURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions: %w", err)
		}

		var resp functionListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse functions response: %w", err)
		}

		for _, fn := range resp.Functions {
			if fn.Name == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeFunction,
				ID:   fn.Name,
				Name: functionShortName(fn.Name),
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageURL = baseURL + "&pageToken=" + url.QueryEscape(resp.NextPageToken)
	}

	return resources, nil
}

func listCloudRunServices(ctx context.Context, client Client, projectID, location string) ([]core.IntegrationResource, error) {
	baseURL := fmt.Sprintf("%s/v2/projects/%s/locations/%s/services?pageSize=100", cloudRunBaseURL, projectID, location)
	pageURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list Cloud Run services: %w", err)
		}

		var resp serviceListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse Cloud Run services response: %w", err)
		}

		for _, svc := range resp.Services {
			if svc.Name == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeFunction,
				ID:   svc.Name,
				Name: functionShortName(svc.Name),
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageURL = baseURL + "&pageToken=" + url.QueryEscape(resp.NextPageToken)
	}

	return resources, nil
}

func functionShortName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 0 {
		return name
	}
	return parts[len(parts)-1]
}
