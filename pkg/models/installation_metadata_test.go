package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestGetInstallationMetadata(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	metadata, err := GetInstallationMetadata()
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, installationMetadataID, metadata.ID)
	assert.NotEmpty(t, metadata.InstallationID)
	assert.False(t, metadata.AllowPrivateNetworkAccess)
}

func TestUpdateInstallationMetadata(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	metadata, err := GetInstallationMetadata()
	require.NoError(t, err)

	metadata.AllowPrivateNetworkAccess = true
	metadata.UpdatedAt = time.Now()

	require.NoError(t, UpdateInstallationMetadata(metadata))

	updated, err := GetInstallationMetadata()
	require.NoError(t, err)
	assert.True(t, updated.AllowPrivateNetworkAccess)
	assert.Equal(t, metadata.InstallationID, updated.InstallationID)
}
