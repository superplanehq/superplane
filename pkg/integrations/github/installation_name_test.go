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

	_, ok = GeneratedOwnerInstallationName(&models.Integration{
		AppName:          "slack",
		InstallationName: "github",
		Metadata:         datatypes.NewJSONType(map[string]any{"owner": "Acme"}),
	})
	assert.False(t, ok)
}
