package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type LocalProvider struct {
	tx        *gorm.DB
	encryptor crypto.Encryptor
	record    *models.Secret
}

func NewLocalProvider(tx *gorm.DB, encryptor crypto.Encryptor, record *models.Secret) *LocalProvider {
	return &LocalProvider{
		tx:        tx,
		encryptor: encryptor,
		record:    record,
	}
}

func (p *LocalProvider) Load(ctx context.Context) (map[string]string, error) {
	name := p.record.Name
	decrypted, err := p.encryptor.Decrypt(context.TODO(), p.record.Data, []byte(name))
	if err != nil {
		return nil, fmt.Errorf("error decrypting secret %s: %v", name, err)
	}

	var values map[string]string
	err = json.Unmarshal(decrypted, &values)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling secret %s: %v", name, err)
	}

	return values, nil
}
