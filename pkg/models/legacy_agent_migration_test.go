package models_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestLegacyAgentMigrationPreservesPlanningAndUsage(t *testing.T) {
	migration, err := os.ReadFile("../../db/migrations/20261102162713_remove-legacy-built-in-agent.up.sql")
	require.NoError(t, err)
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "agent_removal_" + uuid.New().String()[:8]
	require.NoError(t, tx.Exec("CREATE SCHEMA "+schema).Error)
	require.NoError(t, tx.Exec("SET LOCAL search_path TO "+schema).Error)
	// Populate the pre-removal schema, including the real message dependency.
	require.NoError(t, tx.Exec(`
		CREATE TABLE agent_sessions (id integer PRIMARY KEY, provider_session_id text);
		CREATE TABLE agent_session_messages (session_id integer REFERENCES agent_sessions(id), content text);
		CREATE TABLE workflows (id integer PRIMARY KEY, name text, dismissed_agent_suggestion_ids jsonb);
		CREATE TABLE organizations (id integer PRIMARY KEY, enabled_experimental_features jsonb);
		CREATE TABLE factory_planning_sessions (id integer PRIMARY KEY, draft_description text);
		CREATE TABLE workspace_usage_events (id integer PRIMARY KEY, total_tokens bigint);
		CREATE TABLE casbin_rule (ptype text, v2 text);
		INSERT INTO agent_sessions VALUES (1, 'owned-session');
		INSERT INTO agent_session_messages VALUES (1, 'legacy conversation');
		INSERT INTO workflows VALUES (1, 'Keep canvas', ' ["add-ci"]');
		INSERT INTO organizations VALUES (1, '["claude_managed_agents", "factories", "workspace_skills"]');
		INSERT INTO factory_planning_sessions VALUES (1, 'Keep refinement');
		INSERT INTO workspace_usage_events VALUES (1, 123);
		INSERT INTO casbin_rule VALUES ('p', 'agents'), ('p', 'canvases'), ('g', 'agents');
	`).Error)
	require.NoError(t, tx.Exec(string(migration)).Error)
	var planning string
	require.NoError(t, tx.Raw("SELECT draft_description FROM factory_planning_sessions WHERE id = 1").Scan(&planning).Error)
	require.Equal(t, "Keep refinement", planning)
	var tokens int64
	require.NoError(t, tx.Raw("SELECT total_tokens FROM workspace_usage_events WHERE id = 1").Scan(&tokens).Error)
	require.Equal(t, int64(123), tokens)
	var featureJSON string
	require.NoError(t, tx.Raw("SELECT enabled_experimental_features::text FROM organizations WHERE id = 1").Scan(&featureJSON).Error)
	require.JSONEq(t, `["factories", "workspace_skills"]`, featureJSON)
	var canvasName string
	require.NoError(t, tx.Raw("SELECT name FROM workflows WHERE id = 1").Scan(&canvasName).Error)
	require.Equal(t, "Keep canvas", canvasName)
	var retainedPermissions int64
	require.NoError(t, tx.Raw("SELECT count(*) FROM casbin_rule WHERE ptype = 'p' AND v2 = 'canvases'").Scan(&retainedPermissions).Error)
	require.Equal(t, int64(1), retainedPermissions)
	var legacyPermissions int64
	require.NoError(t, tx.Raw("SELECT count(*) FROM casbin_rule WHERE ptype = 'p' AND v2 = 'agents'").Scan(&legacyPermissions).Error)
	require.Zero(t, legacyPermissions)
	var removed bool
	require.NoError(t, tx.Raw(`SELECT to_regclass('agent_sessions') IS NULL
		AND to_regclass('agent_session_messages') IS NULL
		AND NOT EXISTS (SELECT 1 FROM information_schema.columns
		WHERE table_schema = ? AND table_name = 'workflows' AND column_name = 'dismissed_agent_suggestion_ids')`, schema).Scan(&removed).Error)
	require.True(t, removed)
}
