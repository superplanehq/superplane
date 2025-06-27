package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DescribeOrganization(context.Background(), &protos.DescribeOrganizationRequest{
			IdOrName: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("describe organization by ID", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org", "Test Organization")
		require.NoError(t, err)

		response, err := DescribeOrganization(context.Background(), &protos.DescribeOrganizationRequest{
			IdOrName: organization.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, organization.Name, response.Organization.Metadata.Name)
		assert.Equal(t, organization.DisplayName, response.Organization.Metadata.DisplayName)
		assert.Equal(t, organization.CreatedBy.String(), response.Organization.Metadata.CreatedBy)
		assert.Equal(t, *organization.CreatedAt, response.Organization.Metadata.CreatedAt.AsTime())
		assert.Equal(t, *organization.UpdatedAt, response.Organization.Metadata.UpdatedAt.AsTime())
	})

	t.Run("describe organization by name", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org-by-name", "Test Organization By Name")
		require.NoError(t, err)

		response, err := DescribeOrganization(context.Background(), &protos.DescribeOrganizationRequest{
			IdOrName: organization.Name,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, organization.Name, response.Organization.Metadata.Name)
		assert.Equal(t, organization.DisplayName, response.Organization.Metadata.DisplayName)
		assert.Equal(t, organization.CreatedBy.String(), response.Organization.Metadata.CreatedBy)
	})

	t.Run("empty id_or_name -> error", func(t *testing.T) {
		_, err := DescribeOrganization(context.Background(), &protos.DescribeOrganizationRequest{
			IdOrName: "",
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "id_or_name is required", s.Message())
	})
}
