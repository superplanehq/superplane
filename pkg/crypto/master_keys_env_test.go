package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMasterKeysFromEnvUsesFirstEntryAndKeepsOldKey(t *testing.T) {
	first := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	old := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	t.Setenv("ENCRYPTION_MASTER_KEYS", "env://new:"+first+",env://old:"+old)
	masters, primary, err := MasterKeysFromEnv(t.Context())
	require.NoError(t, err)
	require.Equal(t, "env://new", primary)
	require.True(t, masters.Has(primary))
	require.True(t, masters.Has("env://old"))

	wrapped, err := masters.Wrap(t.Context(), "env://old", []byte("data key"), []byte("key ID"))
	require.NoError(t, err)
	plain, err := masters.Unwrap(t.Context(), "env://old", wrapped, []byte("key ID"))
	require.NoError(t, err)
	require.Equal(t, "data key", string(plain))
}

func TestMasterKeysFromEnvRejectsInvalidEntries(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	for _, configured := range []string{
		"env://new:not-base64",
		"env://new:" + key + ",env://new:" + key,
		"env://bad.id:" + key,
		"unknown://key",
	} {
		t.Run(configured[:min(len(configured), 20)], func(t *testing.T) {
			t.Setenv("ENCRYPTION_MASTER_KEYS", configured)
			_, _, err := MasterKeysFromEnv(t.Context())
			require.Error(t, err)
		})
	}
}
