package contexts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func createContextSecret(t *testing.T, encryptor crypto.Encryptor, domainType string, domainID uuid.UUID, name string, data map[string]string) {
	t.Helper()

	plain, err := json.Marshal(data)
	require.NoError(t, err)

	encoded, err := encryptor.Encrypt(context.Background(), plain, []byte(name))
	require.NoError(t, err)

	_, err = models.CreateSecret(name, secrets.ProviderLocal, uuid.New().String(), domainType, domainID, encoded)
	require.NoError(t, err)
}

func Test__SecretsContext_GetKey(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	encryptor := crypto.NewNoOpEncryptor()

	t.Run("resolves an org-only secret (regression guard)", func(t *testing.T) {
		name := support.RandomName("secret")
		createContextSecret(t, encryptor, models.DomainTypeOrganization, r.Organization.ID, name, map[string]string{"key": "org-value"})

		ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
		value, err := ctx.GetKey(name, "key")
		require.NoError(t, err)
		assert.Equal(t, "org-value", string(value))
	})

	t.Run("resolves a canvas-only secret", func(t *testing.T) {
		name := support.RandomName("secret")
		createContextSecret(t, encryptor, models.DomainTypeCanvas, canvas.ID, name, map[string]string{"key": "canvas-value"})

		ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
		value, err := ctx.GetKey(name, "key")
		require.NoError(t, err)
		assert.Equal(t, "canvas-value", string(value))
	})

	t.Run("canvas secret shadows an org secret with the same name", func(t *testing.T) {
		name := support.RandomName("secret")
		createContextSecret(t, encryptor, models.DomainTypeOrganization, r.Organization.ID, name, map[string]string{"key": "org-value"})
		createContextSecret(t, encryptor, models.DomainTypeCanvas, canvas.ID, name, map[string]string{"key": "canvas-value"})

		ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
		value, err := ctx.GetKey(name, "key")
		require.NoError(t, err)
		assert.Equal(t, "canvas-value", string(value))
	})

	t.Run("secret not found in either domain", func(t *testing.T) {
		ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
		_, err := ctx.GetKey(support.RandomName("missing"), "key")
		require.Error(t, err)
	})

	t.Run("a canvas secret from a different canvas isn't visible", func(t *testing.T) {
		otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		name := support.RandomName("secret")
		createContextSecret(t, encryptor, models.DomainTypeCanvas, otherCanvas.ID, name, map[string]string{"key": "other-canvas-value"})

		ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
		_, err := ctx.GetKey(name, "key")
		require.Error(t, err)
	})
}

func Test__SecretsContext_GetSecretKeys(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	encryptor := crypto.NewNoOpEncryptor()

	name := support.RandomName("secret")
	createContextSecret(t, encryptor, models.DomainTypeOrganization, r.Organization.ID, name, map[string]string{"key": "org-value"})
	createContextSecret(t, encryptor, models.DomainTypeCanvas, canvas.ID, name, map[string]string{"key": "canvas-value", "other": "value"})

	ctx := NewSecretsContext(database.Conn(), r.Registry, r.Organization.ID, canvas.ID, encryptor)
	keys, err := ctx.GetSecretKeys(name)
	require.NoError(t, err)
	assert.Equal(t, "canvas-value", string(keys["key"]))
	assert.Equal(t, "value", string(keys["other"]))
}
