package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListIntegrations(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Integration: true})
	defer r.Close()

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListIntegrations(context.Background(), &protos.ListIntegrationsRequest{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("returns list of integrations", func(t *testing.T) {
		res, err := ListIntegrations(context.Background(), &protos.ListIntegrationsRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Integrations, 1)
		assert.Equal(t, r.Integration.ID.String(), res.Integrations[0].Metadata.Id)
		assert.Equal(t, r.Integration.Name, res.Integrations[0].Metadata.Name)
		assert.Equal(t, r.Integration.DomainID.String(), res.Integrations[0].Metadata.DomainId)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_CANVAS, res.Integrations[0].Metadata.DomainType)
		assert.NotEmpty(t, res.Integrations[0].Metadata.CreatedAt)
		assert.Equal(t, r.Integration.CreatedBy.String(), res.Integrations[0].Metadata.CreatedBy)
		assert.Equal(t, protos.Integration_TYPE_SEMAPHORE, res.Integrations[0].Spec.Type)
		assert.Equal(t, r.Integration.URL, res.Integrations[0].Spec.Url)
		assert.Equal(t, protos.Integration_AUTH_TYPE_TOKEN, res.Integrations[0].Spec.Auth.Use)
		assert.NotNil(t, res.Integrations[0].Spec.Auth.Token)
	})
}
