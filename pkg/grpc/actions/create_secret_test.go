package actions

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.CreateSecretRequest{
			CanvasIdOrName: uuid.New().String(),
			Secret: &protos.Secret{
				Name:     "test",
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		_, err := CreateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing requester ID", func(t *testing.T) {
		req := &protos.CreateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Secret: &protos.Secret{
				Name:     "test",
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		_, err := CreateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid requester ID", s.Message())
	})

	t.Run("name still not used -> secret is created", func(t *testing.T) {
		req := &protos.CreateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    uuid.NewString(),
			Secret: &protos.Secret{
				Name:     "test",
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		response, err := CreateSecret(context.Background(), encryptor, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.NotEmpty(t, response.Secret.Id)
		assert.NotEmpty(t, response.Secret.CreatedAt)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, response.Secret.Provider)
		require.NotNil(t, response.Secret.Local)
		require.Equal(t, map[string]string{"test": "***"}, response.Secret.Local.Data)
	})

	t.Run("name already used", func(t *testing.T) {
		req := &protos.CreateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    uuid.NewString(),
			Secret: &protos.Secret{
				Name:     "test",
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		_, err := CreateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
