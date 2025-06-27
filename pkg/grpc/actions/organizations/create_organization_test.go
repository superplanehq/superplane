package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	user := models.User{
		ID:   uuid.New(),
		Name: "test-user",
	}

	err := user.Create()
	require.NoError(t, err)
	authService := auth.SetupTestAuthService(t)
	ctx := context.Background()
	ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())

	t.Run("valid organization -> organization is created", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "test-org",
				DisplayName: "Test Organization",
			},
		}

		response, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, authService)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		assert.NotEmpty(t, response.Organization.Metadata.Id)
		assert.NotEmpty(t, response.Organization.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Organization.Metadata.UpdatedAt)
		assert.Equal(t, "test-org", response.Organization.Metadata.Name)
		assert.Equal(t, "Test Organization", response.Organization.Metadata.DisplayName)
		assert.Equal(t, user.ID.String(), response.Organization.Metadata.CreatedBy)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "test-org",
				DisplayName: "Another Test Organization",
			},
		}

		_, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})

	t.Run("missing name -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				DisplayName: "Test Organization",
			},
		}

		_, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "organization name is required", s.Message())
	})

	t.Run("missing display name -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name: "test-org-2",
			},
		}

		_, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, authService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "organization display name is required", s.Message())
	})
}
