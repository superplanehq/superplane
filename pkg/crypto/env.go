package crypto

import (
	"fmt"
	"os"
)

// FromEnv returns the legacy encryptor for command-line tools that do not
// initialize the versioned key store.
func FromEnv() (Encryptor, error) {
	encryptor, err := LegacyDecryptorFromEnv()
	if err != nil {
		return nil, err
	}
	if encryptor == nil {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required")
	}
	return encryptor, nil
}

// LegacyDecryptorFromEnv reads unmarked values written before versioned
// encryption. Plaintext is an explicit development or demo conversion mode.
func LegacyDecryptorFromEnv() (Encryptor, error) {
	switch os.Getenv("ENCRYPTION_LEGACY_FORMAT") {
	case "plaintext":
		if os.Getenv("APP_ENV") != "development" && os.Getenv("SUPERPLANE_INSTALLATION_TYPE") != "demo" {
			return nil, fmt.Errorf("plaintext legacy format is only available in development or demo installations")
		}
		return NewNoOpEncryptor(), nil
	case "", "aes-gcm":
	default:
		return nil, fmt.Errorf("unsupported ENCRYPTION_LEGACY_FORMAT")
	}
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, nil
	}
	return NewAESGCMEncryptor([]byte(key)), nil
}
