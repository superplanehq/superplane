package organizations

import (
	"context"
	"fmt"
	"github.com/superplanehq/superplane/pkg/features"
	"testing"
	"time"

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
)

func Test__ListBYOKLLMModels(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	t.Cleanup(func() {
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
		resp, err := ListBYOKLLMModels(t.Context(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: "openai"})
		require.NoError(t, err)
		assert.False(t, resp.Connected)
		assert.Empty(t, resp.Selected)
		assert.Empty(t, resp.Candidates)
	})
	t.Run("refreshes provider catalog and preserves successful observations on failure", func(t *testing.T) {
		db := database.DB(t.Context())
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "openai", support.RandomName("openai"), map[string]any{})
		require.NoError(t, err)
		require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
		snapshot, err := models.FindIntegrationModelCatalog(db, integration.ID)
		require.NoError(t, err)
		assert.Nil(t, snapshot, "missing catalog differs from a successful empty result")
		resources := []core.IntegrationResource{{ID: "gpt-4.1", Name: "GPT 4.1"}, {Name: "gpt-4o"}}
		var providerError error
		r.Registry.Integrations["openai"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			ListResources: func(string, core.ListResourcesContext) ([]core.IntegrationResource, error) {
				return resources, providerError
			},
		})
		list := func() *pb.ListBYOKLLMModelsResponse {
			response, err := ListBYOKLLMModels(t.Context(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: "openai"})
			require.NoError(t, err)
			return response
		}
		response := list()
		assert.True(t, response.Connected)
		assert.Equal(t, integration.ID.String(), response.IntegrationId)
		require.Len(t, response.Selected, 2)
		assert.Equal(t, "GPT 4.1", response.Selected[0].Name)
		snapshot, err = models.FindIntegrationModelCatalog(db, integration.ID)
		require.NoError(t, err)
		require.NotNil(t, snapshot)
		assert.WithinDuration(t, time.Now(), snapshot.ObservedAt, time.Second)
		assert.Equal(t, "GPT 4.1", snapshot.Models[0].Name)
		providerError = core.NewProviderTransportError("timeout", fmt.Errorf("timeout"))
		_, err = ListBYOKLLMModels(t.Context(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: "openai"})
		require.Error(t, err)
		retained, err := models.FindIntegrationModelCatalog(db, integration.ID)
		require.NoError(t, err)
		assert.Equal(t, snapshot, retained)
		providerError = nil
		resources = nil
		response = list()
		assert.Empty(t, response.Candidates)
		assert.Empty(t, response.Selected)
		empty, err := models.FindIntegrationModelCatalog(db, integration.ID)
		require.NoError(t, err)
		require.NotNil(t, empty)
		assert.Empty(t, empty.Models)
		replacement := models.NewIntegration(uuid.New(), r.Organization.ID, "openai", support.RandomName("replacement"), map[string]any{})
		require.NoError(t, db.Create(replacement).Error)
		missing, err := models.FindIntegrationModelCatalog(db, replacement.ID)
		require.NoError(t, err)
		assert.Nil(t, missing, "a different key must not reuse the previous integration catalog")
	})

	t.Run("candidate list failure", func(t *testing.T) {
		cases := []struct {
			name         string
			err          error
			expectedCode codes.Code
			expectedMsg  string
		}{
			{
				name:         "auth error",
				err:          core.NewProviderAPIError(401, "Claude credentials are invalid or expired", fmt.Errorf("unauthorized")),
				expectedCode: codes.FailedPrecondition,
				expectedMsg:  "Could not reach the model provider to list models. Check the connected API key and try again.",
			},
			{
				name:         "transport/timeout error",
				err:          core.NewProviderTransportError("request failed: context deadline exceeded", fmt.Errorf("context deadline exceeded")),
				expectedCode: codes.FailedPrecondition,
				expectedMsg:  "Could not reach the model provider to list models. Check the connected API key and try again.",
			},
			{
				name:         "rate limited",
				err:          core.NewProviderAPIError(429, "request got 429 code: too many requests", fmt.Errorf("too many requests")),
				expectedCode: codes.ResourceExhausted,
				expectedMsg:  "The model provider rate limited the request. Try again shortly.",
			},
			{
				name:         "unknown/untyped error",
				err:          fmt.Errorf("provider unavailable"),
				expectedCode: codes.Internal,
				expectedMsg:  "failed to list byok models",
			},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
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
						return nil, tt.err
					},
				})

				_, err = ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{
					Provider: "anthropic",
				})
				assert.Equal(t, tt.expectedCode, grpcerrors.Code(err))
				message, ok := grpcerrors.HandlerMessage(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedMsg, message)
			})
		}
	})
}
