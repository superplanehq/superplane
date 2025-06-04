package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type LocalProvider struct {
	tx      *gorm.DB
	options Options
}

func NewLocalProvider(tx *gorm.DB, options Options) *LocalProvider {
	return &LocalProvider{
		tx:      tx,
		options: options,
	}
}

func (p *LocalProvider) Get(ctx context.Context) (map[string]string, error) {
	name := p.options.SecretName
	decrypted, err := p.options.Encryptor.Decrypt(context.TODO(), p.options.SecretData, []byte(name))
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
