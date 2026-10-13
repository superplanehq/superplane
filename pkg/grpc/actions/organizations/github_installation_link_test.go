package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListIntegrations__GitHubAppInstallationLink(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()

	t.Run("hosted installation includes the GitHub settings URL", func(t *testing.T) {
		require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
			Provider:       models.ProviderGitHub,
			InstallationID: 101,
			AccountLogin:   "acme",
			AccountType:    "Organization",
			HTMLURL:        "https://github.com/organizations/acme/settings/installations/101",
		}))
		integration, err := models.FindOrCreateVCSProviderBinding(db, r.Organization.ID, models.ProviderGitHub, 101, "acme")
		require.NoError(t, err)

		listed := findListedIntegration(ctx, t, r, integration.ID.String())
		property := installationURLProperty(listed)
		require.NotNil(t, property)
		assert.Equal(t, "https://github.com/organizations/acme/settings/installations/101", property.Value)
		assert.False(t, property.Editable)
	})

	t.Run("keeps a stored installation URL", func(t *testing.T) {
		require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
			Provider:       models.ProviderGitHub,
			InstallationID: 202,
			AccountLogin:   "other",
			AccountType:    "Organization",
			HTMLURL:        "https://github.com/organizations/other/settings/installations/202",
		}))
		integration, err := models.FindOrCreateVCSProviderBinding(db, r.Organization.ID, models.ProviderGitHub, 202, "other")
		require.NoError(t, err)
		stored := "https://github.com/settings/installations/202"
		integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{{
			Name:     common.PropertyAppInstallationURL,
			Label:    "GitHub App Installation URL",
			Type:     core.IntegrationPropertyTypeString,
			Value:    stored,
			Editable: false,
		}})
		require.NoError(t, db.Save(integration).Error)

		listed := findListedIntegration(ctx, t, r, integration.ID.String())
		matches := installationURLProperties(listed)
		require.Len(t, matches, 1)
		assert.Equal(t, stored, matches[0].Value)
	})

	t.Run("omits the link when the GitHub integration has no installation", func(t *testing.T) {
		now := time.Now()
		integration := &models.Integration{
			ID:               uuid.New(),
			OrganizationID:   r.Organization.ID,
			AppName:          models.ProviderGitHub,
			InstallationName: "github-pat-" + uuid.NewString(),
			State:            models.IntegrationStateReady,
			Configuration:    datatypes.NewJSONType(map[string]any{}),
			Metadata:         datatypes.NewJSONType(map[string]any{}),
			CreatedAt:        &now,
			UpdatedAt:        &now,
		}
		require.NoError(t, db.Create(integration).Error)

		listed := findListedIntegration(ctx, t, r, integration.ID.String())
		assert.Nil(t, installationURLProperty(listed))
	})
}

func findListedIntegration(ctx context.Context, t *testing.T, r *support.ResourceRegistry, integrationID string) *pb.Integration {
	t.Helper()
	response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	for _, integration := range response.Integrations {
		if integration.Metadata.Id == integrationID {
			return integration
		}
	}
	t.Fatalf("integration %s not listed", integrationID)
	return nil
}

func installationURLProperty(integration *pb.Integration) *pb.Integration_Property {
	matches := installationURLProperties(integration)
	if len(matches) == 0 {
		return nil
	}
	return matches[0]
}

func installationURLProperties(integration *pb.Integration) []*pb.Integration_Property {
	matches := []*pb.Integration_Property{}
	for _, property := range integration.Status.Properties {
		if property.Name == common.PropertyAppInstallationURL {
			matches = append(matches, property)
		}
	}
	return matches
}
