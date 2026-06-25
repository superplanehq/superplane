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

func (p *TestOIDCProvider) Validate(tokenString string) (map[string]any, error) {
	return map[string]any{
		"sub":          "execution:test",
		"aud":          "superplane-ci",
		"org_id":       "org-test",
		"canvas_id":    "canvas-test",
		"node_id":      "node-test",
		"execution_id": "exec-test",
		"component":    "semaphore.runWorkflow",
	}, nil
}
