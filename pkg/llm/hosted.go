package llm

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
)

func APIKeyAAD(provider string) []byte {
	return []byte("hosted_llm_key:" + provider)
}

func ManagementKeyAAD(provider string) []byte {
	return []byte("hosted_llm_mgmt_key:" + provider)
}

func EncryptAPIKey(ctx context.Context, encryptor crypto.Encryptor, provider, apiKey string) ([]byte, error) {
	return encryptHostedSecret(ctx, encryptor, APIKeyAAD(provider), apiKey)
}

func DecryptAPIKey(ctx context.Context, encryptor crypto.Encryptor, provider string, ciphertext []byte) (string, error) {
	return decryptHostedSecret(ctx, encryptor, APIKeyAAD(provider), ciphertext, fmt.Sprintf("hosted %s API key is missing", provider), fmt.Sprintf("decrypt hosted %s API key", provider))
}

func EncryptManagementKey(ctx context.Context, encryptor crypto.Encryptor, provider, managementKey string) ([]byte, error) {
	return encryptHostedSecret(ctx, encryptor, ManagementKeyAAD(provider), managementKey)
}

func DecryptManagementKey(ctx context.Context, encryptor crypto.Encryptor, provider string, ciphertext []byte) (string, error) {
	return decryptHostedSecret(ctx, encryptor, ManagementKeyAAD(provider), ciphertext, fmt.Sprintf("hosted %s provisioning API key is missing", provider), fmt.Sprintf("decrypt hosted %s provisioning API key", provider))
}

func encryptHostedSecret(ctx context.Context, encryptor crypto.Encryptor, aad []byte, secret string) ([]byte, error) {
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}
	return encryptor.Encrypt(ctx, []byte(secret), aad)
}

func decryptHostedSecret(ctx context.Context, encryptor crypto.Encryptor, aad, ciphertext []byte, missingMessage, decryptMessage string) (string, error) {
	if encryptor == nil {
		return "", fmt.Errorf("encryptor is required")
	}
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("%s", missingMessage)
	}
	plain, err := encryptor.Decrypt(ctx, ciphertext, aad)
	if err != nil {
		return "", fmt.Errorf("%s: %w", decryptMessage, err)
	}
	return string(plain), nil
}
