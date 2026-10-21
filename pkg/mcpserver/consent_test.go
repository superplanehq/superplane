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

// joinedOrganization creates an organization where the registry account is an
// active human member.
func joinedOrganization(t *testing.T, r *support.ResourceRegistry) *models.Organization {
	t.Helper()
	organization := support.CreateOrganization(t, r, r.User)
	_, err := models.CreateUser(organization.ID, r.Account.ID, r.Account.Email, r.Account.Name)
	require.NoError(t, err)
	return organization
}

func enableConsentFeatures(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureSuperPlaneMCPServer))
}

// withGatedMCPServer makes the MCP server feature a per-organization flag, as
// it was before its release, so a test can turn it off for one organization.
func withGatedMCPServer(t *testing.T) {
	t.Helper()
	t.Cleanup(features.WithRegistryForTest([]features.Feature{
		{ID: features.FeatureFactories, Label: "Factories"},
		{ID: features.FeatureSuperPlaneMCPServer, Label: "MCP Server"},
	}))
}

func TestListConsentOrganizationsListsOnlyQualifyingOrganizations(t *testing.T) {
	withGatedMCPServer(t)
	r := support.Setup(t)
	ctx := t.Context()

	qualifying := joinedOrganization(t, r)
	enableConsentFeatures(t, qualifying.ID)
	noFactories := joinedOrganization(t, r)
	require.NoError(t, models.EnableExperimentalFeature(noFactories.ID, features.FeatureSuperPlaneMCPServer))
	require.NoError(t, models.DisableExperimentalFeature(noFactories.ID, features.FeatureFactories))
	noMCP := joinedOrganization(t, r)
	require.NoError(t, models.EnableExperimentalFeature(noMCP.ID, features.FeatureFactories))
	require.NoError(t, models.DisableExperimentalFeature(noMCP.ID, features.FeatureSuperPlaneMCPServer))
	notMember := support.CreateOrganization(t, r, r.User)
	enableConsentFeatures(t, notMember.ID)

	organizations, err := ListConsentOrganizations(ctx, r.Account)
	require.NoError(t, err)
	require.Len(t, organizations, 1)
	assert.Equal(t, qualifying.ID.String(), organizations[0].OrganizationID)
	assert.Equal(t, qualifying.Name, organizations[0].Name)
}

func TestListConsentWorkspacesListsOnlyTheOrganizationWorkspaces(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	other := joinedOrganization(t, r)
	enableConsentFeatures(t, other.ID)
	enableConsentFeatures(t, r.Organization.ID)

	mine, err := models.CreateFactory(db, r.Organization.ID, "Mine", "", "MIN")
	require.NoError(t, err)
	_, err = models.CreateFactory(db, other.ID, "Theirs", "", "THR")
	require.NoError(t, err)

	workspaces, err := ListConsentWorkspaces(ctx, r.Account, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, workspaces, 1)
	assert.Equal(t, mine.ID.String(), workspaces[0].FactoryID)
	assert.Equal(t, "Mine", workspaces[0].Label)

	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	workspaces, err = ListConsentWorkspaces(ctx, r.Account, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, workspaces)
}

func TestNewConsentViewPicksTheStep(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	enableConsentFeatures(t, r.Organization.ID)
	_, err := models.CreateFactory(database.DB(ctx), r.Organization.ID, "Only", "", "ONL")
	require.NoError(t, err)

	view, err := NewConsentView(ctx, r.Account, "")
	require.NoError(t, err)
	require.NotNil(t, view.Organization, "one qualifying organization skips the organization step")
	assert.Equal(t, r.Organization.ID.String(), view.Organization.OrganizationID)
	assert.Len(t, view.Workspaces, 1)
	assert.False(t, view.CanChangeOrganization())

	second := joinedOrganization(t, r)
	enableConsentFeatures(t, second.ID)

	view, err = NewConsentView(ctx, r.Account, "")
	require.NoError(t, err)
	assert.Nil(t, view.Organization)
	assert.Len(t, view.Organizations, 2)
	assert.True(t, view.CanChangeOrganization())

	view, err = NewConsentView(ctx, r.Account, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, view.Organization)
	assert.Equal(t, r.Organization.ID.String(), view.Organization.OrganizationID)
	assert.Len(t, view.Workspaces, 1)

	view, err = NewConsentView(ctx, r.Account, uuid.NewString())
	require.NoError(t, err)
	assert.Nil(t, view.Organization)
}

