package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IntegrationRef_IsSet(t *testing.T) {
	t.Parallel()

	assert.False(t, IntegrationRef{}.IsSet())
	assert.True(t, IntegrationRef{Name: "my-github-integration"}.IsSet())
}

func Test__DecodeIntegrationRef(t *testing.T) {
	t.Parallel()

	ref, err := DecodeIntegrationRef(map[string]any{
		"name": "my-github-integration",
	})
	require.NoError(t, err)
	assert.Equal(t, IntegrationRef{Name: "my-github-integration"}, ref)

	ref, err = DecodeIntegrationRef(map[string]any{
		"id": "b38bbb64-d1b1-47a1-a8c3-9bc65ad4cff0",
	})
	require.NoError(t, err)
	assert.Equal(t, IntegrationRef{}, ref)

	ref, err = DecodeIntegrationRef(map[string]any{
		"id":   "6922e066-0f14-4c60-b240-79f0a9f4318f",
		"name": "GitHub Production",
	})
	require.NoError(t, err)
	assert.Equal(t, IntegrationRef{Name: "GitHub Production"}, ref)

	_, err = DecodeIntegrationRef("my-github-integration")
	require.ErrorContains(t, err, "must be an object")
}

func Test__DecodeSecretRef(t *testing.T) {
	t.Parallel()

	ref, err := DecodeSecretRef(map[string]any{"secret": "some-other-secret"})
	require.NoError(t, err)
	assert.Equal(t, SecretRef{Secret: "some-other-secret"}, ref)

	_, err = DecodeSecretRef("some-other-secret")
	require.ErrorContains(t, err, "must be an object")
}

func Test__ValidateIntegrationField(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateIntegration(map[string]any{"name": "my-github-integration"}))
	require.ErrorContains(t, validateIntegration("my-github-integration"), "must be an object")
	require.ErrorContains(t, validateIntegration(map[string]any{}), "integration is required")
}

func Test__ValidateSecretField(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateSecret(map[string]any{"secret": "some-other-secret"}))
	require.ErrorContains(t, validateSecret("some-other-secret"), "must be an object")
	require.ErrorContains(t, validateSecret(map[string]any{}), "secret is required")
}
