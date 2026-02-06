package contexts

// SecretResolveAction tells the builder what to do with a segment after calling SecretResolver.Resolve.
type SecretResolveAction int

const (
	SecretResolveUseValue     SecretResolveAction = iota // replace segment with value
	SecretResolveKeepOriginal                           // keep segment unchanged
	SecretResolveNormal                                 // not a secret; resolve with resolveExpression
)

// SecretResolver is given to NodeConfigurationBuilder. The builder calls Resolve for each
// {{ ... }} segment; the resolver decides whether it's a secret expression and what to do.
type SecretResolver interface {
	Resolve(expression string) (value any, action SecretResolveAction, err error)
}

// NoOpSecretResolver never resolves secrets. Secret expressions get KeepOriginal; others Normal.
type NoOpSecretResolver struct{}

func (NoOpSecretResolver) Resolve(expression string) (any, SecretResolveAction, error) {
	if expressionContainsSecrets(expression) {
		return nil, SecretResolveKeepOriginal, nil
	}
	return nil, SecretResolveNormal, nil
}

// RuntimeSecretResolver evaluates secret expressions at runtime using the given LoadSecret.
type RuntimeSecretResolver struct {
	Builder    *NodeConfigurationBuilder
	LoadSecret func(name string) (map[string]string, error)
}

func (r *RuntimeSecretResolver) Resolve(expression string) (any, SecretResolveAction, error) {
	if !expressionContainsSecrets(expression) {
		return nil, SecretResolveNormal, nil
	}
	value, err := r.Builder.evaluateWithSecrets(expression, r.LoadSecret)
	if err != nil {
		return nil, 0, err
	}
	return value, SecretResolveUseValue, nil
}
