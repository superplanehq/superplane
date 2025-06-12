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

func Test__ListSecrets(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.ListSecretsRequest{
			CanvasIdOrName: uuid.NewString(),
		}

		_, err := ListSecrets(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no secrets", func(t *testing.T) {
		req := &protos.ListSecretsRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		}

		response, err := ListSecrets(context.Background(), encryptor, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.Secrets)
	})

	t.Run("secret exists", func(t *testing.T) {
		local := map[string]string{"test": "test"}
		data, _ := json.Marshal(local)

		models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), r.Canvas.ID, data)
		req := &protos.ListSecretsRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		}

		response, err := ListSecrets(context.Background(), encryptor, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Secrets, 1)

		secret := response.Secrets[0]
		assert.NotEmpty(t, secret.Metadata.Id)
		assert.NotEmpty(t, secret.Metadata.CreatedAt)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, secret.Spec.Provider)
		require.NotNil(t, secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***"}, secret.Spec.Local.Data)
	})
}
