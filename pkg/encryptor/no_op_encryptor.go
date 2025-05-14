package encryptor

import "context"

type NoOpEncryptor struct {
}

func (e *NoOpEncryptor) Encrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	return data, nil
}

func (e *NoOpEncryptor) Decrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	return data, nil
}
