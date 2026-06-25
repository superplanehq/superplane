package oidc

import (
	"time"
)

type Provider interface {
	Sign(subject string, duration time.Duration, audience string, additionalClaims map[string]any) (string, error)
	PublicJWKs() []PublicJWK
}

type PublicJWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}
