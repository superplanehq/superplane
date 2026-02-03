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

func ensureRepoInMetadata(ctx core.MetadataContext, app core.IntegrationContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	projectIDStr := getProjectFromConfiguration(configuration)
	if projectIDStr == "" {
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
		return fmt.Sprintf("%d", r.ID) == projectIDStr
	})

	if repoIndex == -1 {
		return fmt.Errorf("project %s is not accessible to integration", projectIDStr)
	}

	if nodeMetadata.Repository != nil && fmt.Sprintf("%d", nodeMetadata.Repository.ID) == projectIDStr {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Repository: &appMetadata.Repositories[repoIndex],
	})
}

func getProjectFromConfiguration(c any) string {
	configMap, ok := c.(map[string]any)
	if !ok {
		return ""
	}

	r, ok := configMap["project"]
	if !ok {
		return ""
	}

	project, ok := r.(string)
	if !ok {
		return ""
	}

	return project
}
