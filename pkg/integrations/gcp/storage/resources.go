package storage

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

// ResourceTypeBucket lists the Cloud Storage buckets in the project, so the
// get/delete pickers can offer the existing buckets instead of free text.
const ResourceTypeBucket = "storageBucket"

// ListBucketResources lists Cloud Storage buckets for the bucket dropdown.
func ListBucketResources(ctx context.Context, client Client) ([]core.IntegrationResource, error) {
	buckets, err := ListBuckets(ctx, client, client.ProjectID())
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(buckets))
	for _, b := range buckets {
		label := b.Name
		if b.Location != "" {
			label = fmt.Sprintf("%s (%s)", b.Name, b.Location)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeBucket, Name: label, ID: b.Name})
	}
	return out, nil
}
