package contexts

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

type SecretResolutionMode int

const (
	ResolveSecretsImmediately SecretResolutionMode = iota
	DeferSecretResolution
	NoOpSecretResolution
)

// SecretResolver resolves secret names during configuration build.
type SecretResolver struct {
	mode       SecretResolutionMode
	tx         *gorm.DB
	encryptor  crypto.Encryptor
	domainType string
	domainID   uuid.UUID
}

// NewDeferedSecretResolver returns a resolver that defers resolution (returns an error if Resolve is called).
func NewDeferedSecretResolver(mode SecretResolutionMode) SecretResolver {
	return SecretResolver{mode: DeferSecretResolution}
}

// NewImmediateSecretResolver returns a resolver that resolves secrets from the database.
func NewImmediateSecretResolver(tx *gorm.DB, encryptor crypto.Encryptor, domainType string, domainID uuid.UUID) SecretResolver {
	return SecretResolver{
		mode:       ResolveSecretsImmediately,
		tx:         tx,
		encryptor:  encryptor,
		domainType: domainType,
		domainID:   domainID,
	}
}

// NewNoOpSecretResolver returns a resolver that never resolves (for process_queue_context etc.).
func NewNoOpSecretResolver() SecretResolver {
	return SecretResolver{mode: NoOpSecretResolution}
}

// Resolve resolves a secret by name. Returns (value, nil) or (nil, error).
func (r *SecretResolver) Resolve(name string) (any, error) {
	switch r.mode {
	case DeferSecretResolution:
		return nil, fmt.Errorf("secret resolution is deferred")
	case NoOpSecretResolution:
		return nil, fmt.Errorf("secret resolution not available")
	case ResolveSecretsImmediately:
		if r.tx == nil || r.encryptor == nil {
			return nil, fmt.Errorf("secret resolver not configured for immediate resolution")
		}
		provider, err := secrets.NewProvider(r.tx, r.encryptor, name, r.domainType, r.domainID)
		if err != nil {
			return nil, err
		}
		return provider.Load(context.Background())
	default:
		return nil, fmt.Errorf("invalid secret resolution mode")
	}
}

// CanResolveSecrets reports whether this resolver can resolve secrets.
func (r *SecretResolver) CanResolveSecrets() bool {
	if r == nil {
		return false
	}
	return r.mode == ResolveSecretsImmediately && r.tx != nil && r.encryptor != nil
}

// ShouldLeaveSecretUnresolved reports whether an expression containing secrets() should be left as-is (not evaluated).
func ShouldLeaveSecretUnresolved(resolver *SecretResolver, innerExpr string) bool {
	return IsInjectingUnresolvedSecret(innerExpr) && (resolver == nil || !resolver.CanResolveSecrets())
}

// IsInjectingUnresolvedSecret reports whether the expression contains a call to secrets(...).
func IsInjectingUnresolvedSecret(expression string) bool {
	tree, err := parser.Parse(expression)
	if err != nil {
		return false
	}
	collector := &secretsCallCollector{}
	ast.Walk(&tree.Node, collector)
	return collector.found
}

type secretsCallCollector struct {
	found bool
}

func (c *secretsCallCollector) Visit(node *ast.Node) {
	if c.found {
		return
	}
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}
	if id, ok := call.Callee.(*ast.IdentifierNode); ok && id.Value == "secrets" {
		c.found = true
	}
}
