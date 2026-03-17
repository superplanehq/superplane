package clouddns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const ResourceTypeManagedZone = "clouddns.managedZone"

type managedZoneListResponse struct {
	ManagedZones  []managedZoneItem `json:"managedZones"`
	NextPageToken string            `json:"nextPageToken"`
}

type managedZoneItem struct {
	Name    string `json:"name"`
	DNSName string `json:"dnsName"`
}

func ListManagedZoneResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf("%s/projects/%s/managedZones?maxResults=500", cloudDNSBaseURL, projectID)
	pageURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list managed zones: %w", err)
		}

		var resp managedZoneListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse managed zones response: %w", err)
		}

		for _, zone := range resp.ManagedZones {
			if zone.Name == "" {
				continue
			}
			displayName := zone.Name
			if zone.DNSName != "" {
				displayName = fmt.Sprintf("%s (%s)", zone.Name, strings.TrimSuffix(zone.DNSName, "."))
			}
			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeManagedZone,
				ID:   zone.Name,
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
