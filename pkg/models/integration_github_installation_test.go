package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListGitHubIntegrationsUsingInstallation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())

	bound, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", "github-bound", nil)
	require.NoError(t, err)
	bound.Metadata = datatypes.NewJSONType(map[string]any{
		"installationId": "11",
		"hostedApp":      true,
	})
	require.NoError(t, db.Save(bound).Error)

	pending, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", "github-pending", nil)
	require.NoError(t, err)
	pending.Metadata = datatypes.NewJSONType(map[string]any{
		"hostedApp": true,
		"pendingInstallations": []map[string]any{
			{"id": "22", "accountLogin": "octo"},
		},
	})
	require.NoError(t, db.Save(pending).Error)

	other, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", "github-other", nil)
	require.NoError(t, err)
	other.Metadata = datatypes.NewJSONType(map[string]any{
		"installationId": "99",
		"hostedApp":      true,
	})
	require.NoError(t, db.Save(other).Error)

	t.Run("finds a bound installation", func(t *testing.T) {
		found, err := models.ListGitHubIntegrationsUsingInstallation(db, "11")
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Equal(t, bound.ID, found[0].ID)
	})

	t.Run("finds a pending picker entry", func(t *testing.T) {
		found, err := models.ListGitHubIntegrationsUsingInstallation(db, "22")
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Equal(t, pending.ID, found[0].ID)
	})

	t.Run("returns empty when no connection uses the installation", func(t *testing.T) {
		found, err := models.ListGitHubIntegrationsUsingInstallation(db, "404")
		require.NoError(t, err)
		assert.Empty(t, found)
	})
}
