package cloudstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeBucket = "cloudstorage.bucket"
)

type bucketListResponse struct {
	Items         []bucketItem `json:"items"`
	NextPageToken string       `json:"nextPageToken"`
}

type bucketItem struct {
	Name         string `json:"name"`
	Location     string `json:"location"`
	StorageClass string `json:"storageClass"`
}

func ListBucketResources(ctx context.Context, client Client, projectID string) ([]core.IntegrationResource, error) {
	if projectID == "" {
		projectID = client.ProjectID()
	}
	if projectID == "" {
		return nil, nil
	}

	baseURL := fmt.Sprintf("%s/b?project=%s&maxResults=100", storageBaseURL, url.QueryEscape(projectID))
	pageURL := baseURL
	var resources []core.IntegrationResource

	for {
		data, err := client.GetURL(ctx, pageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list buckets: %w", err)
		}

		var resp bucketListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse buckets response: %w", err)
		}

		for _, bucket := range resp.Items {
			if bucket.Name == "" {
				continue
			}

			displayName := bucket.Name
			if bucket.Location != "" {
				displayName = fmt.Sprintf("%s (%s)", bucket.Name, bucket.Location)
			}

			resources = append(resources, core.IntegrationResource{
				Type: ResourceTypeBucket,
				ID:   bucket.Name,
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
