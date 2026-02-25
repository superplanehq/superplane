package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_FindSecretValue(t *testing.T) {
	secrets := []core.IntegrationSecret{
		{Name: "other", Value: []byte("x")},
		{Name: SecretNameAccessToken, Value: []byte("my-token")},
		{Name: SecretNameServiceAccountKey, Value: []byte("key-json")},
	}
	assert.Nil(t, FindSecretValue(secrets, "missing"))
	assert.Equal(t, []byte("my-token"), FindSecretValue(secrets, SecretNameAccessToken))
	assert.Equal(t, []byte("key-json"), FindSecretValue(secrets, SecretNameServiceAccountKey))
	assert.Nil(t, FindSecretValue(nil, SecretNameAccessToken))
}

func Test_AuthMethodFromMetadata(t *testing.T) {
	assert.Equal(t, AuthMethodServiceAccountKey, AuthMethodFromMetadata(nil))
	assert.Equal(t, AuthMethodServiceAccountKey, AuthMethodFromMetadata(map[string]any{}))
	assert.Equal(t, AuthMethodServiceAccountKey, AuthMethodFromMetadata(map[string]any{"authMethod": "other"}))
	assert.Equal(t, AuthMethodWIF, AuthMethodFromMetadata(map[string]any{"authMethod": AuthMethodWIF}))
}

func Test_TokenSourceFromIntegration(t *testing.T) {
	t.Run("no credentials returns error", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}}
		_, err := TokenSourceFromIntegration(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no GCP credentials found")
	})

	t.Run("WIF with access token returns token source", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				SecretNameAccessToken: {Name: SecretNameAccessToken, Value: []byte("wif-access-token")},
			},
			Metadata: map[string]any{"authMethod": AuthMethodWIF},
		}
		ts, err := TokenSourceFromIntegration(ctx)
		require.NoError(t, err)
		require.NotNil(t, ts)
		tok, err := ts.Token()
		require.NoError(t, err)
		assert.Equal(t, "wif-access-token", tok.AccessToken)
		assert.Equal(t, "Bearer", tok.TokenType)
	})

	t.Run("WIF with expired token in metadata returns error", func(t *testing.T) {
		expired := time.Now().Add(-time.Hour).Format(time.RFC3339)
		ctx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				SecretNameAccessToken: {Name: SecretNameAccessToken, Value: []byte("tok")},
			},
			Metadata: map[string]any{
				"authMethod":           AuthMethodWIF,
				"accessTokenExpiresAt": expired,
			},
		}
		_, err := TokenSourceFromIntegration(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access token expired")
	})

	t.Run("service account key with invalid JSON returns error", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				SecretNameServiceAccountKey: {Name: SecretNameServiceAccountKey, Value: []byte(`{invalid`)},
			},
			Metadata: map[string]any{"authMethod": AuthMethodServiceAccountKey},
		}
		_, err := TokenSourceFromIntegration(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create credentials from service account key")
	})
}

func Test_CredentialsFromIntegration(t *testing.T) {
	t.Run("delegates to TokenSource and returns credentials", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				SecretNameAccessToken: {Name: SecretNameAccessToken, Value: []byte("tok")},
			},
			Metadata: map[string]any{"authMethod": AuthMethodWIF},
		}
		creds, err := CredentialsFromIntegration(ctx)
		require.NoError(t, err)
		require.NotNil(t, creds)
		assert.NotNil(t, creds.TokenSource)
	})

	t.Run("error from TokenSource is propagated", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}}
		_, err := CredentialsFromIntegration(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no GCP credentials found")
	})
}
