package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryWorkspaceRepositories(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	newWorkspace := func(t *testing.T, provider string) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		integrationID := uuid.NewString()
		repository := "acme/app"
		patch := models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &repository,
		}
		if provider != "" {
			patch.VCSProvider = &provider
		}
		require.NoError(t, factory.UpdateOnboarding(db, patch))
		return factory
	}

	addContextRepo := func(t *testing.T, factory *models.Factory, repository string, enabled bool) {
		t.Helper()
		_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", enabled, models.FactoryAgentResourceConfig{
			Repository: repository,
		})
		require.NoError(t, err)
	}

	t.Run("lists the primary repository first and then enabled context repositories", func(t *testing.T) {
		factory := newWorkspace(t, "")
		addContextRepo(t, factory, "acme/web", true)
		addContextRepo(t, factory, "acme/lib", true)
		addContextRepo(t, factory, "acme/old", false)

		repositories, err := factory.WorkspaceRepositories(db)
		require.NoError(t, err)
		assert.Equal(t, []string{"acme/app", "acme/lib", "acme/web"}, repositories)
	})

	t.Run("matches workspace repositories without case", func(t *testing.T) {
		factory := newWorkspace(t, "")
		addContextRepo(t, factory, "acme/web", true)

		for _, repository := range []string{"acme/app", "ACME/Web"} {
			ok, err := factory.IsWorkspaceRepository(db, repository)
			require.NoError(t, err)
			assert.True(t, ok, repository)
		}

		ok, err := factory.IsWorkspaceRepository(db, "acme/other")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("keeps only the primary repository for Bitbucket", func(t *testing.T) {
		factory := newWorkspace(t, models.ProviderBitbucket)
		addContextRepo(t, factory, "acme/web", true)

		repositories, err := factory.WorkspaceRepositories(db)
		require.NoError(t, err)
		assert.Equal(t, []string{"acme/app"}, repositories)
	})
}