func TestResolveConsentGrantAcceptsSeveralWorkspacesOfOneOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	enableConsentFeatures(t, r.Organization.ID)

	first, err := models.CreateFactory(db, r.Organization.ID, "First", "", "FIR")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, "Second", "", "SEC")
	require.NoError(t, err)

	userID, orgID, factoryIDs, err := ResolveConsentGrant(
		ctx, r.Account, r.Organization.ID.String(),
		[]string{first.ID.String(), second.ID.String(), first.ID.String()},
	)
	require.NoError(t, err)
	assert.Equal(t, r.User, userID)
	assert.Equal(t, r.Organization.ID, orgID)
	assert.Equal(t, []uuid.UUID{first.ID, second.ID}, factoryIDs)
}

func TestResolveConsentGrantRejectsInvalidChoices(t *testing.T) {
	withGatedMCPServer(t)
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)
	enableConsentFeatures(t, r.Organization.ID)
	mine, err := models.CreateFactory(db, r.Organization.ID, "Mine", "", "MIN")
	require.NoError(t, err)

	other := joinedOrganization(t, r)
	enableConsentFeatures(t, other.ID)
	theirs, err := models.CreateFactory(db, other.ID, "Theirs", "", "THR")
	require.NoError(t, err)

	notMember := support.CreateOrganization(t, r, r.User)
	enableConsentFeatures(t, notMember.ID)
	foreign, err := models.CreateFactory(db, notMember.ID, "Foreign", "", "FOR")
	require.NoError(t, err)

	orgID := r.Organization.ID.String()
	tests := []struct {
		name       string
		orgID      string
		factoryIDs []string
	}{
		{"no workspace", orgID, nil},
		{"malformed workspace", orgID, []string{"not-a-uuid"}},
		{"unknown workspace", orgID, []string{uuid.NewString()}},
		{"workspace of another organization", orgID, []string{mine.ID.String(), theirs.ID.String()}},
		{"organization without membership", notMember.ID.String(), []string{foreign.ID.String()}},
		{"malformed organization", "not-a-uuid", []string{mine.ID.String()}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, _, err := ResolveConsentGrant(ctx, r.Account, test.orgID, test.factoryIDs)
			require.Error(t, err)
		})
	}

	t.Run("a disabled feature flag", func(t *testing.T) {
		for _, feature := range []string{features.FeatureFactories, features.FeatureSuperPlaneMCPServer} {
			require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, feature))
			_, _, _, err := ResolveConsentGrant(ctx, r.Account, orgID, []string{mine.ID.String()})
			require.Error(t, err, feature)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, feature))
		}
	})
}

func TestRenderConsentPageStepsAndTheme(t *testing.T) {
	organizations := []OrganizationOption{{OrganizationID: "org-1", Name: "Acme"}, {OrganizationID: "org-2", Name: "Beta"}}

	page, err := RenderConsentPage(ConsentView{ClientName: "Cursor", ConsentToken: "token", Organizations: organizations})
	require.NoError(t, err)
	html := string(page)
	assert.Contains(t, html, "--foreground: #26251e")
	assert.Contains(t, html, "font-family: \"Inter\"")
	assert.Contains(t, html, "Allow access to SuperPlane")
	assert.Contains(t, html, `name="organization_id"`)
	assert.Contains(t, html, "Acme")
	assert.NotContains(t, html, `name="factory_id"`)

	page, err = RenderConsentPage(ConsentView{
		ClientName:    "Cursor",
		ConsentToken:  "token",
		Organizations: organizations,
		Organization:  &organizations[0],
		Workspaces:    []WorkspaceOption{{FactoryID: "abc", Label: "Alpha"}},
	})
	require.NoError(t, err)
	html = string(page)
	assert.Contains(t, html, `type="checkbox" name="factory_id" value="abc"`)
	assert.Contains(t, html, "Alpha")
	assert.Contains(t, html, "Acme")
	assert.Contains(t, html, "Change organization")

	page, err = RenderConsentPage(ConsentView{
		ClientName:    "Cursor",
		ConsentToken:  "token",
		Organizations: organizations[:1],
		Organization:  &organizations[0],
		Workspaces:    []WorkspaceOption{{FactoryID: "abc", Label: "Alpha"}},
	})
	require.NoError(t, err)
	assert.NotContains(t, string(page), "Change organization")
}
