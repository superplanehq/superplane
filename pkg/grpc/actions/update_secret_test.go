package actions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	local := map[string]string{"test": "test"}
	data, _ := json.Marshal(local)

	_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), r.Canvas.ID, data)
	require.NoError(t, err)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.UpdateSecretRequest{
			CanvasIdOrName: uuid.New().String(),
		}

		_, err := UpdateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing requester ID", func(t *testing.T) {
		req := &protos.UpdateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       "test",
		}

		_, err := UpdateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid requester ID", s.Message())
	})

	t.Run("secret does not exist -> error", func(t *testing.T) {
		req := &protos.UpdateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    uuid.NewString(),
			IdOrName:       "test2",
		}

		_, err := UpdateSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("secret data is updated", func(t *testing.T) {
		req := &protos.UpdateSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       "test",
			RequesterId:    uuid.NewString(),
			Secret: &protos.Secret{
				Metadata: &protos.Secret_Metadata{
					Name: "test",
				},
				Spec: &protos.Secret_Spec{
					Provider: protos.Secret_PROVIDER_LOCAL,
					Local: &protos.Secret_Local{
						Data: map[string]string{
							"test":  "test",
							"test2": "test2",
						},
					},
				},
			},
		}

		response, err := UpdateSecret(context.Background(), encryptor, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.NotEmpty(t, response.Secret.Metadata.Id)
		assert.NotEmpty(t, response.Secret.Metadata.CreatedAt)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, response.Secret.Spec.Provider)
		require.NotNil(t, response.Secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***", "test2": "***"}, response.Secret.Spec.Local.Data)
	})
}
