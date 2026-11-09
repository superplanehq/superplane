package sentry

import (
	"github.com/superplanehq/superplane/pkg/core"
)

const IntegrationAuthTokenKey = "SENTRY_AUTH_TOKEN"

// ResolveSecrets gives generic components the same current token used by
// Sentry actions. NewClient refreshes hosted app tokens when needed.
func (s *Sentry) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return core.IntegrationSecrets{}, err
	}

	return core.IntegrationSecrets{
		Values: map[string][]byte{IntegrationAuthTokenKey: []byte(client.userToken)},
	}, nil
}
