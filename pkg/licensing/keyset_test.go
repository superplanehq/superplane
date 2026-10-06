package licensing_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

func TestEmbeddedTrustAnchorsAreValid(t *testing.T) {
	_, err := licensing.ProductionRootKeySet()
	require.NoError(t, err)

	_, err = licensing.TrustedKeyStore(nil)
	require.NoError(t, err, "the bootstrap key list must verify with the embedded root keys")
}

func TestParseKeySetRejectsUnsafeKeys(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")

	validKey := func(t *testing.T) map[string]any {
		var set struct {
			Keys []map[string]any `json:"keys"`
		}
		require.NoError(t, json.Unmarshal(licensingtest.JWKS(issuer), &set))
		return set.Keys[0]
	}

	cases := map[string]func(map[string]any){
		"RSA key type":      func(k map[string]any) { k["kty"] = "RSA" },
		"P-384 curve":       func(k map[string]any) { k["crv"] = "P-384" },
		"encryption use":    func(k map[string]any) { k["use"] = "enc" },
		"ES384 algorithm":   func(k map[string]any) { k["alg"] = "ES384" },
		"empty key ID":      func(k map[string]any) { k["kid"] = "" },
		"key ID with slash": func(k map[string]any) { k["kid"] = "../key" },
		"short coordinate":  func(k map[string]any) { k["x"] = "AQAB" },
		"point off curve":   func(k map[string]any) { k["y"] = k["x"] },
		"private parameter": func(k map[string]any) { k["d"] = "secret" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			key := validKey(t)
			mutate(key)
			data, err := json.Marshal(map[string]any{"keys": []any{key}})
			require.NoError(t, err)

			_, err = licensing.ParseKeySet(data)
			require.Error(t, err)
		})
	}

	t.Run("duplicate key IDs", func(t *testing.T) {
		duplicate := licensingtest.NewIssuer(issuer.KeyID)
		_, err := licensing.ParseKeySet(licensingtest.JWKS(issuer, duplicate))
		require.Error(t, err)
	})

	t.Run("empty key set", func(t *testing.T) {
		_, err := licensing.ParseKeySet([]byte(`{"keys":[]}`))
		require.Error(t, err)
	})

	t.Run("merge rejects a reused key ID", func(t *testing.T) {
		first := licensingtest.KeySet(issuer)
		second := licensingtest.KeySet(licensingtest.NewIssuer(issuer.KeyID))
		_, err := first.Merge(second)
		require.Error(t, err)
	})
}
