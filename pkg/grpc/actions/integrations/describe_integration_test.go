package integrations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeIntegration(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})
	defer r.Close()

	t.Run("integration that does not exist -> error", func(t *testing.T) {
		_, err := DescribeIntegration(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "integration not found", s.Message())
	})

	t.Run("using id", func(t *testing.T) {
		response, err := DescribeIntegration(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), r.Integration.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, r.Integration.ID.String(), response.Integration.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.Integration.Metadata.DomainId)
		assert.Equal(t, *r.Integration.CreatedAt, response.Integration.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Integration.Name, response.Integration.Metadata.Name)
	})

	t.Run("using name", func(t *testing.T) {
		response, err := DescribeIntegration(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), r.Integration.Name)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, r.Integration.ID.String(), response.Integration.Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), response.Integration.Metadata.DomainId)
		assert.Equal(t, *r.Integration.CreatedAt, response.Integration.Metadata.CreatedAt.AsTime())
		assert.Equal(t, r.Integration.Name, response.Integration.Metadata.Name)
	})
}
