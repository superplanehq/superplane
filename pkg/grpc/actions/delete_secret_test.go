package actions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test__DeleteSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	local := map[string]string{"test": "test"}
	data, _ := json.Marshal(local)

	_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), r.Canvas.ID, data)
	require.NoError(t, err)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.DeleteSecretRequest{
			CanvasIdOrName: uuid.New().String(),
		}

		_, err := DeleteSecret(context.Background(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing requester ID", func(t *testing.T) {
		req := &protos.DeleteSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       "test",
		}

		_, err := DeleteSecret(context.Background(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid requester ID", s.Message())
	})

	t.Run("secret does not exist -> error", func(t *testing.T) {
		req := &protos.DeleteSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			RequesterId:    uuid.NewString(),
			IdOrName:       "test2",
		}

		_, err := DeleteSecret(context.Background(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "secret not found", s.Message())
	})

	t.Run("secret is deleted", func(t *testing.T) {
		req := &protos.DeleteSecretRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			IdOrName:       "test",
			RequesterId:    uuid.NewString(),
		}

		_, err := DeleteSecret(context.Background(), req)
		require.NoError(t, err)

		_, err = models.FindSecretByName(r.Canvas.ID.String(), "test")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
