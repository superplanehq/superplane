package mcpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestListConsentWorkspacesListsWorkspaceWithFactories(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factory, err := models.CreateFactory(db, r.Organization.ID, "Hidden", "", "HID")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	visible, err := ListConsentWorkspacesForOrganization(ctx, r.Account, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, visible, 1)
	assert.Equal(t, factory.ID.String(), visible[0].FactoryID)
	assert.Equal(t, "Hidden", visible[0].Label)
}

func TestBuildConsentViewSelectsTheOnlyOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	factory, err := models.CreateFactory(database.DB(ctx), r.Organization.ID, "Hidden", "", "HID")
	require.NoError(t, err)

	view, err := BuildConsentView(ctx, r.Account, "", "")
	require.NoError(t, err)
	require.True(t, view.ShowWorkspaces)
	assert.Equal(t, r.Organization.ID.String(), view.OrganizationID)
	require.Len(t, view.Workspaces, 1)
	assert.Equal(t, factory.ID.String(), view.Workspaces[0].FactoryID)
}

func TestResolveConsentWorkspacesRequiresFactories(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factory, err := models.CreateFactory(db, r.Organization.ID, "Hidden", "", "HID")
	require.NoError(t, err)
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))

	_, _, _, err = ResolveConsentWorkspaces(ctx, r.Account, r.Organization.ID.String(), []string{factory.ID.String()})
	require.EqualError(t, err, consentOrganizationError)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	userID, orgID, factoryIDs, err := ResolveConsentWorkspaces(ctx, r.Account, r.Organization.ID.String(), []string{factory.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, r.User, userID)
	assert.Equal(t, r.Organization.ID, orgID)
	assert.Equal(t, []uuid.UUID{factory.ID}, factoryIDs)
}

func TestResolveConsentWorkspacesAcceptsTwoAndRejectsOutsideOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	first, err := models.CreateFactory(db, r.Organization.ID, "One", "", "ONE")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, "Two", "", "TWO")
	require.NoError(t, err)
	otherOrg := support.CreateOrganization(t, r, r.User)
	require.NoError(t, models.EnableExperimentalFeature(otherOrg.ID, features.FeatureFactories))
	_, err = models.CreateUser(otherOrg.ID, r.Account.ID, r.Account.Email, r.Account.Name)
	require.NoError(t, err)
	outside, err := models.CreateFactory(db, otherOrg.ID, "Outside", "", "OUT")
	require.NoError(t, err)

	_, _, ids, err := ResolveConsentWorkspaces(ctx, r.Account, r.Organization.ID.String(), []string{second.ID.String(), first.ID.String()})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{first.ID, second.ID}, ids)
	assert.Equal(t, ids, mustSorted(t, ids))

	_, _, _, err = ResolveConsentWorkspaces(ctx, r.Account, r.Organization.ID.String(), []string{outside.ID.String()})
	require.EqualError(t, err, consentWorkspaceInvalid)
	_, _, _, err = ResolveConsentWorkspaces(ctx, r.Account, "", nil)
	require.EqualError(t, err, consentOrganizationError)
	_, _, _, err = ResolveConsentWorkspaces(ctx, r.Account, r.Organization.ID.String(), nil)
	require.EqualError(t, err, consentWorkspaceRequired)
}

func mustSorted(t *testing.T, ids []uuid.UUID) []uuid.UUID {
	t.Helper()
	sorted, err := models.NormalizeFactoryIDs(ids)
	require.NoError(t, err)
	return sorted
}

func TestRenderConsentPageUsesFactoriesTheme(t *testing.T) {
	page, err := RenderConsentPage("Cursor", "token", ConsentView{
		Organizations:  []OrganizationOption{{ID: "org", Name: "Acme", Selected: true}},
		Workspaces:     []WorkspaceOption{{FactoryID: "abc", Label: "Acme / Factory"}},
		OrganizationID: "org",
		ShowWorkspaces: true,
	})
	require.NoError(t, err)
	html := string(page)
	assert.Contains(t, html, "--foreground: #26251e")
	assert.Contains(t, html, "font-family: \"Inter\"")
	assert.Contains(t, html, "Allow access to SuperPlane")
	assert.Contains(t, html, "Choose the organization, then the workspaces Cursor can use.")
	assert.Contains(t, html, `name="organization_id"`)
	assert.Contains(t, html, `name="workspaces_for" value="org"`)
	assert.Contains(t, html, `name="factory_id" value="abc"`)
	assert.Contains(t, html, "Acme / Factory")
	assert.NotContains(t, html, "<script")
}
