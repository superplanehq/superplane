package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewProviderFromKeyDirLoadsSymlinkedKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataDir := filepath.Join(dir, "..data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}

	key := mustGenerateKey(t)
	keyPath := filepath.Join(dataDir, "1769117887.pem")
	if err := os.WriteFile(keyPath, pemEncodeKey(key), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	linkPath := filepath.Join(dir, "1769117887.pem")
	if err := os.Symlink(filepath.Join("..data", "1769117887.pem"), linkPath); err != nil {
		t.Fatalf("symlink key: %v", err)
	}

	provider, err := NewProviderFromKeyDir(dir)
	if err != nil {
		t.Fatalf("NewProviderFromKeyDir: %v", err)
	}
	if len(provider.PublicJWKs()) != 1 {
		t.Fatalf("expected 1 public JWK, got %d", len(provider.PublicJWKs()))
	}
	if _, err := provider.Sign("subject", time.Minute); err != nil {
		t.Fatalf("Sign: %v", err)
	}
}

func TestNewProviderFromKeyDirSkipsNonRegularFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	key := mustGenerateKey(t)
	keyPath := filepath.Join(dir, "1769117887.pem")
	if err := os.WriteFile(keyPath, pemEncodeKey(key), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	provider, err := NewProviderFromKeyDir(dir)
	if err != nil {
		t.Fatalf("NewProviderFromKeyDir: %v", err)
	}
	if len(provider.PublicJWKs()) != 1 {
		t.Fatalf("expected 1 public JWK, got %d", len(provider.PublicJWKs()))
	}
}

func mustGenerateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func pemEncodeKey(key *rsa.PrivateKey) []byte {
	der := x509.MarshalPKCS1PrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})
}
