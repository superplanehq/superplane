package contexts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

// SecretResolver loads the keys of a named organization secret for use inside
// the expression engine. A nil resolver puts the expression engine into
// deferred mode: secrets() placeholders are left untouched so they can be
// resolved later, at execution time, with a non-nil resolver.
type SecretResolver interface {
	Resolve(name string) (map[string]string, error)
}

// RuntimeSecretResolver looks up secrets in the database, decrypts them and
// returns their keys. It is used at execution time only; the resolved values
// are kept in-memory and never written back to the stored configuration.
type RuntimeSecretResolver struct {
	tx         *gorm.DB
	encryptor  crypto.Encryptor
	domainType string
	domainID   uuid.UUID
}

// NewRuntimeSecretResolver returns a resolver scoped to the given domain.
// Component executions and execution hooks always run within an organization
// transaction, so this is the resolver used everywhere except in the
// deferred (queue) phase.
func NewRuntimeSecretResolver(tx *gorm.DB, encryptor crypto.Encryptor, domainType string, domainID uuid.UUID) *RuntimeSecretResolver {
	return &RuntimeSecretResolver{
		tx:         tx,
		encryptor:  encryptor,
		domainType: domainType,
		domainID:   domainID,
	}
}

// Resolve fetches and decrypts the named secret, returning its keys as a map.
// Missing secrets surface as an error whose message contains the secret name
// and the phrase "not found".
func (r *RuntimeSecretResolver) Resolve(name string) (map[string]string, error) {
	provider, err := secrets.NewProvider(r.tx, r.encryptor, name, r.domainType, r.domainID)
	if err != nil {
		return nil, fmt.Errorf("secret %q: %v", name, err)
	}

	values, err := provider.Load(context.Background())
	if err != nil {
		return nil, fmt.Errorf("secret %q: %v", name, err)
	}

	return values, nil
}

// ResolveStoredConfiguration runs a runtime-mode builder over the stored
// configuration map, returning the in-memory result with secrets() calls
// resolved. The stored map itself is left untouched, so the secret values
// never reach the database or logs that emit the persisted configuration.
func ResolveStoredConfiguration(builder *NodeConfigurationBuilder, stored map[string]any) (map[string]any, error) {
	if stored == nil {
		return nil, nil
	}
	return builder.Build(stored)
}
