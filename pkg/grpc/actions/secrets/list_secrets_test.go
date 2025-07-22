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
	protos "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListSecrets(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	t.Run("no secrets", func(t *testing.T) {
		response, err := ListSecrets(context.Background(), encryptor, models.DomainTypeCanvas, r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.Secrets)
	})

	t.Run("secret exists", func(t *testing.T) {
		local := map[string]string{"test": "test"}
		data, _ := json.Marshal(local)

		_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), models.DomainTypeCanvas, r.Canvas.ID, data)
		require.NoError(t, err)

		response, err := ListSecrets(context.Background(), encryptor, models.DomainTypeCanvas, r.Canvas.ID.String())
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
