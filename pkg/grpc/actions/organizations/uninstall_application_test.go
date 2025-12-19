package organizations

import (
	"context"
	"slices"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__UninstallApplication(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("uninstall existing installation -> success", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Verify installation is visible in the list
		//
		listResponse, err := ListApplications(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		found := slices.ContainsFunc(listResponse.Applications, func(app *pb.AppInstallation) bool {
			return app.Metadata.Id == installationID
		})
		assert.True(t, found, "Installation should be visible before uninstallation")

		//
		// Uninstall the application
		//
		uninstallResponse, err := UninstallApplication(ctx, r.Organization.ID.String(), installationID)
		require.NoError(t, err)
		require.NotNil(t, uninstallResponse)

		//
		// Verify installation is no longer visible in the list
		//
		listResponse, err = ListApplications(ctx, r.Registry, r.Organization.ID.String())
		require.NoError(t, err)
		found = slices.ContainsFunc(listResponse.Applications, func(app *pb.AppInstallation) bool {
			return app.Metadata.Id == installationID
		})
		assert.False(t, found, "Uninstalled installation should not be visible in list")

		//
		// Verify installation is soft-deleted in the database
		//
		installation, err := models.FindMaybeDeletedInstallationInTransaction(database.Conn(), uuid.MustParse(installationID))
		require.NoError(t, err)
		assert.True(t, installation.DeletedAt.Valid)
	})

	t.Run("invalid organization ID -> error", func(t *testing.T) {
		//
		// Try to uninstall with an invalid organization ID
		//
		_, err := UninstallApplication(ctx, "invalid-uuid", uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("invalid installation ID -> error", func(t *testing.T) {
		//
		// Try to uninstall with an invalid installation ID
		//
		_, err := UninstallApplication(ctx, r.Organization.ID.String(), "invalid-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid installation ID")
	})

	t.Run("non-existent installation -> error", func(t *testing.T) {
		//
		// Try to uninstall a non-existent installation
		//
		_, err := UninstallApplication(ctx, r.Organization.ID.String(), uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "application installation not found")
	})

	t.Run("installation from different organization -> error", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		//
		// Create a second organization
		//
		org2, err := models.CreateOrganization("org-2", "")
		require.NoError(t, err)

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create installation in first organization
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Try to uninstall using the second organization's ID
		//
		_, err = UninstallApplication(ctx, org2.ID.String(), installationID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("uninstall twice -> error on second attempt", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Uninstall the application (first time)
		//
		_, err = UninstallApplication(ctx, r.Organization.ID.String(), installationID)
		require.NoError(t, err)

		//
		// Try to uninstall again (should fail)
		//
		_, err = UninstallApplication(ctx, r.Organization.ID.String(), installationID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "application installation not found")
	})

	t.Run("uninstall modifies installation name to prevent name conflicts -> success", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create installation
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Uninstall the application
		//
		_, err = UninstallApplication(ctx, r.Organization.ID.String(), installationID)
		require.NoError(t, err)

		//
		// Verify the installation name has been modified
		//
		installation, err := models.FindMaybeDeletedInstallationInTransaction(database.Conn(), uuid.MustParse(installationID))
		require.NoError(t, err)
		assert.True(t, installation.DeletedAt.Valid)
		assert.NotEqual(t, installationName, installation.InstallationName)
		assert.Contains(t, installation.InstallationName, "deleted-", "Installation name should be prefixed with 'deleted-'")
	})
}
