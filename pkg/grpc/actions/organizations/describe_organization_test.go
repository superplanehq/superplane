package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeOrganization(t *testing.T) {
	r := support.Setup(t)

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DescribeOrganization(context.Background(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("describe organization by ID", func(t *testing.T) {
		response, err := DescribeOrganization(context.Background(), r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, r.Organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, r.Organization.Name, response.Organization.Metadata.Name)
		assert.Equal(t, r.Organization.Description, response.Organization.Metadata.Description)
		assert.Equal(t, *r.Organization.CreatedAt, response.Organization.Metadata.CreatedAt.AsTime())
		assert.Equal(t, *r.Organization.UpdatedAt, response.Organization.Metadata.UpdatedAt.AsTime())
		require.NotNil(t, response.Organization.Spec)
	})

	t.Run("invalid organization id -> internal error, not panic", func(t *testing.T) {
		// PostgreSQL will reject a non-UUID literal with a non-NotFound error.
		// The handler must surface that as Internal (not raw error / 500 with no payload).
		_, err := DescribeOrganization(context.Background(), "not-a-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, s.Code())
		assert.Equal(t, "failed to describe organization", s.Message())
	})

	t.Run("organization with nil timestamps -> no panic", func(t *testing.T) {
		// Historical/partially-migrated rows may have NULL created_at/updated_at.
		// Dereferencing those pointers used to panic the gRPC handler, surfacing
		// as an info-level HTTP 500 in Sentry with no underlying context.
		err := database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			UpdateColumns(map[string]any{
				"created_at": nil,
				"updated_at": nil,
			}).Error
		require.NoError(t, err)

		response, err := DescribeOrganization(context.Background(), r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Nil(t, response.Organization.Metadata.CreatedAt)
		assert.Nil(t, response.Organization.Metadata.UpdatedAt)
	})
}
