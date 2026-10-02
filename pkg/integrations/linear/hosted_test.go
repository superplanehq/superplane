package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__callbackRedirectURL(t *testing.T) {
	integrationID := "11111111-1111-1111-1111-111111111111"
	settingsURL := "https://app.example/org-1/settings/integrations/" + integrationID

	t.Run("appends the connection id to a stored return path", func(t *testing.T) {
		got := callbackRedirectURL(core.HTTPRequestContext{
			BaseURL: "https://app.example",
			Integration: &contexts.IntegrationContext{
				IntegrationID: integrationID,
				Metadata: Metadata{
					SetupReturnPath: "/org-1/workspaces/acme/lines/line-1/setup/linear",
				},
			},
		}, settingsURL)

		assert.Equal(t,
			"https://app.example/org-1/workspaces/acme/lines/line-1/setup/linear?linearIntegrationId="+integrationID,
			got,
		)
	})

	t.Run("falls back to settings when no return path is stored", func(t *testing.T) {
		got := callbackRedirectURL(core.HTTPRequestContext{
			BaseURL: "https://app.example",
			Integration: &contexts.IntegrationContext{
				IntegrationID: integrationID,
			},
		}, settingsURL)

		assert.Equal(t, settingsURL, got)
	})
}
