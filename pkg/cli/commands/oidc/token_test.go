package oidc

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	spoidc "github.com/superplanehq/superplane/pkg/oidc"
)

func TestValidateTokenRoundTrip(t *testing.T) {
	t.Parallel()

	provider, err := spoidc.NewProviderFromKeyDir("http://superplane.test", filepath.Join("..", "..", "..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	publicKeys, err := publicKeysFromJWKs(provider.PublicJWKs())
	require.NoError(t, err)

	token, err := provider.Sign("execution:test", time.Hour, "superplane-ci", map[string]any{
		"org_id": uuid.NewString(),
	})
	require.NoError(t, err)

	claims, err := validateToken(token, "http://superplane.test", publicKeys)
	require.NoError(t, err)
	require.NotEmpty(t, claims["org_id"])
}
