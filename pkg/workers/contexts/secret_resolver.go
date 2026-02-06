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

interface SecretResolver {
	Resolve(name string) (any, error)
}

struct DeferedSecretResolver type {
}

func (r *DeferedSecretResolver) Resolve(name string) (any, error) {
	return "secret("+name+")", nil
}

struct RuntimeSecretResolver type {
	tx         *gorm.DB
	encryptor  crypto.Encryptor
	domainType string
	domainID   uuid.UUID
}

func (r *SecretResolver) Resolve(name string) (any, error) {
	provider, err := secrets.NewProvider(r.tx, r.encryptor, name, r.domainType, r.domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to find secret (%s)", name)
	}

	return provider.Load(context.Background())
}

// // CanResolveSecrets reports whether this resolver can resolve secrets.
// func (r *SecretResolver) CanResolveSecrets() bool {
// 	if r == nil {
// 		return false
// 	}
// 	return r.mode == ResolveSecretsImmediately && r.tx != nil && r.encryptor != nil
// }

// // ShouldLeaveSecretUnresolved reports whether an expression containing secrets() should be left as-is (not evaluated).
// func ShouldLeaveSecretUnresolved(resolver *SecretResolver, innerExpr string) bool {
// 	return IsInjectingUnresolvedSecret(innerExpr) && (resolver == nil || !resolver.CanResolveSecrets())
// }

// // IsInjectingUnresolvedSecret reports whether the expression contains a call to secrets(...).
// func IsInjectingUnresolvedSecret(expression string) bool {
// 	tree, err := parser.Parse(expression)
// 	if err != nil {
// 		return false
// 	}
// 	collector := &secretsCallCollector{}
// 	ast.Walk(&tree.Node, collector)
// 	return collector.found
// }

// type secretsCallCollector struct {
// 	found bool
// }

// func (c *secretsCallCollector) Visit(node *ast.Node) {
// 	if c.found {
// 		return
// 	}
// 	call, ok := (*node).(*ast.CallNode)
// 	if !ok {
// 		return
// 	}
// 	if id, ok := call.Callee.(*ast.IdentifierNode); ok && id.Value == "secrets" {
// 		c.found = true
// 	}
// }
