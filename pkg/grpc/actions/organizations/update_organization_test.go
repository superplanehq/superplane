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

func Test__UpdateOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()

	t.Run("organization does not exist -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "updated-name",
				DisplayName: "Updated Display Name",
			},
		}

		_, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName:     uuid.New().String(),
			Organization: organization,
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("update organization by ID -> success", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org", "Test Organization")
		require.NoError(t, err)

		updatedOrg := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "updated-org",
				DisplayName: "Updated Organization",
			},
		}

		response, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName:     organization.ID.String(),
			Organization: updatedOrg,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "updated-org", response.Organization.Metadata.Name)
		assert.Equal(t, "Updated Organization", response.Organization.Metadata.DisplayName)
		assert.Equal(t, organization.CreatedBy.String(), response.Organization.Metadata.CreatedBy)
		assert.Equal(t, *organization.CreatedAt, response.Organization.Metadata.CreatedAt.AsTime())
		assert.True(t, response.Organization.Metadata.UpdatedAt.AsTime().After(*organization.UpdatedAt))
	})

	t.Run("update organization by name -> success", func(t *testing.T) {
		organization, err := models.CreateOrganization(userID, "test-org-2", "Test Organization 2")
		require.NoError(t, err)

		updatedOrg := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				DisplayName: "Updated Organization 2",
			},
		}

		response, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName:     organization.Name,
			Organization: updatedOrg,
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "test-org-2", response.Organization.Metadata.Name) // Name should remain unchanged
		assert.Equal(t, "Updated Organization 2", response.Organization.Metadata.DisplayName)
	})

	t.Run("empty id_or_name -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "updated-name",
				DisplayName: "Updated Display Name",
			},
		}

		_, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName:     "",
			Organization: organization,
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "id_or_name is required", s.Message())
	})

	t.Run("nil organization -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "organization is required", s.Message())
	})

	t.Run("nil organization metadata -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), &protos.UpdateOrganizationRequest{
			IdOrName:     uuid.New().String(),
			Organization: &protos.Organization{}, // Organization exists but Metadata is nil
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "organization metadata is required", s.Message())
	})
}
