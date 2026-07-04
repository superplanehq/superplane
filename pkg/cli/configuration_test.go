package cli

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func setupViper(t *testing.T) {
	t.Helper()

	viper.Reset()

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	require.NoError(t, viper.WriteConfigAs(path))
	require.NoError(t, viper.ReadInConfig())

	t.Cleanup(viper.Reset)
}

func TestContextSelectorPrefersOrganizationID(t *testing.T) {
	withID := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Nandor Lite Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	require.Equal(t,
		"https://app.superplane.com/eb3aca82-3864-4f76-b1d6-f670c297f136",
		ContextSelector(withID),
	)

	legacy := ConfigContext{
		URL:          "https://app.superplane.com",
		Organization: "Nandor Lite Playground",
		APIToken:     "tok",
	}
	require.Equal(t,
		"https://app.superplane.com/Nandor Lite Playground",
		ContextSelector(legacy),
	)
}

func TestSwitchContextMatchesByNameAndID(t *testing.T) {
	setupViper(t)

	ctx := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Nandor Lite Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{ctx}))

	byName, err := SwitchContext("https://app.superplane.com", "Nandor Lite Playground")
	require.NoError(t, err)
	require.Equal(t, ctx.OrganizationID, byName.OrganizationID)

	byID, err := SwitchContext("https://app.superplane.com", "eb3aca82-3864-4f76-b1d6-f670c297f136")
	require.NoError(t, err)
	require.Equal(t, ctx.OrganizationID, byID.OrganizationID)

	withTrailingSlash, err := SwitchContext("https://app.superplane.com/", "Nandor Lite Playground")
	require.NoError(t, err)
	require.Equal(t, ctx.OrganizationID, withTrailingSlash.OrganizationID)
}

func TestSwitchContextErrorsOnUnknownAndEmpty(t *testing.T) {
	setupViper(t)

	ctx := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Nandor Lite Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{ctx}))

	_, err := SwitchContext("https://app.superplane.com", "Does Not Exist")
	require.Error(t, err)

	_, err = SwitchContext("https://wrong.example.com", "Nandor Lite Playground")
	require.Error(t, err)

	_, err = SwitchContext("", "Nandor Lite Playground")
	require.Error(t, err)
	_, err = SwitchContext("https://app.superplane.com", "")
	require.Error(t, err)
}

func TestSwitchContextDisambiguatesBySameNameDifferentURL(t *testing.T) {
	setupViper(t)

	prod := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Shared Name",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok-prod",
	}
	staging := ConfigContext{
		URL:            "https://staging.superplane.com",
		Organization:   "Shared Name",
		OrganizationID: "bbbbbbbb-0000-0000-0000-000000000002",
		APIToken:       "tok-staging",
	}
	require.NoError(t, SaveContexts([]ConfigContext{prod, staging}))

	got, err := SwitchContext("https://staging.superplane.com", "Shared Name")
	require.NoError(t, err)
	require.Equal(t, staging.OrganizationID, got.OrganizationID)

	got, err = SwitchContext("https://app.superplane.com", "Shared Name")
	require.NoError(t, err)
	require.Equal(t, prod.OrganizationID, got.OrganizationID)
}

func TestUpsertContextUpgradesLegacyEntryInPlace(t *testing.T) {
	setupViper(t)

	legacy := ConfigContext{
		URL:          "https://app.superplane.com",
		Organization: "Nandor Lite Playground",
		APIToken:     "tok-legacy",
	}
	require.NoError(t, SaveContexts([]ConfigContext{legacy}))

	upgraded := legacy
	upgraded.OrganizationID = "eb3aca82-3864-4f76-b1d6-f670c297f136"
	upgraded.APIToken = "tok-new"

	_, err := UpsertContext(upgraded)
	require.NoError(t, err)

	all := GetContexts()
	require.Len(t, all, 1)
	require.Equal(t, "eb3aca82-3864-4f76-b1d6-f670c297f136", all[0].OrganizationID)
	require.Equal(t, "tok-new", all[0].APIToken)
}

func TestSwitchContextByOrganizationUniqueByID(t *testing.T) {
	setupViper(t)

	ctx := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{ctx}))

	got, err := SwitchContextByOrganization("eb3aca82-3864-4f76-b1d6-f670c297f136", "")
	require.NoError(t, err)
	require.Equal(t, ctx.OrganizationID, got.OrganizationID)

	current, ok := GetCurrentContext()
	require.True(t, ok)
	require.Equal(t, ctx.OrganizationID, current.OrganizationID)
}

