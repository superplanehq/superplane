package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func Test__IsGeneratedInstallationName(t *testing.T) {
	assert.True(t, IsGeneratedInstallationName("github"))
	assert.True(t, IsGeneratedInstallationName("github-2"))
	assert.True(t, IsGeneratedInstallationName(" github-12 "))
	assert.False(t, IsGeneratedInstallationName("github-acme"))
	assert.False(t, IsGeneratedInstallationName("github-acme (1)"))
	assert.False(t, IsGeneratedInstallationName("My GitHub"))
}

func Test__OwnerInstallationName(t *testing.T) {
	assert.Equal(t, "github-acme", OwnerInstallationName("Acme"))
	assert.Equal(t, "github-acme", OwnerInstallationName(" acme "))
}

func Test__GeneratedOwnerInstallationName(t *testing.T) {
	name, ok := GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "github",
		InstallationName: "github",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Acme"}),
	})
	assert.True(t, ok)
	assert.Equal(t, "github-acme", name)

	_, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "github",
		InstallationName: "github-acme",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Acme"}),
	})
	assert.False(t, ok)

	// A rebind to another account regenerates a name from the new owner.
	name, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "github",
		InstallationName: "github-acme",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Octo"}),
	})
	assert.True(t, ok)
	assert.Equal(t, "github-octo", name)

	// A uniqueness suffix for the same owner stays untouched.
	_, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "github",
		InstallationName: "github-acme (2)",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Acme"}),
	})
	assert.False(t, ok)

	// A name the user typed does not regenerate.
	_, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "github",
		InstallationName: "My GitHub",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Octo"}),
	})
	assert.False(t, ok)

	_, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "slack",
		InstallationName: "github",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Acme"}),
	})
	assert.False(t, ok)
}
