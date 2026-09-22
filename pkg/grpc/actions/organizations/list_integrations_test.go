package organizations

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func Test__ListIntegrations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("no integrations -> returns empty list", func(t *testing.T) {
		response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, response.Integrations)
	})

	t.Run("one integration -> returns it", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create an integration
		//
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL,
			r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)

		//
		// List integrations
		//
		response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Integrations, 1)

		//
		// Verify the returned integration matches what was created
		//
		assert.Equal(t, name, response.Integrations[0].Metadata.Name)
		assert.Equal(t, "dummy", response.Integrations[0].Metadata.IntegrationName)
		assert.Equal(t, models.IntegrationStateReady, response.Integrations[0].Status.State)
	})

	t.Run("multiple integrations -> returns all", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create a second integration
		//
		name2 := support.RandomName("integration")
		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL,
			r.Organization.ID.String(), "dummy", name2, appConfig)
		require.NoError(t, err)

		//
		// List integrations - should include the one we just created
		//
		response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		var found bool
		for _, integration := range response.Integrations {
			if integration.Metadata.Name == name2 {
				found = true
				break
			}
		}
		assert.True(t, found, "expected newly created integration not found")
		assert.GreaterOrEqual(t, len(response.Integrations), 2)
	})

	t.Run("integrations from different organization -> not visible", func(t *testing.T) {
		//
		// Create a second organization
		//
		org2 := support.CreateOrganization(t, r, r.User)

		//
		// List integrations for the new organization
		//
		response, err := ListIntegrations(ctx, r.Registry, org2.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, response.Integrations)
	})
}

func Test__ListIntegrations_LoadsSecretsInOneQuery(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		},
	})
	appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
	require.NoError(t, err)

	created := make([]*pb.CreateIntegrationResponse, 0, 2)
	for range 2 {
		name := support.RandomName("integration")
		resp, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL,
			r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		created = append(created, resp)
	}

	for _, item := range created {
		id, err := uuid.Parse(item.Integration.Metadata.Id)
		require.NoError(t, err)
		seedIntegrationSecret(t, r, id, "token", "secret-value")
	}

	secretQueries := &atomic.Int64{}
	callbackName := "count-integration-secrets:" + t.Name()
	db := database.DB(ctx)
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && strings.Contains(tx.Statement.Table, "app_installation_secrets") {
			secretQueries.Add(1)
		}
	}))
	t.Cleanup(func() {
		db.Callback().Query().Remove(callbackName)
	})

	response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(response.Integrations), 2)
	assert.Equal(t, int64(1), secretQueries.Load())
}
