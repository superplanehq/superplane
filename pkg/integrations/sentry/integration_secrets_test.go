package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSentryExportsConnectedTokenForGenericRequests(t *testing.T) {
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"baseUrl": "https://sentry.example.com", "userToken": "personal-token"},
		Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
	}

	secrets, err := (&Sentry{}).ResolveSecrets(core.IntegrationSecretContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: integration,
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("personal-token"), secrets.Values["SENTRY_AUTH_TOKEN"])
}
