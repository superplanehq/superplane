package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/protobuf/encoding/protojson"
)

type secretFieldIntegration struct {
	*impl.DummyIntegration
}

func (s *secretFieldIntegration) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:      "adminKey",
			Type:      configuration.FieldTypeString,
			Sensitive: true,
			Togglable: true,
		},
	}
}

func Test__encryptConfigurationIfNeeded(t *testing.T) {
	encryptor := crypto.NewNoOpEncryptor()
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)

	integration := &secretFieldIntegration{DummyIntegration: impl.NewDummyIntegration(impl.DummyIntegrationOptions{})}
	installationID := uuid.New()

	t.Run("empty sensitive value removes the stored secret", func(t *testing.T) {
		existing := map[string]any{"adminKey": "encrypted-old", "apiKey": "keep"}
		result, err := encryptConfigurationIfNeeded(
			context.Background(),
			reg,
			integration,
			map[string]any{"adminKey": ""},
			installationID,
			existing,
		)
		require.NoError(t, err)
		_, exists := result["adminKey"]
		assert.False(t, exists)
		_, exists = existing["adminKey"]
		assert.False(t, exists)
		assert.Equal(t, "keep", existing["apiKey"])
	})

	t.Run("null sensitive value removes the stored secret", func(t *testing.T) {
		existing := map[string]any{"adminKey": "encrypted-old"}
		result, err := encryptConfigurationIfNeeded(
			context.Background(),
			reg,
			integration,
			map[string]any{"adminKey": nil},
			installationID,
			existing,
		)
		require.NoError(t, err)
		_, exists := result["adminKey"]
		assert.False(t, exists)
		_, exists = existing["adminKey"]
		assert.False(t, exists)
	})

	t.Run("new sensitive value is encrypted", func(t *testing.T) {
		existing := map[string]any{"adminKey": "encrypted-old"}
		result, err := encryptConfigurationIfNeeded(
			context.Background(),
			reg,
			integration,
			map[string]any{"adminKey": "new-secret"},
			installationID,
			existing,
		)
		require.NoError(t, err)
		assert.NotEqual(t, "new-secret", result["adminKey"])
		assert.NotEmpty(t, result["adminKey"])
		assert.Equal(t, "encrypted-old", existing["adminKey"])
	})
}

func Test__updateIntegrationRequestKeepsClearedSensitiveValues(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		req := &pb.UpdateIntegrationRequest{}
		require.NoError(t, protojson.Unmarshal([]byte(`{"configuration":{"adminKey":""}}`), req))

		value, exists := req.Configuration.AsMap()["adminKey"]
		require.True(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("null", func(t *testing.T) {
		req := &pb.UpdateIntegrationRequest{}
		require.NoError(t, protojson.Unmarshal([]byte(`{"configuration":{"adminKey":null}}`), req))

		value, exists := req.Configuration.AsMap()["adminKey"]
		require.True(t, exists)
		assert.Nil(t, value)
	})
}
