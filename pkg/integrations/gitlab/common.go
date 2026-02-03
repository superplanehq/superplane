package gitlab

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

func ensureRepoInMetadata(ctx core.MetadataContext, app core.IntegrationContext, projectID string) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if projectID == "" {
		return fmt.Errorf("project is required")
	}

	//
	// Validate that the app has access to this repository
	//
	var appMetadata Metadata
	if err := mapstructure.Decode(app.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return fmt.Sprintf("%d", r.ID) == projectID
	})

	if repoIndex == -1 {
		return fmt.Errorf("project %s is not accessible to integration", projectID)
	}

	if nodeMetadata.Repository != nil && fmt.Sprintf("%d", nodeMetadata.Repository.ID) == projectID {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Repository: &appMetadata.Repositories[repoIndex],
	})
}
