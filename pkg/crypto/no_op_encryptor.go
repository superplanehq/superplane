package crypto

import "context"

type NoOpEncryptor struct {
}

func NewNoOpEncryptor() Encryptor {
	return &NoOpEncryptor{}
}

func (e *NoOpEncryptor) Encrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	return data, nil
}

func (e *NoOpEncryptor) Decrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	return data, nil
}
