package secrets

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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

func NewProvider(encryptor crypto.Encryptor, name string, canvasId string) (Provider, error) {
	secret, err := models.FindSecretByName(canvasId, name)
	if err != nil {
		return nil, fmt.Errorf("error finding secret %s: %v", name, err)
	}

	switch secret.Provider {
	case ProviderLocal:
		return NewLocalProvider(database.Conn(), encryptor, secret), nil
	default:
		return nil, fmt.Errorf("provider not supported: %s", secret.Provider)
	}
}
