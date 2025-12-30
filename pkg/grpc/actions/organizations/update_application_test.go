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

func Test__UpdateApplication(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("update configuration with successful sync -> installation updated", func(t *testing.T) {
		//
		// Register a test application that succeeds on Sync
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, installResponse)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Update the installation configuration
		//
		updateResponse, err := UpdateApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), installationID, map[string]any{"key": "value2", "new_key": "new_value"})
		require.NoError(t, err)
		require.NotNil(t, updateResponse)
		require.NotNil(t, updateResponse.Installation)

		//
		// Verify installation was updated
		//
		installation, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, models.AppInstallationStateReady, installation.State)
		assert.Empty(t, installation.StateDescription)

		//
		// Verify configuration was updated (merged with existing config)
		//
		config := installation.Configuration.Data()
		assert.Equal(t, "value2", config["key"])
		assert.Equal(t, "new_value", config["new_key"])

		//
		// Verify the response reflects updated installation
		//
		assert.Equal(t, models.AppInstallationStateReady, updateResponse.Installation.Status.State)
		assert.Empty(t, updateResponse.Installation.Status.StateDescription)
	})

	t.Run("update configuration with sync failure -> installation in error state", func(t *testing.T) {
		//
		// Register a test application that succeeds initially but fails on update
		//
		syncCount := 0
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			syncCount++
			if syncCount == 1 {
				ctx.AppInstallation.SetState("ready", "")
				return nil
			}
			return errors.New("sync failed on update")
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value1"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Update the installation configuration (this should fail)
		//
		updateResponse, err := UpdateApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), installationID, map[string]any{"key": "value2"})
		require.NoError(t, err)
		require.NotNil(t, updateResponse)

		//
		// Verify installation is in error state
		//
		installation, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, models.AppInstallationStateError, installation.State)
		assert.Contains(t, installation.StateDescription, "sync failed on update")

		//
		// Verify the response reflects error state
		//
		assert.Equal(t, models.AppInstallationStateError, updateResponse.Installation.Status.State)
		assert.Contains(t, updateResponse.Installation.Status.StateDescription, "sync failed on update")
	})

	t.Run("invalid installation ID -> error", func(t *testing.T) {
		//
		// Try to update with an invalid installation ID
		//
		_, err := UpdateApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), "invalid-uuid", map[string]any{"key": "value"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid installation ID")
	})

	t.Run("non-existent installation -> error", func(t *testing.T) {
		//
		// Try to update a non-existent installation
		//
		_, err := UpdateApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), uuid.NewString(), map[string]any{"key": "value"})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "application installation not found")
	})

	t.Run("update preserves existing configuration keys -> success", func(t *testing.T) {
		//
		// Register a test application that succeeds on Sync
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"})
		require.NoError(t, err)

		//
		// Create installation with multiple config keys
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Update only one key
		//
		_, err = UpdateApplication(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), installationID, map[string]any{"key2": "updated_value2"})
		require.NoError(t, err)

		//
		// Verify all keys are preserved, and only the updated key changed
		//
		installation, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		config := installation.Configuration.Data()
		assert.Equal(t, "value1", config["key1"], "key1 should be preserved")
		assert.Equal(t, "updated_value2", config["key2"], "key2 should be updated")
		assert.Equal(t, "value3", config["key3"], "key3 should be preserved")
	})
}
