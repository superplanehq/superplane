package secrets

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
)

const (
	ProviderLocal = "local"
)

type Provider interface {
	Get(ctx context.Context) (map[string]string, error)
}

type Options struct {
	CanvasID   uuid.UUID
	SecretName string
	SecretData []byte
	Encryptor  crypto.Encryptor
}

func NewProvider(provider string, options Options) (Provider, error) {
	switch provider {
	case ProviderLocal:
		return NewLocalProvider(database.Conn(), options), nil
	default:
		return nil, fmt.Errorf("provider not supported: %s", provider)
	}
}
