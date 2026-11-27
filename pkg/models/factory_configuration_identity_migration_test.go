package models_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestWorkspacePublicIDsMigrationPreservesSwappedOwnership(t *testing.T) {
	migration, err := os.ReadFile("../../db/migrations/20261127171730_workspace-yaml-public-ids.up.sql")
	require.NoError(t, err)
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "workspace_ids_" + uuid.New().String()[:8]
	require.NoError(t, tx.Exec("CREATE SCHEMA "+schema).Error)
	require.NoError(t, tx.Exec("SET LOCAL search_path TO "+schema).Error)
	require.NoError(t, tx.Exec(`
		CREATE TABLE factories (
		  id uuid PRIMARY KEY, organization_id uuid, name text,
		  CONSTRAINT factories_organization_id_name_key UNIQUE (organization_id, name)
		);
		CREATE TABLE workflows (
		  id uuid PRIMARY KEY, factory_id uuid REFERENCES factories(id), name text,
		  created_at timestamptz, deleted_at timestamptz
		);
		CREATE TABLE factory_lines (
		  id uuid PRIMARY KEY, factory_id uuid REFERENCES factories(id),
		  stages jsonb NOT NULL DEFAULT '[]', created_at timestamptz
		);
		CREATE TABLE factory_configuration_automations (
		  organization_id uuid, factory_id uuid REFERENCES factories(id),
		  line_id uuid REFERENCES factory_lines(id), stage_key text, manifest_id text,
		  canvas_id uuid REFERENCES workflows(id), binding jsonb,
		  PRIMARY KEY (factory_id, line_id, stage_key, manifest_id)
		);
	`).Error)
	organizationID, factoryID, lineID := uuid.New(), uuid.New(), uuid.New()
	first, second := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, tx.Exec("INSERT INTO factories VALUES (?, ?, 'Workspace')", factoryID, organizationID).Error)
	require.NoError(t, tx.Exec("INSERT INTO factory_lines VALUES (?, ?, '[]', ?)", lineID, factoryID, now).Error)
	require.NoError(t, tx.Exec(`INSERT INTO workflows (id, factory_id, name, created_at) VALUES
		(?, ?, 'Review Task', ?), (?, ?, 'Review-Task', ?)`, first, factoryID, now.Add(-time.Minute), second, factoryID, now).Error)
	// Name-based adoption swaps these two valid, line-scoped public IDs.
	require.NoError(t, tx.Exec(`INSERT INTO factory_configuration_automations VALUES
		(?, ?, ?, 'verify', 'review-task-2', ?, '{"integration_id":"first-connection"}'),
		(?, ?, ?, 'verify', 'review-task', ?, '{"integration_id":"second-connection"}')`,
		organizationID, factoryID, lineID, first, organizationID, factoryID, lineID, second).Error)
	require.NoError(t, tx.Exec(string(migration)).Error)
	var assignments []struct {
		CanvasID   uuid.UUID
		PublicID   string
		ManifestID string
		Binding    string
	}
	require.NoError(t, tx.Raw(`SELECT canvas.id AS canvas_id, canvas.configuration_id AS public_id,
		ownership.manifest_id, ownership.binding::text
		FROM workflows canvas JOIN factory_configuration_automations ownership ON ownership.canvas_id = canvas.id
		ORDER BY canvas.created_at`).Scan(&assignments).Error)
	require.Len(t, assignments, 2)
	require.Equal(t, first, assignments[0].CanvasID)
	require.Equal(t, "review-task", assignments[0].PublicID)
	require.Equal(t, "review-task", assignments[0].ManifestID)
	require.JSONEq(t, `{"integration_id":"first-connection"}`, assignments[0].Binding)
	require.Equal(t, second, assignments[1].CanvasID)
	require.Equal(t, "review-task-2", assignments[1].PublicID)
	require.Equal(t, "review-task-2", assignments[1].ManifestID)
	require.JSONEq(t, `{"integration_id":"second-connection"}`, assignments[1].Binding)
}
