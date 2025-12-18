package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func Test__InstallApplication(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("duplicate installation name -> error", func(t *testing.T) {
		installationName := "my-github-app"
		appConfig, err := structpb.NewStruct(map[string]interface{}{
			"organization": "test-org",
		})
		require.NoError(t, err)

		//
		// Create first installation
		//
		response, err := InstallApplication(ctx, r.Registry, "http://localhost", r.Organization.ID.String(), "github", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Installation)
		assert.Equal(t, installationName, response.Installation.Metadata.Name)

		//
		// Try to create second installation with the same name
		//
		_, err = InstallApplication(ctx, r.Registry, "http://localhost", r.Organization.ID.String(), "github", installationName, appConfig)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
		assert.Contains(t, s.Message(), "an installation with the name my-github-app already exists")
	})

	t.Run("reuse installation name after deletion -> success", func(t *testing.T) {
		installationName := "reusable-github-app"
		appConfig, err := structpb.NewStruct(map[string]interface{}{
			"organization": "test-org",
		})
		require.NoError(t, err)

		//
		// Create first installation
		//
		response, err := InstallApplication(ctx, r.Registry, "http://localhost", r.Organization.ID.String(), "github", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response)
		installationID := response.Installation.Metadata.Id

		//
		// Verify installation exists with the correct name
		//
		installation, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, installationID, installation.ID.String())
		assert.Equal(t, installationName, installation.InstallationName)

		//
		// Delete the installation
		//
		_, err = UninstallApplication(ctx, r.Organization.ID.String(), installationID)
		require.NoError(t, err)

		//
		// Verify the installation is soft-deleted and the name has been modified
		//
		deletedInstallation, err := models.FindMaybeDeletedInstallationInTransaction(database.Conn(), installation.ID)
		require.NoError(t, err)
		assert.True(t, deletedInstallation.DeletedAt.Valid)
		assert.NotEqual(t, installationName, deletedInstallation.InstallationName)
		assert.Contains(t, deletedInstallation.InstallationName, "deleted-")

		//
		// Verify we cannot find it by the original name anymore
		//
		_, err = models.FindAppInstallationByName(r.Organization.ID, installationName)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		//
		// Create a new installation with the same name
		//
		response2, err := InstallApplication(ctx, r.Registry, "http://localhost", r.Organization.ID.String(), "github", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, installationName, response2.Installation.Metadata.Name)
		assert.NotEqual(t, installationID, response2.Installation.Metadata.Id, "New installation should have different ID")

		//
		// Verify we can find the new installation by name
		//
		newInstallation, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, installationName, newInstallation.InstallationName)
		assert.Equal(t, response2.Installation.Metadata.Id, newInstallation.ID.String())
	})

	t.Run("different organizations can have installations with the same name -> success", func(t *testing.T) {
		//
		// Create a second organization
		//
		org2, err := models.CreateOrganization("org-2", "")
		require.NoError(t, err)

		installationName := "shared-name"
		appConfig, err := structpb.NewStruct(map[string]interface{}{
			"organization": "test-org",
		})
		require.NoError(t, err)

		//
		// Create installation in first organization
		//
		response1, err := InstallApplication(ctx, r.Registry, "http://localhost", r.Organization.ID.String(), "github", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response1)
		assert.Equal(t, installationName, response1.Installation.Metadata.Name)

		//
		// Create installation with same name in second organization
		//
		response2, err := InstallApplication(ctx, r.Registry, "http://localhost", org2.ID.String(), "github", installationName, appConfig)
		require.NoError(t, err)
		require.NotNil(t, response2)
		assert.Equal(t, installationName, response2.Installation.Metadata.Name)
		assert.NotEqual(t, response1.Installation.Metadata.Id, response2.Installation.Metadata.Id)

		//
		// Verify both installations exist and can be found by name in their respective organizations
		//
		installation1, err := models.FindAppInstallationByName(r.Organization.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, response1.Installation.Metadata.Id, installation1.ID.String())

		installation2, err := models.FindAppInstallationByName(org2.ID, installationName)
		require.NoError(t, err)
		assert.Equal(t, response2.Installation.Metadata.Id, installation2.ID.String())
	})
}
