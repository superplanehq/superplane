package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestGetInstallationMetadata(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	metadata, err := GetInstallationMetadata(database.Conn())
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, installationMetadataID, metadata.ID)
	assert.NotEmpty(t, metadata.InstallationID)
	assert.False(t, metadata.AllowPrivateNetworkAccess)
	assert.True(t, metadata.SignupsEnabled)
}

func TestUpdateInstallationMetadata(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	metadata, err := GetInstallationMetadata(database.Conn())
	require.NoError(t, err)

	metadata.AllowPrivateNetworkAccess = true
	metadata.SignupsEnabled = false
	metadata.UpdatedAt = time.Now()

	require.NoError(t, UpdateInstallationMetadata(database.Conn(), metadata))

	updated, err := GetInstallationMetadata(database.Conn())
	require.NoError(t, err)
	assert.True(t, updated.AllowPrivateNetworkAccess)
	assert.False(t, updated.SignupsEnabled)
	assert.Equal(t, metadata.InstallationID, updated.InstallationID)
}

func TestSetOwnerSetupStepIfOpenRejectsClosedWizard(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	require.NoError(t, SetOwnerSetupStep(database.Conn(), OwnerSetupStepLicense))

	updated, err := SetOwnerSetupStepIfOpen(database.Conn(), "")
	require.NoError(t, err)
	assert.True(t, updated)

	updated, err = SetOwnerSetupStepIfOpen(database.Conn(), OwnerSetupStepSMTP)
	require.NoError(t, err)
	assert.False(t, updated)

	step, err := GetOwnerSetupStep(database.Conn())
	require.NoError(t, err)
	assert.Empty(t, step)
}

func TestSetOwnerSetupStepIfOpenDoesNotReopenAfterOverlappingClose(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	require.NoError(t, SetOwnerSetupStep(database.Conn(), OwnerSetupStepLicense))

	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	var metadata InstallationMetadata
	require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", installationMetadataID).
		First(&metadata).Error)

	type outcome struct {
		updated bool
		err     error
	}
	done := make(chan outcome, 1)
	go func() {
		updated, err := SetOwnerSetupStepIfOpen(database.Conn(), OwnerSetupStepSMTP)
		done <- outcome{updated: updated, err: err}
	}()

	time.Sleep(200 * time.Millisecond)
	require.NoError(t, tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Update("owner_setup_step", gorm.Expr("NULL")).Error)
	require.NoError(t, tx.Commit().Error)

	select {
	case result := <-done:
		require.NoError(t, result.err)
		assert.False(t, result.updated)
	case <-time.After(5 * time.Second):
		t.Fatal("step update did not finish after the close committed")
	}

	step, err := GetOwnerSetupStep(database.Conn())
	require.NoError(t, err)
	assert.Empty(t, step)
}
