package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

func TestClientHostKeyCallback(t *testing.T) {
	trustedKey := newTestPublicKey(t)
	otherKey := newTestPublicKey(t)
	fingerprint := gossh.FingerprintSHA256(trustedKey)

	clients := map[string]*Client{
		"password":    NewClientPassword("example.com", 22, "root", []byte("password")),
		"private key": NewClientKey("example.com", 22, "root", nil, nil),
	}

	for name, client := range clients {
		client.HostKeyFingerprint = fingerprint

		t.Run(name+" accepts matching key", func(t *testing.T) {
			err := client.hostKeyCallback()("example.com:22", nil, trustedKey)
			require.NoError(t, err)
		})

		t.Run(name+" rejects different key", func(t *testing.T) {
			err := client.hostKeyCallback()("example.com:22", nil, otherKey)
			require.Error(t, err)
			assert.ErrorIs(t, err, errHostKeyFingerprintMismatch)
			assert.Contains(t, err.Error(), "expected "+fingerprint)
			assert.Contains(t, err.Error(), "got "+gossh.FingerprintSHA256(otherKey))
		})
	}
}

func TestClientHostKeyCallbackAllowsLegacyConfiguration(t *testing.T) {
	client := NewClientPassword("example.com", 22, "root", []byte("password"))

	require.NoError(t, client.hostKeyCallback()("example.com:22", nil, newTestPublicKey(t)))
}

func TestSSHCommandDoesNotRetryHostKeyMismatch(t *testing.T) {
	command := &SSHCommand{}
	err := fmt.Errorf("failed to dial: %w", errHostKeyFingerprintMismatch)

	assert.False(t, command.isConnectError(err))
	assert.True(t, errors.Is(err, errHostKeyFingerprintMismatch))
}

func newTestPublicKey(t *testing.T) gossh.PublicKey {
	t.Helper()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	key, err := gossh.NewPublicKey(publicKey)
	require.NoError(t, err)
	return key
}
