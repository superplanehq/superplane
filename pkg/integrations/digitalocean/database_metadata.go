package digitalocean

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type DatabaseNodeMetadata struct {
	DatabaseClusterID   string `json:"databaseClusterId" mapstructure:"databaseClusterId"`
	DatabaseClusterName string `json:"databaseClusterName" mapstructure:"databaseClusterName"`
	DatabaseName        string `json:"databaseName" mapstructure:"databaseName"`
}

func resolveDatabaseClusterMetadata(ctx core.SetupContext, clusterID string) error {
	if strings.Contains(clusterID, "{{") {
		return ctx.Metadata.Set(DatabaseNodeMetadata{
			DatabaseClusterID:   clusterID,
			DatabaseClusterName: clusterID,
		})
	}

	var existing DatabaseNodeMetadata
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

	// Do not carry DatabaseName across cluster changes: resolveDatabaseMetadata
	// would treat (newClusterID, oldName) as cached and skip listing DBs on the new cluster.
	preservedDBName := existing.DatabaseName
	if existing.DatabaseClusterID != "" && existing.DatabaseClusterID != clusterID {
		preservedDBName = ""
	}

	return ctx.Metadata.Set(DatabaseNodeMetadata{
		DatabaseClusterID:   clusterID,
		DatabaseClusterName: clusterName,
		DatabaseName:        preservedDBName,
	})
}

func resolveDatabaseMetadata(ctx core.SetupContext, clusterID, databaseName string) error {
	if err := resolveDatabaseClusterMetadata(ctx, clusterID); err != nil {
		return err
	}

	if strings.Contains(databaseName, "{{") {
		var existing DatabaseNodeMetadata
		_ = mapstructure.Decode(ctx.Metadata.Get(), &existing)
		existing.DatabaseName = databaseName
		return ctx.Metadata.Set(existing)
	}

	var existing DatabaseNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil {
		if existing.DatabaseClusterID == clusterID && existing.DatabaseName == databaseName {
			return nil
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	databases, err := client.ListDatabases(clusterID)
	if err != nil {
		return fmt.Errorf("failed to list databases for cluster %q: %w", clusterID, err)
	}

	found := false
	for _, database := range databases {
		if database.Name != databaseName {
			continue
		}
		found = true
		break
	}

	if !found {
		return fmt.Errorf("database %q was not found in cluster %q", databaseName, clusterID)
	}

	existing.DatabaseClusterID = clusterID
	if existing.DatabaseClusterName == "" {
		existing.DatabaseClusterName = clusterID
	}
	existing.DatabaseName = databaseName

	return ctx.Metadata.Set(existing)
}
