package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestLockedPendingRunnerExpires(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)
	registration := &models.RunnerRegistration{
		JTI:       uuid.New(),
		RunnerID:  runner.ID,
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-2 * time.Minute),
	}
	require.NoError(t, db.Create(registration).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		return locked.Expire(tx, now)
	}))

	reloaded, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloaded.State)
	require.NotNil(t, reloaded.TerminationReason)
	assert.Equal(t, models.RunnerTerminationRegistrationExpired, *reloaded.TerminationReason)

	var reloadedRegistration models.RunnerRegistration
	require.NoError(t, db.Where("jti = ?", registration.JTI).First(&reloadedRegistration).Error)
	require.NotNil(t, reloadedRegistration.RevokedAt)
	assert.WithinDuration(t, now, *reloadedRegistration.RevokedAt, time.Millisecond)
}
