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
