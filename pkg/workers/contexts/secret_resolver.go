package contexts

import (
	"strings"
)

type SecretResolutionMode int

const (
	ResolveSecretsImmediately SecretResolutionMode = iota	
	DeferSecretResolution
}

type SecretResolver struct {
	mode SecretResolutionMode
	provider *secrets.Provider
}

func NewDeferedSecretResolver(mode SecretResolutionMode) *SecretResolver {
	return &SecretResolver{mode: DeferSecretResolution}
}

func NewImmediateSecretResolver(provider *secrets.Provider) *SecretResolver {
	return &SecretResolver{
		mode: ResolveSecretsImmediately,
		provider: provider,
	}
}

func (r *SecretResolver) ResolveSecret(secretName string) map[string]string {
	if r.mode == DeferSecretResolution {
		return "{{ secrets(" + secretName + ") }}"
	}

	secretValue, err := r.provider.GetSecret(secretName)
	if err != nil {
		return fmt.Errorf("failed to resolve secret '%s'", secretName)
	}

	return secretValue
}

func (r *SecretResolver) IsInjectingUnresolvedSecret(expression string) bool {
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
