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

func Test__DescribeIntegration(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("describe existing integration -> success", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create an integration
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Describe the integration
		//
		describeResponse, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)
		require.NotNil(t, describeResponse)
		require.NotNil(t, describeResponse.Integration)

		//
		// Verify the response contains the expected integration details
		//
		assert.Equal(t, integrationID, describeResponse.Integration.Metadata.Id)
		assert.Equal(t, name, describeResponse.Integration.Metadata.Name)
		assert.Equal(t, "dummy", describeResponse.Integration.Spec.IntegrationName)
		assert.Equal(t, models.IntegrationStateReady, describeResponse.Integration.Status.State)
	})

	t.Run("invalid organization ID -> error", func(t *testing.T) {
		//
		// Try to describe with an invalid organization ID
		//
		_, err := DescribeIntegration(ctx, r.Registry, "invalid-uuid", uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("invalid integration ID -> error", func(t *testing.T) {
		//
		// Try to describe with an invalid integration ID
		//
		_, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), "invalid-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid integration ID")
	})

	t.Run("non-existent integration -> error", func(t *testing.T) {
		//
		// Try to describe a non-existent integration
		//
		fakeIntegrationID := uuid.NewString()
		_, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), fakeIntegrationID)
		require.Error(t, err)
	})

	t.Run("integration from different organization -> error", func(t *testing.T) {
		//
		// Register a test integration
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		//
		// Create a second organization
		//
		org2, err := models.CreateOrganization("org-2", "")
		require.NoError(t, err)

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{"key": "value"})
		require.NoError(t, err)

		//
		// Create integration in first organization
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Try to describe the integration using the second organization's ID
		//
		_, err = DescribeIntegration(ctx, r.Registry, org2.ID.String(), integrationID)
		require.Error(t, err)
	})

	t.Run("describe integration with configuration -> configuration returned", func(t *testing.T) {
		//
		// Register a test application
		//
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			return nil
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": 123,
		})
		require.NoError(t, err)

		//
		// Create integration with configuration
		//
		createResponse, err := CreateIntegration(ctx, r.Registry, nil, baseURL, baseURL, r.Organization.ID.String(), "dummy", name, appConfig)
		require.NoError(t, err)
		integrationID := createResponse.Integration.Metadata.Id

		//
		// Describe the integration
		//
		describeResponse, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integrationID)
		require.NoError(t, err)
		require.NotNil(t, describeResponse)
		require.NotNil(t, describeResponse.Integration)

		//
		// Verify configuration is returned
		//
		assert.NotNil(t, describeResponse.Integration.Spec.Configuration)
	})
}
