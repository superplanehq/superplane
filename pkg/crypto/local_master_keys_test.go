package crypto

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
)

func TestLocalMasterKeysSelectNamedKey(t *testing.T) {
	first := bytes.Repeat([]byte{1}, 32)
	second := bytes.Repeat([]byte{2}, 32)
	configured := "local-v1:" + base64.StdEncoding.EncodeToString(first) +
		",local-v2:" + base64.StdEncoding.EncodeToString(second)
	keys, err := ParseLocalMasterKeys(configured)
	if err != nil {
		t.Fatal(err)
	}

	wrapped, err := keys.Wrap(context.Background(), "local-v1", []byte("data key"), []byte("key-id"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Unwrap(context.Background(), "local-v2", wrapped, []byte("key-id")); err == nil {
		t.Fatal("another master key decrypted the value")
	}
	plain, err := keys.Unwrap(context.Background(), "local-v1", wrapped, []byte("key-id"))
	if err != nil || !bytes.Equal(plain, []byte("data key")) {
		t.Fatalf("unwrap: %q, %v", plain, err)
	}
}

func TestLocalMasterKeysRejectInvalidConfiguration(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	for _, configured := range []string{
		"",
		"local-v1:bad",
		"local-v1:" + valid + ",local-v1:" + valid,
		"bad:name:" + valid,
	} {
		if _, err := ParseLocalMasterKeys(configured); err == nil {
			t.Errorf("accepted invalid configuration %q", configured)
		}
	}
}
