package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func Test__NextUniqueInstallationName(t *testing.T) {
	assert.Equal(t, "github-acme", nextUniqueInstallationName("github-acme", nil))
	assert.Equal(t, "github-acme", nextUniqueInstallationName("github-acme", func(string) bool { return false }))

	taken := map[string]bool{"github-acme": true}
	assert.Equal(t, "github-acme (1)", nextUniqueInstallationName("github-acme", func(name string) bool { return taken[name] }))

	taken["github-acme (1)"] = true
	assert.Equal(t, "github-acme (2)", nextUniqueInstallationName("github-acme", func(name string) bool { return taken[name] }))
}

func Test__AssignUniqueInstallationName(t *testing.T) {
	organization, err := CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)

	first, err := CreateIntegration(uuid.New(), organization.ID, "github", "github", nil)
	require.NoError(t, err)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return first.AssignUniqueInstallationName(tx, "github-acme")
	}))
	assert.Equal(t, "github-acme", first.InstallationName)
	require.NoError(t, database.Conn().Save(first).Error)

	second, err := CreateIntegration(uuid.New(), organization.ID, "github", "github-2", nil)
	require.NoError(t, err)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return second.AssignUniqueInstallationName(tx, "github-acme")
	}))
	assert.Equal(t, "github-acme (1)", second.InstallationName)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return first.AssignUniqueInstallationName(tx, "github-acme")
	}))
	assert.Equal(t, "github-acme", first.InstallationName)
}
