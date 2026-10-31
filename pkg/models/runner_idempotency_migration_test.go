package models_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunnerIdempotencyMigrationPreservesInstallationGrants(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	other := newTestRunnerFleet()
	other.Slug = "second-installation-fleet"
	require.NoError(t, other.Create(db))
	migration, err := os.ReadFile("../../db/migrations/20261031204546_scope-runner-idempotency-to-fleet.up.sql")
	require.NoError(t, err)
	// Restore the pre-migration index within one rollback-only transaction.
	// This executes the real migration against historical installation rows.
	tx := db.Begin()
	defer tx.Rollback()
	require.NoError(t, tx.Exec("DROP INDEX runners_fleet_creation_idempotency_key").Error)
	require.NoError(t, tx.Exec("CREATE UNIQUE INDEX runners_creation_idempotency_key ON runners (creation_idempotency_key) WHERE creation_idempotency_key IS NOT NULL").Error)
	var runners []*models.Runner
	var grants []*models.RunnerRegistration
	for _, tagged := range []bool{false, true} {
		key, hash := uuid.NewString(), "historical-request-hash"
		tags := map[string]string{}
		if tagged {
			tags = map[string]string{models.RunnerTagFleetManagerID: "historical-manager", models.RunnerTagProvider: "aws"}
		}
		runner := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, RunnerVersion: "dev", State: models.RunnerStatePending,
			Ephemeral: true, CreationIdempotencyKey: &key, CreationRequestHash: &hash, Tags: datatypes.NewJSONType(tags)}
		require.NoError(t, tx.Create(runner).Error)
		grant := &models.RunnerRegistration{JTI: uuid.New(), RunnerID: runner.ID, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
		require.NoError(t, tx.Create(grant).Error)
		runners, grants = append(runners, runner), append(grants, grant)
	}
	require.NoError(t, tx.Exec(string(migration)).Error)
	for i, runner := range runners {
		stored, err := models.FindRunnerByCreationIdempotencyKey(tx, fleet.ID, *runner.CreationIdempotencyKey)
		require.NoError(t, err)
		assert.Equal(t, runner.ID, stored.ID)
		assert.Equal(t, "historical-request-hash", *stored.CreationRequestHash)
		assert.Equal(t, runner.Tags.Data(), stored.Tags.Data())
		grant, err := models.FindRunnerRegistration(tx, stored.ID)
		require.NoError(t, err)
		assert.Equal(t, grants[i].JTI, grant.JTI)
		assert.WithinDuration(t, grants[i].ExpiresAt, grant.ExpiresAt, time.Microsecond)
		assert.Nil(t, grant.ConsumedAt)
		assert.Nil(t, grant.RevokedAt)
		// The migration also permits arbitrary same-key creation in a new fleet.
		copy := *runner
		copy.ID, copy.FleetID = uuid.New(), other.ID
		require.NoError(t, tx.Create(&copy).Error)
	}
}
