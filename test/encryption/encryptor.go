package encryption

import (
	"bytes"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type dataKeys struct {
	key crypto.DataKey
}

func (k dataKeys) Active(context.Context) (crypto.DataKey, error) {
	return k.key, nil
}

func (k dataKeys) Find(_ context.Context, id uuid.UUID) (crypto.DataKey, error) {
	if id != k.key.ID {
		return crypto.DataKey{}, errors.New("unknown test data key")
	}
	return k.key, nil
}

func NewEncryptor() crypto.Encryptor {
	return crypto.NewVersionedEncryptor(dataKeys{
		key: crypto.DataKey{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Material: bytes.Repeat([]byte{1}, 32)},
	})
}
