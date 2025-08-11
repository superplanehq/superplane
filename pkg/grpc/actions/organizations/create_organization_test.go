package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("valid organization -> organization is created", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "test-org",
				DisplayName: "Test Organization",
				Description: "This is a test organization",
			},
		}

		response, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, r.AuthService)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		assert.NotEmpty(t, response.Organization.Metadata.Id)
		assert.NotEmpty(t, response.Organization.Metadata.CreatedAt)
		assert.NotEmpty(t, response.Organization.Metadata.UpdatedAt)
		assert.Equal(t, "test-org", response.Organization.Metadata.Name)
		assert.Equal(t, "Test Organization", response.Organization.Metadata.DisplayName)
		assert.Equal(t, "This is a test organization", response.Organization.Metadata.Description)
		assert.Equal(t, r.User.String(), response.Organization.Metadata.CreatedBy)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        r.Organization.Name,
				DisplayName: r.Organization.DisplayName,
			},
		}

		_, err := CreateOrganization(ctx, &protos.CreateOrganizationRequest{
			Organization: organization,
		}, r.AuthService)

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
		}, r.AuthService)

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
		}, r.AuthService)

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "organization display name is required", s.Message())
	})
}
