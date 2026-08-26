package organizations

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__ListBYOKLLMModels(t *testing.T) {
	r := support.Setup(t)
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := ListBYOKLLMModels(context.Background(), r.Registry, "bad", &pb.ListBYOKLLMModelsRequest{Provider: "openai"})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("unsupported provider", func(t *testing.T) {
		_, err := ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: "unknown"})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("factory not found", func(t *testing.T) {
		_, err := ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{
			Provider:  "openai",
			FactoryId: uuid.NewString(),
		})
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("no connected integration", func(t *testing.T) {
		_, err := models.UpsertOrganizationBYOKModelAllowlist(database.Conn(), r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{"gpt-4.1"})
		require.NoError(t, err)

		resp, err := ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{
			Provider: "openai",
		})
		require.NoError(t, err)
		assert.False(t, resp.Connected)
		require.Len(t, resp.Selected, 1)
		assert.Equal(t, "gpt-4.1", resp.Selected[0].Id)
		assert.Empty(t, resp.Candidates)
	})

	t.Run("lists candidates from connected integration", func(t *testing.T) {
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"openai",
			support.RandomName("openai"),
			map[string]any{},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)

		r.Registry.Integrations["openai"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			ListResources: func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
				assert.Equal(t, "model", resourceType)
				return []core.IntegrationResource{
					{ID: "gpt-4.1", Name: "GPT 4.1"},
					{Name: "gpt-4o"},
				}, nil
			},
		})

		resp, err := ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{
			Provider: "openai",
		})
		require.NoError(t, err)
		assert.True(t, resp.Connected)
		assert.Equal(t, integration.ID.String(), resp.IntegrationId)
		require.Len(t, resp.Candidates, 2)
		assert.Equal(t, "GPT 4.1", resp.Candidates[0].Name)
		require.Len(t, resp.Selected, 1)
		assert.Equal(t, "GPT 4.1", resp.Selected[0].Name)
	})

	t.Run("candidate list failure", func(t *testing.T) {
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"claude",
			support.RandomName("claude"),
			map[string]any{},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)

		r.Registry.Integrations["claude"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			ListResources: func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
				return nil, fmt.Errorf("provider unavailable")
			},
		})

		_, err = ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{
			Provider: "anthropic",
		})
		assert.Equal(t, codes.Internal, grpcerrors.Code(err))
	})
}

func Test__UpdateBYOKLLMModels(t *testing.T) {
	r := support.Setup(t)
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := UpdateBYOKLLMModels(context.Background(), "bad", &pb.UpdateBYOKLLMModelsRequest{Provider: "openai"})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("duplicate models", func(t *testing.T) {
		_, err := UpdateBYOKLLMModels(context.Background(), r.Organization.ID.String(), &pb.UpdateBYOKLLMModelsRequest{
			Provider:      "openai",
			AllowedModels: []string{"gpt-4.1", "gpt-4.1"},
		})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("saves selected models", func(t *testing.T) {
		resp, err := UpdateBYOKLLMModels(context.Background(), r.Organization.ID.String(), &pb.UpdateBYOKLLMModelsRequest{
			Provider:      "openai",
			AllowedModels: []string{"gpt-4.1", "gpt-4o"},
		})
		require.NoError(t, err)
		require.Len(t, resp.Selected, 2)
		assert.Equal(t, "gpt-4.1", resp.Selected[0].Id)
		assert.Equal(t, "gpt-4o", resp.Selected[1].Id)
	})
}
