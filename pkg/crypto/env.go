package crypto

import (
	"fmt"
	"os"
)

// FromEnv returns the process encryptor. Tests and local development can
// set NO_ENCRYPTION=yes to skip AES.
func FromEnv() (Encryptor, error) {
	if os.Getenv("NO_ENCRYPTION") == "yes" {
		return NewNoOpEncryptor(), nil
	}
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required")
	}
	return NewAESGCMEncryptor([]byte(key)), nil
}
