package oidc

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type RSAProvider struct {
	privateKey  *rsa.PrivateKey
	publicKeys  map[string]*rsa.PublicKey
	publicJWKs  []PublicJWK
	activeKeyID string
}

type keyEntry struct {
	name string
	key  *rsa.PrivateKey
}

func NewProviderFromKeyDir(keysPath string) (Provider, error) {
	entries, err := os.ReadDir(keysPath)
	if err != nil {
		return nil, err
	}

	var keys []keyEntry
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("stat key %s: %w", entry.Name(), err)
			}
			if !info.Mode().IsRegular() {
				continue
			}
		}
		keyPath := filepath.Join(keysPath, entry.Name())
		privateKeyPEM, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", entry.Name(), err)
		}
		privateKey, err := parsePrivateKeyPEM(privateKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("parse key %s: %w", entry.Name(), err)
		}
		keys = append(keys, keyEntry{name: entry.Name(), key: privateKey})
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no OIDC keys found in %s", keysPath)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].name < keys[j].name
	})

	activeKey := keys[len(keys)-1].key
	return newSignerFromKeys(activeKey, keys)
}

func (s *RSAProvider) PublicJWKs() []PublicJWK {
	return s.publicJWKs
}

func (s *RSAProvider) Sign(subject string, duration time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(duration).Unix(),
		"sub": subject,
	})

	token.Header["kid"] = s.activeKeyID
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func newSignerFromKeys(activeKey *rsa.PrivateKey, keys []keyEntry) (Provider, error) {
	publicKeys := make(map[string]*rsa.PublicKey, len(keys))
	publicJWKs := make([]PublicJWK, 0, len(keys))

	for _, entry := range keys {
		publicKey := &entry.key.PublicKey
		keyID, err := keyIDForPublicKey(publicKey)
		if err != nil {
			return nil, err
		}
		if _, exists := publicKeys[keyID]; exists {
			return nil, fmt.Errorf("duplicate OIDC key id: %s", keyID)
		}
		publicKeys[keyID] = publicKey
		publicJWKs = append(publicJWKs, publicJWKFromKey(keyID, publicKey))
	}

	activeKeyID, err := keyIDForPublicKey(&activeKey.PublicKey)
	if err != nil {
		return nil, err
	}
	if _, exists := publicKeys[activeKeyID]; !exists {
		return nil, errors.New("active OIDC key is not registered")
	}

	return &RSAProvider{
		privateKey:  activeKey,
		publicKeys:  publicKeys,
		publicJWKs:  publicJWKs,
		activeKeyID: activeKeyID,
	}, nil
}

func parsePrivateKeyPEM(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}

func keyIDForPublicKey(publicKey *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	kidHash := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(kidHash[:]), nil
}

func publicJWKFromKey(keyID string, publicKey *rsa.PublicKey) PublicJWK {
	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	eBytes := bigIntFromInt(publicKey.E).Bytes()
	e := base64.RawURLEncoding.EncodeToString(eBytes)
	return PublicJWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: keyID,
		N:   n,
		E:   e,
	}
}

func bigIntFromInt(value int) *big.Int {
	return new(big.Int).SetInt64(int64(value))
}
