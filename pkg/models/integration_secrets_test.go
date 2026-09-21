package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestListIntegrationSecretsForInstallations(t *testing.T) {
	r := support.Setup(t)
	now := time.Now()
	firstID := uuid.New()
	secondID := uuid.New()

	require.NoError(t, database.Conn().Create(&models.Integration{
		ID:               firstID,
		OrganizationID:   r.Organization.ID,
		AppName:          "dummy",
		InstallationName: support.RandomName("integration"),
		State:            models.IntegrationStateReady,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}).Error)
	require.NoError(t, database.Conn().Create(&models.Integration{
		ID:               secondID,
		OrganizationID:   r.Organization.ID,
		AppName:          "dummy",
		InstallationName: support.RandomName("integration"),
		State:            models.IntegrationStateReady,
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}).Error)

	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: r.Organization.ID,
		InstallationID: firstID,
		Name:           "token-a",
		Value:          []byte("a"),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)
	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: r.Organization.ID,
		InstallationID: secondID,
		Name:           "token-b",
		Value:          []byte("b"),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)

	found, err := models.ListIntegrationSecretsForInstallations(database.Conn(), []uuid.UUID{firstID, secondID})
	require.NoError(t, err)
	require.Len(t, found[firstID], 1)
	require.Equal(t, "token-a", found[firstID][0].Name)
	require.Len(t, found[secondID], 1)
	require.Equal(t, "token-b", found[secondID][0].Name)

	empty, err := models.ListIntegrationSecretsForInstallations(database.Conn(), nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}
