package secrets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	local := map[string]string{"test": "test"}
	data, _ := json.Marshal(local)

	_, err := models.CreateSecret("test", secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)

	t.Run("secret does not exist -> error", func(t *testing.T) {
		_, err := UpdateSecret(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), "test2", &protos.Secret{})
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "secret not found", msg)
	})

	t.Run("secret data is updated", func(t *testing.T) {
		secret := &protos.Secret{
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
		}

		response, err := UpdateSecret(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), "test", secret)
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

func Test__UpdateSecretMigratesLocalDataToIDAssociatedData(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := crypto.NewAESGCMEncryptor([]byte("1234567890abcdefghijklmnopqrstuv"))
	name := support.RandomName("secret")
	legacyData := map[string]string{"test": "test"}
	raw, err := json.Marshal(legacyData)
	require.NoError(t, err)

	encrypted, err := encryptor.Encrypt(context.Background(), raw, []byte(name))
	require.NoError(t, err)

	secret, err := models.CreateSecret(name, secrets.ProviderLocal, uuid.NewString(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
	require.NoError(t, err)

	updatedData := map[string]string{"test": "updated"}
	spec := &protos.Secret{
		Metadata: &protos.Secret_Metadata{Name: name},
		Spec: &protos.Secret_Spec{
			Provider: protos.Secret_PROVIDER_LOCAL,
			Local:    &protos.Secret_Local{Data: updatedData},
		},
	}

	_, err = UpdateSecret(context.Background(), encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), name, spec)
	require.NoError(t, err)

	updated, err := models.FindSecretByID(models.DomainTypeOrganization, r.Organization.ID, secret.ID.String())
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(context.Background(), updated.Data, []byte(updated.ID.String()))
	require.NoError(t, err)
	var actual map[string]string
	require.NoError(t, json.Unmarshal(decrypted, &actual))
	require.Equal(t, updatedData, actual)

	_, err = encryptor.Decrypt(context.Background(), updated.Data, []byte(updated.Name))
	require.Error(t, err)
}
