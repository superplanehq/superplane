package tpu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeTPULocation        = "tpu.location"
	ResourceTypeTPUAcceleratorType = "tpu.acceleratorType"
	ResourceTypeTPURuntimeVersion  = "tpu.runtimeVersion"
	ResourceTypeTPUNode            = "tpu.node"
)

type locationListResponse struct {
	Locations []struct {
		LocationID  string `json:"locationId"`
		DisplayName string `json:"displayName"`
	} `json:"locations"`
	NextPageToken string `json:"nextPageToken"`
}

type acceleratorTypeListResponse struct {
	AcceleratorTypes []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"acceleratorTypes"`
	NextPageToken string `json:"nextPageToken"`
}

type runtimeVersionListResponse struct {
	RuntimeVersions []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"runtimeVersions"`
	NextPageToken string `json:"nextPageToken"`
}

type nodeListResponse struct {
	Nodes []struct {
		Name            string `json:"name"`
		State           string `json:"state"`
		AcceleratorType string `json:"acceleratorType"`
	} `json:"nodes"`
	NextPageToken string `json:"nextPageToken"`
}

// ListLocationResources lists the zones where Cloud TPU is available.
func ListLocationResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = resolveProject(client, projectID)
	if projectID == "" {
		return nil, nil
	}
	baseURL := fmt.Sprintf("%s/projects/%s/locations?pageSize=500", tpuBaseURL, projectID)

	var resources []core.IntegrationResource
	err := paginate(ctx, client, baseURL, func(data []byte) (string, error) {
		var resp locationListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse TPU locations response: %w", err)
		}
		for _, loc := range resp.Locations {
			if loc.LocationID == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTPULocation,
				ID:   loc.LocationID,
				Name: loc.LocationID,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// ListAcceleratorTypeResources lists the accelerator types (e.g. v2-8) available
// in a location.
func ListAcceleratorTypeResources(ctx context.Context, client Client, projectID, location string) ([]core.IntegrationResource, error) {
	projectID = resolveProject(client, projectID)
	location = strings.TrimSpace(location)
	if projectID == "" || location == "" {
		return nil, nil
	}
	baseURL := fmt.Sprintf("%s/projects/%s/locations/%s/acceleratorTypes?pageSize=500", tpuBaseURL, projectID, location)

	var resources []core.IntegrationResource
	err := paginate(ctx, client, baseURL, func(data []byte) (string, error) {
		var resp acceleratorTypeListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse accelerator types response: %w", err)
		}
		for _, at := range resp.AcceleratorTypes {
			id := at.Type
			if id == "" {
				id = lastSegment(at.Name)
			}
			if id == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTPUAcceleratorType,
				ID:   id,
				Name: id,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// ListRuntimeVersionResources lists the TPU runtime versions available in a
// location.
func ListRuntimeVersionResources(ctx context.Context, client Client, projectID, location string) ([]core.IntegrationResource, error) {
	projectID = resolveProject(client, projectID)
	location = strings.TrimSpace(location)
	if projectID == "" || location == "" {
		return nil, nil
	}
	baseURL := fmt.Sprintf("%s/projects/%s/locations/%s/runtimeVersions?pageSize=500", tpuBaseURL, projectID, location)

	var resources []core.IntegrationResource
	err := paginate(ctx, client, baseURL, func(data []byte) (string, error) {
		var resp runtimeVersionListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse runtime versions response: %w", err)
		}
		for _, rv := range resp.RuntimeVersions {
			id := rv.Version
			if id == "" {
				id = lastSegment(rv.Name)
			}
			if id == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTPURuntimeVersion,
				ID:   id,
				Name: id,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

// ListNodeResources lists every TPU node in the project across all locations,
// using the aggregated "locations/-" parent. The resource ID is the node's full
// resource name so that the Get and Delete components can derive its location
// without a separate location field.
func ListNodeResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = resolveProject(client, projectID)
	if projectID == "" {
		return nil, nil
	}
	baseURL := fmt.Sprintf("%s/projects/%s/locations/-/nodes?pageSize=500", tpuBaseURL, projectID)

	var resources []core.IntegrationResource
	err := paginate(ctx, client, baseURL, func(data []byte) (string, error) {
		var resp nodeListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return "", fmt.Errorf("failed to parse TPU nodes response: %w", err)
		}
		for _, node := range resp.Nodes {
			if node.Name == "" {
				continue
			}
			name := lastSegment(node.Name)
			display := name
			if _, location, _, err := parseNodeName(node.Name); err == nil && location != "" {
				display = fmt.Sprintf("%s (%s)", name, location)
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeTPUNode,
				ID:   node.Name,
				Name: display,
			})
		}
		return resp.NextPageToken, nil
	})
	return resources, err
}

func resolveProject(client Client, projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	return projectID
}

// paginate walks a paginated list endpoint, invoking handle for each page and
// following nextPageToken until it is empty.
func paginate(ctx context.Context, client Client, baseURL string, handle func(data []byte) (string, error)) error {
	pageURL := baseURL
	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return err
		}
		token, err := handle(data)
		if err != nil {
			return err
		}
		if token == "" {
			break
		}
		pageURL = baseURL + "&pageToken=" + url.QueryEscape(token)
	}
	return nil
}
