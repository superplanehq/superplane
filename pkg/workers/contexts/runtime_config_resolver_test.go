package contexts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ResolveRuntimeConfig_ResolvesSecretsExpression(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	encryptor := &crypto.NoOpEncryptor{}
	secretName := "api-keys"
	secretData := map[string]string{"token": "sk-test-123"}
	raw, err := json.Marshal(secretData)
	require.NoError(t, err)
	encrypted, err := encryptor.Encrypt(context.Background(), raw, []byte(secretName))
	require.NoError(t, err)

	_, err = models.CreateSecret(secretName, secrets.ProviderLocal, r.User.String(), models.DomainTypeOrganization, r.Organization.ID, encrypted)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent}},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{"node-1": map[string]any{"x": "y"}})

	config := map[string]any{
		"api_key": `{{ secrets("api-keys").token }}`,
	}

	resolved, err := ResolveRuntimeConfig(config, builder, database.Conn(), encryptor, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, "sk-test-123", resolved["api_key"])
}

func Test_ResolveRuntimeConfig_NoExpressions_ReturnsCopy(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent}},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).WithInput(map[string]any{})

	config := map[string]any{
		"plain": "value",
		"nested": map[string]any{"a": "b"},
	}

	resolved, err := ResolveRuntimeConfig(config, builder, database.Conn(), &crypto.NoOpEncryptor{}, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, "value", resolved["plain"])
	assert.Equal(t, map[string]any{"a": "b"}, resolved["nested"])
}

func Test_ResolveRuntimeConfig_SecretNotFound_ReturnsError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent}},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).WithInput(map[string]any{})

	config := map[string]any{
		"api_key": `{{ secrets("nonexistent").key }}`,
	}

	_, err := ResolveRuntimeConfig(config, builder, database.Conn(), &crypto.NoOpEncryptor{}, r.Organization.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func Test_expressionContainsSecrets(t *testing.T) {
	assert.True(t, expressionContainsSecrets(`secrets("name").key`))
	assert.True(t, expressionContainsSecrets(`"Bearer " + secrets("api").token`))
	assert.False(t, expressionContainsSecrets(`$.trigger.body`))
	assert.False(t, expressionContainsSecrets(`root().x`))
	assert.False(t, expressionContainsSecrets(`previous().value`))
}
