package secrets

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	ProviderLocal = "local"
)

type Provider interface {
	Load(ctx context.Context) (map[string]string, error)
}

type Options struct {
	CanvasID   uuid.UUID
	SecretName string
	Encryptor  crypto.Encryptor
}

func NewProvider(tx *gorm.DB, encryptor crypto.Encryptor, name, domainType string, domainID uuid.UUID) (Provider, error) {
	secret, err := models.FindSecretByNameInTransaction(tx, domainType, domainID, name)
	if err != nil {
		return nil, fmt.Errorf("error finding secret %s: %v", name, err)
	}

	switch secret.Provider {
	case ProviderLocal:
		return NewLocalProvider(tx, encryptor, secret), nil
	default:
		return nil, fmt.Errorf("provider not supported: %s", secret.Provider)
	}
}
