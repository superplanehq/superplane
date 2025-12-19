package organizations

import (
	"context"
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

func Test__DescribeApplication(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("describe existing installation -> success", func(t *testing.T) {
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
		// Describe the installation
		//
		describeResponse, err := DescribeApplication(ctx, r.Registry, r.Organization.ID.String(), installationID)
		require.NoError(t, err)
		require.NotNil(t, describeResponse)
		require.NotNil(t, describeResponse.Installation)

		//
		// Verify the response contains the expected installation details
		//
		assert.Equal(t, installationID, describeResponse.Installation.Metadata.Id)
		assert.Equal(t, installationName, describeResponse.Installation.Metadata.Name)
		assert.Equal(t, "dummy", describeResponse.Installation.Spec.AppName)
		assert.Equal(t, models.AppInstallationStateReady, describeResponse.Installation.Status.State)
	})

	t.Run("invalid organization ID -> error", func(t *testing.T) {
		//
		// Try to describe with an invalid organization ID
		//
		_, err := DescribeApplication(ctx, r.Registry, "invalid-uuid", uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("invalid installation ID -> error", func(t *testing.T) {
		//
		// Try to describe with an invalid installation ID
		//
		_, err := DescribeApplication(ctx, r.Registry, r.Organization.ID.String(), "invalid-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid installation ID")
	})

	t.Run("non-existent installation -> error", func(t *testing.T) {
		//
		// Try to describe a non-existent installation
		//
		fakeInstallationID := uuid.NewString()
		_, err := DescribeApplication(ctx, r.Registry, r.Organization.ID.String(), fakeInstallationID)
		require.Error(t, err)
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
		// Try to describe the installation using the second organization's ID
		//
		_, err = DescribeApplication(ctx, r.Registry, org2.ID.String(), installationID)
		require.Error(t, err)
	})

	t.Run("describe installation with configuration -> configuration returned", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
			ctx.AppInstallation.SetState("ready", "")
			return nil
		})

		installationName := support.RandomName("installation")
		appConfig, err := structpb.NewStruct(map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": 123,
		})
		require.NoError(t, err)

		//
		// Create installation with configuration
		//
		installResponse, err := InstallApplication(ctx, r.Registry, baseURL, r.Organization.ID.String(), "dummy", installationName, appConfig)
		require.NoError(t, err)
		installationID := installResponse.Installation.Metadata.Id

		//
		// Describe the installation
		//
		describeResponse, err := DescribeApplication(ctx, r.Registry, r.Organization.ID.String(), installationID)
		require.NoError(t, err)
		require.NotNil(t, describeResponse)
		require.NotNil(t, describeResponse.Installation)

		//
		// Verify configuration is returned (note: configuration might be filtered/processed)
		//
		assert.NotNil(t, describeResponse.Installation.Spec.Configuration)
	})
}