func TestSwitchContextByOrganizationUniqueByName(t *testing.T) {
	setupViper(t)

	ctx := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{ctx}))

	got, err := SwitchContextByOrganization("Playground", "")
	require.NoError(t, err)
	require.Equal(t, ctx.OrganizationID, got.OrganizationID)
}

func TestSwitchContextByOrganizationAmbiguousWithoutURL(t *testing.T) {
	setupViper(t)

	a := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Shared",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok-a",
	}
	b := ConfigContext{
		URL:            "https://staging.example.com",
		Organization:   "Shared",
		OrganizationID: "bbbbbbbb-0000-0000-0000-000000000002",
		APIToken:       "tok-b",
	}
	require.NoError(t, SaveContexts([]ConfigContext{a, b}))

	_, err := SwitchContextByOrganization("Shared", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous organization")
	require.Contains(t, err.Error(), "https://app.superplane.com")
	require.Contains(t, err.Error(), "https://staging.example.com")
}

func TestSwitchContextByOrganizationResolvesWithURLFlag(t *testing.T) {
	setupViper(t)

	a := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Shared",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok-a",
	}
	b := ConfigContext{
		URL:            "https://staging.example.com",
		Organization:   "Shared",
		OrganizationID: "bbbbbbbb-0000-0000-0000-000000000002",
		APIToken:       "tok-b",
	}
	require.NoError(t, SaveContexts([]ConfigContext{a, b}))

	got, err := SwitchContextByOrganization("Shared", "https://staging.example.com")
	require.NoError(t, err)
	require.Equal(t, b.OrganizationID, got.OrganizationID)
}

func TestSwitchContextByOrganizationDuplicateSameURLError(t *testing.T) {
	setupViper(t)

	a := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Dup",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok-a",
	}
	b := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Dup",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok-b",
	}
	require.NoError(t, SaveContexts([]ConfigContext{a, b}))

	_, err := SwitchContextByOrganization("Dup", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple saved contexts")
}

func TestSwitchContextByOrganizationUnknown(t *testing.T) {
	setupViper(t)

	ctx := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Only",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{ctx}))

	_, err := SwitchContextByOrganization("nope", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no context found")
}

func TestGetCurrentContextResolvesAfterUpgrade(t *testing.T) {
	setupViper(t)

	legacy := ConfigContext{
		URL:          "https://app.superplane.com",
		Organization: "Nandor Lite Playground",
		APIToken:     "tok",
	}
	require.NoError(t, SaveContexts([]ConfigContext{legacy}))
	viper.Set(ConfigKeyCurrentContext, ContextSelector(legacy))
	require.NoError(t, WriteConfig())

	upgraded := legacy
	upgraded.OrganizationID = "eb3aca82-3864-4f76-b1d6-f670c297f136"
	_, err := UpsertContext(upgraded)
	require.NoError(t, err)

	current, ok := GetCurrentContext()
	require.True(t, ok)
	require.Equal(t, upgraded.OrganizationID, current.OrganizationID)
}

func TestEnvironmentContextOverridesConfiguredContext(t *testing.T) {
	setupViper(t)

	_, err := UpsertContext(ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Saved",
		OrganizationID: "aaaaaaaa-0000-0000-0000-000000000001",
		APIToken:       "saved-token",
	})
	require.NoError(t, err)

	t.Setenv(EnvURL, "https://agent.superplane.com/")
	t.Setenv(EnvToken, "agent-token")

	require.NoError(t, ValidateEnvironmentContext())
	require.Equal(t, "https://agent.superplane.com", GetAPIURL())
	require.Equal(t, "agent-token", GetAPIToken())

	context, ok := GetEnvironmentContext()
	require.True(t, ok)
	require.Equal(t, "https://agent.superplane.com", context.URL)
	require.Equal(t, "agent-token", context.APIToken)

	config := NewEnvironmentContext(context)
	err = config.SetActiveApp("canvas-from-env-run")
	require.Error(t, err)
	require.Contains(t, err.Error(), "pass --app-id")
	require.Empty(t, config.GetActiveApp())

	current, ok := GetCurrentContext()
	require.True(t, ok)
	require.Nil(t, current.Canvas)
	require.Equal(t, "saved-token", current.APIToken)
}

func TestEnvironmentContextRequiresURLAndToken(t *testing.T) {
	setupViper(t)

	t.Setenv(EnvURL, "https://agent.superplane.com")

	err := ValidateEnvironmentContext()
	require.Error(t, err)
	require.Contains(t, err.Error(), EnvURL)
	require.Contains(t, err.Error(), EnvToken)
}
