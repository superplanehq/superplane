package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
)

func Test__ListIntegrationResources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Run("missing integration returns not found", func(t *testing.T) {
		_, err := ListIntegrationResources(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			uuid.NewString(),
			map[string]string{"type": "repository"},
		)

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("missing integration implementation returns failed precondition", func(t *testing.T) {
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"missing-app",
			support.RandomName("integration"),
			map[string]any{},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)

		_, err = ListIntegrationResources(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			integration.ID.String(),
			map[string]string{"type": "repository"},
		)

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("pending integration returns empty resources", func(t *testing.T) {
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"missing-app",
			support.RandomName("integration"),
			map[string]any{},
		)
		require.NoError(t, err)

		resp, err := ListIntegrationResources(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			integration.ID.String(),
			map[string]string{"type": "repository"},
		)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Resources)
	})

	t.Run("error integration returns failed precondition", func(t *testing.T) {
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"missing-app",
			support.RandomName("integration"),
			map[string]any{},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateError).Error)

		_, err = ListIntegrationResources(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			integration.ID.String(),
			map[string]string{"type": "repository"},
		)

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("integration list failure returns failed precondition", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			ListResources: func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
				return nil, errors.New("boom")
			},
		})

		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"dummy",
			support.RandomName("integration"),
			map[string]any{},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)

		_, err = ListIntegrationResources(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			integration.ID.String(),
			map[string]string{"type": "repository"},
		)

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})
}

func TestIntegrationModelResourcesRefreshSelectableCatalog(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	integration := models.NewIntegration(uuid.New(), r.Organization.ID, "openai", support.RandomName("openai"), map[string]any{})
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)
	r.Registry.Integrations["openai"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{ListResources: func(string, core.ListResourcesContext) ([]core.IntegrationResource, error) {
		return []core.IntegrationResource{{ID: "gpt-5", Name: "GPT Five"}}, nil
	}})
	_, err := ListIntegrationResources(t.Context(), r.Registry, r.Organization.ID.String(), integration.ID.String(), map[string]string{"type": "repository"})
	require.NoError(t, err)
	missing, err := models.FindIntegrationModelCatalog(db, integration.ID)
	require.NoError(t, err)
	assert.Nil(t, missing, "other resource requests cannot initialize a model catalog")
	_, err = ListIntegrationResources(t.Context(), r.Registry, r.Organization.ID.String(), integration.ID.String(), map[string]string{"type": "model"})
	require.NoError(t, err)
	listed, err := models.ListSelectableLLMModels(db, r.Organization.ID, nil)
	require.NoError(t, err)
	found := false
	for _, model := range listed {
		if model.Key == "byok::openai::gpt-5" {
			found = true
			assert.Equal(t, "GPT Five", model.Model.Name)
		}
	}
	assert.True(t, found, "the integration model picker refreshes shared provider discovery")
}
