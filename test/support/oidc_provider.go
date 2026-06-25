package support

import (
	"time"

	"github.com/superplanehq/superplane/pkg/oidc"
)

type TestOIDCProvider struct{}

func NewOIDCProvider() oidc.Provider {
	return &TestOIDCProvider{}
}

func (p *TestOIDCProvider) PublicJWKs() []oidc.PublicJWK {
	return []oidc.PublicJWK{
		{
			Kty: "RSA",
			Use: "sig",
			Alg: "RS256",
			Kid: "test",
		},
	}
}

func (p *TestOIDCProvider) Sign(subject string, duration time.Duration, audience string, additionalClaims map[string]any) (string, error) {
	return "test", nil
}
