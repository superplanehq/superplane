package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

// contextCapturingEncryptor records the context passed to Decrypt so tests
// can assert that LocalProvider forwards the caller's context instead of
// substituting context.TODO() or context.Background().
type contextCapturingEncryptor struct {
	plaintext   []byte
	decryptErr  error
	receivedCtx context.Context
}

var _ crypto.Encryptor = (*contextCapturingEncryptor)(nil)

func (e *contextCapturingEncryptor) Encrypt(_ context.Context, data []byte, _ []byte) ([]byte, error) {
	return data, nil
}

func (e *contextCapturingEncryptor) Decrypt(ctx context.Context, _ []byte, _ []byte) ([]byte, error) {
	e.receivedCtx = ctx
	if e.decryptErr != nil {
		return nil, e.decryptErr
	}
	return e.plaintext, nil
}

func Test__LocalProvider(t *testing.T) {
	type ctxKey string

	t.Run("forwards caller context value to encryptor", func(t *testing.T) {
		payload, err := json.Marshal(map[string]string{"key": "value"})
		require.NoError(t, err)

		encryptor := &contextCapturingEncryptor{plaintext: payload}
		provider := NewLocalProvider(nil, encryptor, &models.Secret{
			Name: "test-secret",
			Data: payload,
		})

		ctx := context.WithValue(context.Background(), ctxKey("trace-id"), "abc-123")

		_, err = provider.Load(ctx)
		require.NoError(t, err)
		require.NotNil(t, encryptor.receivedCtx)
		require.Equal(t, "abc-123", encryptor.receivedCtx.Value(ctxKey("trace-id")),
			"expected encryptor to receive caller's context, but the value was not propagated")
	})

	t.Run("forwards caller context cancellation to encryptor", func(t *testing.T) {
		payload, err := json.Marshal(map[string]string{"key": "value"})
		require.NoError(t, err)

		encryptor := &contextCapturingEncryptor{plaintext: payload}
		provider := NewLocalProvider(nil, encryptor, &models.Secret{
			Name: "test-secret",
			Data: payload,
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = provider.Load(ctx)
		require.NoError(t, err)
		require.NotNil(t, encryptor.receivedCtx)
		require.ErrorIs(t, encryptor.receivedCtx.Err(), context.Canceled,
			"expected propagated context to carry caller's cancellation")
	})

	t.Run("returns parsed values on successful decryption", func(t *testing.T) {
		expected := map[string]string{"username": "admin", "password": "s3cret"}
		serialized, err := json.Marshal(expected)
		require.NoError(t, err)

		encryptor := &contextCapturingEncryptor{plaintext: serialized}
		provider := NewLocalProvider(nil, encryptor, &models.Secret{
			Name: "creds",
			Data: serialized,
		})

		got, err := provider.Load(context.Background())
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})

	t.Run("wraps decryption errors with the secret name", func(t *testing.T) {
		encryptor := &contextCapturingEncryptor{decryptErr: errors.New("boom")}
		provider := NewLocalProvider(nil, encryptor, &models.Secret{
			Name: "broken",
			Data: []byte("garbage"),
		})

		_, err := provider.Load(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error decrypting secret broken")
	})

	t.Run("returns error when decrypted payload is not valid JSON", func(t *testing.T) {
		encryptor := &contextCapturingEncryptor{plaintext: []byte("not-json")}
		provider := NewLocalProvider(nil, encryptor, &models.Secret{
			Name: "malformed",
			Data: []byte("garbage"),
		})

		_, err := provider.Load(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "error unmarshaling secret malformed")
	})
}
