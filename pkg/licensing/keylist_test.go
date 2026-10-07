package licensing_test

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

func TestVerifyKeyList(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026-01")
	roots := licensingtest.KeySet(root)

	t.Run("accepts a list signed by a root key", func(t *testing.T) {
		list, err := licensing.VerifyKeyList(root.KeyList(3, signer), roots)
		require.NoError(t, err)
		assert.Equal(t, int64(3), list.Version)
		assert.Equal(t, []string{"signing-2026-01"}, list.Keys.KeyIDs())
	})

	cases := map[string][]byte{
		"another root":   licensingtest.NewIssuer("root-2026").KeyList(3, signer),
		"a license type": root.SignWithHeader(licensingtest.Header(root.KeyID), licensingtest.KeyListClaims(3, signer)),
		"a license":      signer.License(licensing.FeatureGroups),
		"an extra header": root.SignWithHeader(
			map[string]any{"alg": "ES256", "kid": root.KeyID, "typ": licensing.KeyListType, "jku": "https://example.com"},
			licensingtest.KeyListClaims(3, signer),
		),
		"a root key ID as a signing key": root.KeyList(3, licensingtest.NewIssuer("root-2026")),
		"duplicate key IDs":              root.KeyList(3, signer, licensingtest.NewIssuer("signing-2026-01")),
		"no keys":                        root.KeyList(3),
		"oversized":                      bytes.Repeat([]byte("a"), licensing.MaxKeyListBytes+1),
		"malformed":                      []byte("a.b"),
	}

	claims := map[string]func(map[string]any){
		"another issuer":   func(c map[string]any) { c["iss"] = "https://example.com" },
		"another audience": func(c map[string]any) { c["aud"] = licensing.ExpectedAudience },
		"version zero":     func(c map[string]any) { c["version"] = 0 },
		"no version":       func(c map[string]any) { delete(c, "version") },
		"no issue time":    func(c map[string]any) { delete(c, "iat") },
		"an extra claim":   func(c map[string]any) { c["exp"] = 1 },
	}
	for name, mutate := range claims {
		c := licensingtest.KeyListClaims(3, signer)
		mutate(c)
		cases[name] = root.SignWithHeader(licensingtest.KeyListHeader(root.KeyID), c)
	}

	for name, raw := range cases {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := licensing.VerifyKeyList(raw, roots)
			require.ErrorIs(t, err, licensing.ErrInvalidKeyList)
		})
	}

	t.Run("rejects a tampered list", func(t *testing.T) {
		segments := strings.Split(string(root.KeyList(3, signer)), ".")
		other := strings.Split(string(root.KeyList(4, signer)), ".")
		tampered := segments[0] + "." + other[1] + "." + segments[2]
		_, err := licensing.VerifyKeyList([]byte(tampered), roots)
		require.ErrorIs(t, err, licensing.ErrInvalidKeyList)

		forged := segments[0] + "." + segments[1] + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		_, err = licensing.VerifyKeyList([]byte(forged), roots)
		require.ErrorIs(t, err, licensing.ErrInvalidKeyList)
	})
}

func TestKeyStore(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	first := licensingtest.NewIssuer("signing-2026-01")
	second := licensingtest.NewIssuer("signing-2026-02")
	roots := licensingtest.KeySet(root)

	verified := func(t *testing.T, version int64, signers ...*licensingtest.Issuer) *licensing.KeyList {
		t.Helper()
		list, err := licensing.VerifyKeyList(root.KeyList(version, signers...), roots)
		require.NoError(t, err)
		return list
	}

	t.Run("starts with the bootstrap list", func(t *testing.T) {
		store, err := licensing.NewKeyStore(roots, root.KeyList(2, first), nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), store.Current().Version)
		_, ok := store.PublicKey(first.KeyID)
		assert.True(t, ok)
	})

	t.Run("rejects an invalid bootstrap list", func(t *testing.T) {
		_, err := licensing.NewKeyStore(roots, first.KeyList(2, first), nil)
		require.ErrorIs(t, err, licensing.ErrInvalidKeyList)
	})

	t.Run("trusts nothing without a key list", func(t *testing.T) {
		store, err := licensing.NewKeyStore(roots, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, store.Current())
		_, ok := store.PublicKey(first.KeyID)
		assert.False(t, ok)
	})

	t.Run("accepts only newer lists", func(t *testing.T) {
		store, err := licensing.NewKeyStore(roots, root.KeyList(2, first), nil)
		require.NoError(t, err)

		changed, err := store.Update(verified(t, 2, first, second))
		require.NoError(t, err)
		assert.False(t, changed, "a list with the same version does not change the keys")

		changed, err = store.Update(verified(t, 3, second))
		require.NoError(t, err)
		assert.True(t, changed)
		_, ok := store.PublicKey(first.KeyID)
		assert.False(t, ok, "a newer list can remove a key")

		_, err = store.Update(verified(t, 2, first))
		require.ErrorIs(t, err, licensing.ErrKeyListDowngrade)
		_, ok = store.PublicKey(first.KeyID)
		assert.False(t, ok, "an older list cannot restore a removed key")
	})

	t.Run("trusts extra keys in addition to the list", func(t *testing.T) {
		extra := licensingtest.NewIssuer("development")
		store, err := licensing.NewKeyStore(roots, root.KeyList(2, first), licensingtest.KeySet(extra))
		require.NoError(t, err)

		_, ok := store.PublicKey(extra.KeyID)
		assert.True(t, ok)
		_, ok = store.PublicKey(first.KeyID)
		assert.True(t, ok)
	})
}
