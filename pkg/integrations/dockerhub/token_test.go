package dockerhub

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test__ParseJWTExpiry(t *testing.T) {
	t.Run("parses exp claim", func(t *testing.T) {
		exp := time.Now().Add(10 * time.Minute).Unix()
		token := buildJWT(t, map[string]any{"exp": exp})

		parsed, err := parseJWTExpiry(token)
		require.NoError(t, err)
		require.Equal(t, time.Unix(exp, 0), *parsed)
	})

	t.Run("missing exp -> error", func(t *testing.T) {
		token := buildJWT(t, map[string]any{"sub": "user"})
		_, err := parseJWTExpiry(token)
		require.Error(t, err)
	})
}

func buildJWT(t *testing.T, payload map[string]any) string {
	t.Helper()

	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	require.NoError(t, err)

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	headerSegment := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadSegment := base64.RawURLEncoding.EncodeToString(payloadBytes)

	return headerSegment + "." + payloadSegment + ".signature"
}
