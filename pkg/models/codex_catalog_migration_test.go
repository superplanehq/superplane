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

func TestCodexCatalogMigrationPreservesLeasesAndEnforcesScopeAndUniqueness(t *testing.T) {
	migration, err := os.ReadFile("../../db/migrations/20261031144703_add-codex-catalog-leases.up.sql")
	require.NoError(t, err)
	sql := strings.ReplaceAll(strings.ReplaceAll(string(migration), "BEGIN;", ""), "COMMIT;", "")
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Exec(`CREATE TEMP TABLE factory_codex_leases (
 id uuid PRIMARY KEY, connection_id uuid NOT NULL, task_id uuid NOT NULL,
 generation bigint NOT NULL, purpose varchar(16) NOT NULL, factory_id uuid,
 secret_hash text NOT NULL DEFAULT '', secret_ciphertext bytea, revoked_at timestamptz,
 CONSTRAINT factory_codex_leases_task_key UNIQUE (connection_id, task_id),
 CONSTRAINT factory_codex_leases_purpose_check CHECK (purpose IN ('login', 'run')),
 CONSTRAINT factory_codex_leases_scope_check CHECK ((purpose = 'login' AND factory_id IS NULL) OR (purpose = 'run' AND factory_id IS NOT NULL))
) ON COMMIT DROP;
CREATE UNIQUE INDEX factory_codex_leases_secret_idx ON factory_codex_leases (secret_hash) WHERE secret_hash <> '';
CREATE UNIQUE INDEX factory_codex_leases_login_idx ON factory_codex_leases (connection_id) WHERE purpose = 'login';`).Error)
	connectionID := uuid.New()
	loginID, runID := uuid.New(), uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO factory_codex_leases
 (id, connection_id, task_id, generation, purpose, factory_id, secret_hash, secret_ciphertext)
 VALUES (?, ?, ?, 7, 'login', NULL, '', NULL), (?, ?, ?, 8, 'run', ?, 'run-secret', ?)`,
		loginID, connectionID, uuid.New(), runID, connectionID, uuid.New(), uuid.New(), []byte("encrypted-run-secret")).Error)
	require.NoError(t, tx.Exec(sql).Error)
	var preserved []models.FactoryCodexLease
	require.NoError(t, tx.Order("generation").Find(&preserved).Error)
	require.Len(t, preserved, 2)
	assert.Equal(t, loginID, preserved[0].ID)
	assert.Equal(t, models.CodexLeasePurposeLogin, preserved[0].Purpose)
	assert.EqualValues(t, 7, preserved[0].Generation)
	assert.Equal(t, runID, preserved[1].ID)
	assert.Equal(t, "run-secret", preserved[1].SecretHash)
	assert.Equal(t, []byte("encrypted-run-secret"), preserved[1].SecretCiphertext)
	assert.NotNil(t, preserved[1].FactoryID)
	catalogID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO factory_codex_leases
 (id, connection_id, task_id, generation, purpose, secret_hash)
 VALUES (?, ?, ?, 9, 'catalog', 'catalog-secret')`, catalogID, connectionID, uuid.New()).Error)
	require.NoError(t, tx.Exec("UPDATE factory_codex_leases SET revoked_at = now() WHERE id = ?", catalogID).Error)
	for _, scenario := range []struct {
		name       string
		connection uuid.UUID
		purpose    string
		workspace  any
		secret     string
	}{
		{"revoked catalog still coalesces", connectionID, "catalog", nil, "another-secret"},
		{"catalog forbids workspace", uuid.New(), "catalog", uuid.New(), ""},
		{"run requires workspace", uuid.New(), "run", nil, ""},
		{"login forbids workspace", uuid.New(), "login", uuid.New(), ""},
		{"unknown purpose", uuid.New(), "unknown", nil, ""},
		{"secret remains unique", uuid.New(), "catalog", nil, "run-secret"},
		{"login remains unique", connectionID, "login", nil, ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			err := tx.Transaction(func(inner *gorm.DB) error {
				return inner.Exec(`INSERT INTO factory_codex_leases
 (id, connection_id, task_id, generation, purpose, factory_id, secret_hash)
 VALUES (?, ?, ?, 10, ?, ?, ?)`, uuid.New(), scenario.connection, uuid.New(), scenario.purpose, scenario.workspace, scenario.secret).Error
			})
			require.Error(t, err)
		})
	}
}
