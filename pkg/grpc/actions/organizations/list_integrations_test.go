package organizations

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__ListIntegrations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("invalid organization ID -> invalid argument", func(t *testing.T) {
		_, err := ListIntegrations(ctx, r.Registry, "not-a-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("organization with no integrations -> empty list", func(t *testing.T) {
		org, err := models.CreateOrganization("empty-org", "")
		require.NoError(t, err)

		response, err := ListIntegrations(ctx, r.Registry, org.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, response.Integrations)
	})

	t.Run("organization with integrations -> integrations returned", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		found := slices.ContainsFunc(response.Integrations, func(integration *pb.Integration) bool {
			return integration.Metadata.Id == integrationID
		})
		assert.True(t, found, "Created integration should appear in the list")
	})

	t.Run("integration whose app is no longer registered -> skipped, no failure", func(t *testing.T) {
		//
		// Register an integration so we can create an installation, then
		// remove it from the registry to simulate the production scenario
		// where an installation references an app name that the running
		// process does not know about (e.g. removed app, mismatched build).
		// ListIntegrations must not return a 500; the unknown installation
		// is skipped via the per-integration serialization error guard.
		//
		r.Registry.Integrations["temp-app"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "temp-app", name, appConfig)
		require.NoError(t, err)
		unknownIntegrationID := createResponse.Integration.Metadata.Id

		delete(r.Registry.Integrations, "temp-app")

		response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		found := slices.ContainsFunc(response.Integrations, func(integration *pb.Integration) bool {
			return integration.Metadata.Id == unknownIntegrationID
		})
		assert.False(t, found, "Integration whose app is missing from the registry should be skipped")
	})

	t.Run("integrations are scoped to the organization", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		otherOrg, err := models.CreateOrganization("other-org", "")
		require.NoError(t, err)

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		response, err := ListIntegrations(ctx, r.Registry, otherOrg.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		found := slices.ContainsFunc(response.Integrations, func(integration *pb.Integration) bool {
			return integration.Metadata.Id == integrationID
		})
		assert.False(t, found, "Integration belonging to a different organization must not be listed")
	})
}
