package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestListConsentWorkspacesRequiresMCPServerFlag(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	_, err := models.CreateFactory(db, r.Organization.ID, "Hidden", "", "HID")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	hidden, err := ListConsentWorkspaces(ctx, r.Account)
	require.NoError(t, err)
	assert.Empty(t, hidden)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	visible, err := ListConsentWorkspaces(ctx, r.Account)
	require.NoError(t, err)
	require.Len(t, visible, 1)
	assert.Contains(t, visible[0].Label, "Hidden")
}

func TestRenderConsentPageUsesFactoriesTheme(t *testing.T) {
	page, err := RenderConsentPage("Cursor", "token", "", []WorkspaceOption{
		{FactoryID: "abc", Label: "Acme / Factory"},
	})
	require.NoError(t, err)
	html := string(page)
	assert.Contains(t, html, "--foreground: #26251e")
	assert.Contains(t, html, "font-family: \"Inter\"")
	assert.Contains(t, html, "Allow access to SuperPlane")
	assert.Contains(t, html, "Acme / Factory")
}
