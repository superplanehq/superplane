package crypto

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

// To avoid always going to the /.well-known/openid-configuration endpoint,
// we rely on the way go-oidc caches public keys in oidc.IDTokenVerifier,
// and we keep a map of verifiers in memory for our integrations.
type OIDCVerifier struct {
	verifiers map[string]*oidc.IDTokenVerifier
}

func NewOIDCVerifier() *OIDCVerifier {
	return &OIDCVerifier{
		verifiers: make(map[string]*oidc.IDTokenVerifier),
	}
}

func (v *OIDCVerifier) Verify(ctx context.Context, issuer, audience, token string) (*oidc.IDToken, error) {
	verifier, ok := v.verifiers[issuer]
	if ok {
		return verifier.Verify(ctx, token)
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: audience})
	v.verifiers[issuer] = verifier
	return verifier.Verify(ctx, token)
}
