package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateOrganization(t *testing.T) {
	r := support.Setup(t)

	t.Run("organization does not exist -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name: "updated-name",
			},
		}

		_, err := UpdateOrganization(context.Background(), uuid.New().String(), organization)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "organization not found", msg)
	})

	t.Run("update organization by ID -> success", func(t *testing.T) {
		updatedOrg := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "updated-org",
				Description: "Updated description",
			},
		}

		response, err := UpdateOrganization(context.Background(), r.Organization.ID.String(), updatedOrg)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, r.Organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "updated-org", response.Organization.Metadata.Name)
		assert.Equal(t, "Updated description", response.Organization.Metadata.Description)
		assert.Equal(t, *r.Organization.CreatedAt, response.Organization.Metadata.CreatedAt.AsTime())
		assert.True(t, response.Organization.Metadata.UpdatedAt.AsTime().After(*r.Organization.UpdatedAt))
		require.NotNil(t, response.Organization.Spec)
	})

	t.Run("nil organization -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), uuid.New().String(), nil)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "organization is required", msg)
	})

	t.Run("nil organization metadata -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), uuid.New().String(), &protos.Organization{})
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "organization metadata is required", msg)
	})

	t.Run("update organization by slug -> success", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.Slug, &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Description: "updated via slug",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Organization)
		assert.Equal(t, org.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "updated via slug", response.Organization.Metadata.Description)
	})

	t.Run("update slug -> success", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "my-new-slug",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Organization)
		assert.Equal(t, "my-new-slug", response.Organization.Metadata.Slug)

		reloaded, err := models.FindOrganizationBySlug(database.Conn(), "my-new-slug")
		require.NoError(t, err)
		assert.Equal(t, org.ID, reloaded.ID)
	})

	t.Run("update slug to one already in use -> invalid argument", func(t *testing.T) {
		taken, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: taken.Slug,
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug to a reserved word -> invalid argument", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "admin",
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug with invalid characters -> invalid argument", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "Not A Slug!",
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug to empty -> keeps existing slug", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Description: "no slug change",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, org.Slug, response.Organization.Metadata.Slug)
	})
}
