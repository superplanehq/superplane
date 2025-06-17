package secrets

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

func Test__DescribeSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.DescribeSecretRequest{
			CanvasIdOrName: uuid.NewString(),
			IdOrName:       "test",
		}

		_, err := DescribeSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("secret does not exist -> error", func(t *testing.T) {
		req := &protos.DescribeSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       uuid.NewString(),
		}

		_, err := DescribeSecret(context.Background(), encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("secret exists", func(t *testing.T) {
		local := map[string]string{"test": "test"}
		data, _ := json.Marshal(local)

		_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), r.Canvas.ID, data)
		require.NoError(t, err)

		req := &protos.DescribeSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       "test",
		}

		response, err := DescribeSecret(context.Background(), encryptor, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		require.NotNil(t, response.Secret.Metadata)
		require.NotNil(t, response.Secret.Spec)
		assert.NotEmpty(t, response.Secret.Metadata.Id)
		assert.NotEmpty(t, response.Secret.Metadata.CreatedAt)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, response.Secret.Spec.Provider)
		require.NotNil(t, response.Secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***"}, response.Secret.Spec.Local.Data)
	})
}
