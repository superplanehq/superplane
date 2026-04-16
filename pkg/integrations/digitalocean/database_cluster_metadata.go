package digitalocean

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type DatabaseClusterNodeMetadata struct {
	DatabaseClusterID   string `json:"databaseClusterId" mapstructure:"databaseClusterId"`
	DatabaseClusterName string `json:"databaseClusterName" mapstructure:"databaseClusterName"`
}

func resolveDatabaseClusterMetadata(ctx core.SetupContext, clusterID string) error {
	if strings.Contains(clusterID, "{{") {
		return ctx.Metadata.Set(DatabaseClusterNodeMetadata{
			DatabaseClusterID:   clusterID,
			DatabaseClusterName: clusterID,
		})
	}

	var existing DatabaseClusterNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.DatabaseClusterID == clusterID && existing.DatabaseClusterName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	clusters, err := client.ListDatabaseClusters()
	if err != nil {
		return fmt.Errorf("failed to list database clusters: %w", err)
	}

	clusterName := clusterID
	found := false
	for _, cluster := range clusters {
		if cluster.ID != clusterID {
			continue
		}
		found = true
		if cluster.Name != "" {
			clusterName = cluster.Name
		}
		break
	}

	if !found {
		return fmt.Errorf("database cluster %q was not found", clusterID)
	}

	return ctx.Metadata.Set(DatabaseClusterNodeMetadata{
		DatabaseClusterID:   clusterID,
		DatabaseClusterName: clusterName,
	})
}
