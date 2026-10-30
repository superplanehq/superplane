package models_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestOrganizationCodexMigrationPreservesOnlyUnambiguousCredentials(t *testing.T) {
	migration, err := os.ReadFile("../../db/migrations/20261030121001_organization-codex-connections.up.sql")
	require.NoError(t, err)
	// The harness owns the transaction so each old-schema scenario is isolated.
	sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "\nBEGIN;", ""), "\nCOMMIT;", "")
	for _, scenario := range []struct {
		name     string
		statuses []string
		lease    bool
		expected string
	}{
		{name: "single ready", statuses: []string{"ready"}, expected: "ready"},
		{name: "one usable duplicate", statuses: []string{"disconnected", "ready"}, expected: "ready"},
		{name: "ambiguous accounts", statuses: []string{"ready", "ready"}, expected: "reconnect_required"},
		{name: "disconnected", statuses: []string{"disconnected", "disconnected"}, expected: "disconnected"},
		{name: "outstanding lease", statuses: []string{"ready", "ready"}, lease: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			tx := database.DB(t.Context()).Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			require.NoError(t, tx.Exec(`CREATE TEMP TABLE factories (id uuid PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE factory_codex_connections (
 id uuid PRIMARY KEY, organization_id uuid, owner_user_id uuid, factory_id uuid,
 status text, resume_status text DEFAULT '', verification_url text DEFAULT '', verification_code text DEFAULT '',
 account_label text, bundle_ciphertext bytea, credential_version bigint DEFAULT 1, final_checkpoint_version bigint DEFAULT 1,
 reservation_generation bigint DEFAULT 1, revocation_pending_at timestamptz,
 usage_limits jsonb DEFAULT '[{"limitId":"saved"}]', available_models jsonb DEFAULT '[{"model":"saved"}]', usage_observed_at timestamptz DEFAULT now(),
 CONSTRAINT factory_codex_connections_factory_owner_key UNIQUE (factory_id, owner_user_id)
) ON COMMIT DROP;
CREATE TEMP TABLE factory_codex_leases (id uuid, connection_id uuid, purpose text) ON COMMIT DROP;`).Error)
			orgID, ownerID := uuid.New(), uuid.New()
			var readyID uuid.UUID
			for _, status := range scenario.statuses {
				id := uuid.New()
				if status == "ready" {
					readyID = id
				}
				require.NoError(t, tx.Exec(`INSERT INTO factory_codex_connections
 (id, organization_id, owner_user_id, factory_id, status, account_label, bundle_ciphertext)
 VALUES (?, ?, ?, ?, ?, 'saved-account', ?)`, id, orgID, ownerID, uuid.New(), status, []byte("original-encrypted-bundle")).Error)
			}
			if scenario.lease {
				require.NoError(t, tx.Exec("INSERT INTO factory_codex_leases VALUES (?, ?, 'run')", uuid.New(), readyID).Error)
				err := tx.Transaction(func(inner *gorm.DB) error { return inner.Exec(sql).Error })
				require.ErrorContains(t, err, "Drain all Codex leases")
				var retained int64
				require.NoError(t, tx.Table("factory_codex_connections").Where("factory_id IS NOT NULL AND bundle_ciphertext = ?", []byte("original-encrypted-bundle")).Count(&retained).Error)
				assert.EqualValues(t, 2, retained, "a failed migration must leave credentials and schema intact")
				return
			}
			require.NoError(t, tx.Exec(sql).Error)
			var connections []models.FactoryCodexConnection
			require.NoError(t, tx.Find(&connections).Error)
			require.Len(t, connections, 1)
			connection := connections[0]
			assert.Equal(t, scenario.expected, connection.Status)
			if scenario.expected == "ready" {
				assert.Equal(t, readyID, connection.ID)
				assert.Equal(t, []byte("original-encrypted-bundle"), connection.BundleCiphertext)
				assert.NotEmpty(t, connection.UsageLimits)
				return
			}
			assert.Empty(t, connection.BundleCiphertext)
			assert.Empty(t, connection.AccountLabel)
			assert.Empty(t, connection.UsageLimits)
			assert.Empty(t, connection.AvailableModels)
			assert.Nil(t, connection.UsageObservedAt)
		})
	}
}
