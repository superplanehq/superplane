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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		assert.Equal(t, codes.NotFound, status.Code(err))
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
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("non-ready integration returns empty resources", func(t *testing.T) {
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

	t.Run("integration list failure returns unavailable", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
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
		assert.Equal(t, codes.Unavailable, status.Code(err))
	})
}
