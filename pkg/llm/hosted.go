package llm

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
)

func APIKeyAAD(provider string) []byte {
	return []byte("hosted_llm_key:" + provider)
}

func EncryptAPIKey(ctx context.Context, encryptor crypto.Encryptor, provider, apiKey string) ([]byte, error) {
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}
	return encryptor.Encrypt(ctx, []byte(apiKey), APIKeyAAD(provider))
}

func DecryptAPIKey(ctx context.Context, encryptor crypto.Encryptor, provider string, ciphertext []byte) (string, error) {
	if encryptor == nil {
		return "", fmt.Errorf("encryptor is required")
	}
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("hosted %s API key is missing", provider)
	}
	plain, err := encryptor.Decrypt(ctx, ciphertext, APIKeyAAD(provider))
	if err != nil {
		return "", fmt.Errorf("decrypt hosted %s API key: %w", provider, err)
	}
	return string(plain), nil
}
