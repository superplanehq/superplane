package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__UpdateIntegration(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("update configuration with successful sync -> integration updated", func(t *testing.T) {
		//
		// Register a test integration that succeeds on Sync
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		integrationName := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		//
		// Create integration
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", integrationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Update the integration configuration
		//
		updateResponse, err := UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), integrationID, map[string]any{"key": "value2", "new_key": "new_value"}, "")
		require.NoError(t, err)
		require.NotNil(t, updateResponse)
		require.NotNil(t, updateResponse.Integration)

		//
		// Verify integration was updated
		//
		integration, err := models.FindIntegrationByName(r.Organization.ID, integrationName)
		require.NoError(t, err)
		assert.Equal(t, models.IntegrationStateReady, integration.State)
		assert.Empty(t, integration.StateDescription)

		//
		// Verify configuration was updated (merged with existing config)
		//
		config := integration.Configuration.Data()
		assert.Equal(t, "value2", config["key"])
		assert.Equal(t, "new_value", config["new_key"])

		//
		// Verify the response reflects updated integration
		//
		assert.Equal(t, models.IntegrationStateReady, updateResponse.Integration.Status.State)
		assert.Empty(t, updateResponse.Integration.Status.StateDescription)
	})

	t.Run("update configuration with sync failure -> integration in error state", func(t *testing.T) {
		//
		// Register a test integration that succeeds initially but fails on update
		//
		syncCount := 0
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				syncCount++
				if syncCount == 1 {
					ctx.Integration.Ready()
					return nil
				}
				return errors.New("sync failed on update")
			},
		})

		integrationName := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		//
		// Create integration
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", integrationName, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Update the integration configuration (this should fail)
		//
		updateResponse, err := UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), integrationID, map[string]any{"key": "value2"}, "")
		require.NoError(t, err)
		require.NotNil(t, updateResponse)

		//
		// Verify integration is in error state
		//
		integration, err := models.FindIntegrationByName(r.Organization.ID, integrationName)
		require.NoError(t, err)
		assert.Equal(t, models.IntegrationStateError, integration.State)
		assert.Contains(t, integration.StateDescription, "sync failed on update")

		//
		// Verify the response reflects error state
		//
		assert.Equal(t, models.IntegrationStateError, updateResponse.Integration.Status.State)
		assert.Contains(t, updateResponse.Integration.Status.StateDescription, "sync failed on update")
	})

	t.Run("invalid integration ID -> error", func(t *testing.T) {
		//
		// Try to update with an invalid integration ID
		//
		_, err := UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "invalid-uuid", map[string]any{"key": "value"}, "")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid integration ID")
	})

	t.Run("non-existent integration -> error", func(t *testing.T) {
		//
		// Try to update a non-existent integration
		//
		_, err := UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), uuid.NewString(), map[string]any{"key": "value"}, "")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "integration not found")
	})

	t.Run("update preserves existing configuration keys -> success", func(t *testing.T) {
		//
		// Register a test integration that succeeds on Sync
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		integrationName := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"})
		require.NoError(t, err)

		//
		// Create integration with multiple config keys
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", integrationName, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Update only one key
		//
		_, err = UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), integrationID, map[string]any{"key2": "updated_value2"}, "")
		require.NoError(t, err)

		//
		// Verify all keys are preserved, and only the updated key changed
		//
		integration, err := models.FindIntegrationByName(r.Organization.ID, integrationName)
		require.NoError(t, err)
		config := integration.Configuration.Data()
		assert.Equal(t, "value1", config["key1"], "key1 should be preserved")
		assert.Equal(t, "updated_value2", config["key2"], "key2 should be updated")
		assert.Equal(t, "value3", config["key3"], "key3 should be preserved")
	})

	t.Run("update integration name -> integration updated", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		integrationName := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", integrationName, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		updatedName := support.RandomName("integration")
		updateResponse, err := UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), integrationID, nil, updatedName)
		require.NoError(t, err)
		require.NotNil(t, updateResponse)
		require.NotNil(t, updateResponse.Integration)

		_, err = models.FindIntegrationByName(r.Organization.ID, integrationName)
		require.Error(t, err)

		integration, err := models.FindIntegrationByName(r.Organization.ID, updatedName)
		require.NoError(t, err)
		assert.Equal(t, updatedName, integration.InstallationName)
		assert.Equal(t, updatedName, updateResponse.Integration.Metadata.Name)
	})

	t.Run("update integration name to existing name -> already exists", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		firstName := support.RandomName("integration")
		secondName := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		_, err = CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", firstName, appConfig)
		require.NoError(t, err)

		secondIntegration, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", secondName, appConfig)
		require.NoError(t, err)

		_, err = UpdateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), secondIntegration.Integration.Metadata.Id, nil, firstName)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
		assert.Contains(t, s.Message(), "already exists")
	})
}
