package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__IsGeneratedInstallationName(t *testing.T) {
	assert.True(t, IsGeneratedInstallationName("github"))
	assert.True(t, IsGeneratedInstallationName("github-2"))
	assert.True(t, IsGeneratedInstallationName(" github-12 "))
	assert.False(t, IsGeneratedInstallationName("github-acme"))
	assert.False(t, IsGeneratedInstallationName("github-acme (1)"))
	assert.False(t, IsGeneratedInstallationName("My GitHub"))
}

func Test__NextOwnerInstallationName(t *testing.T) {
	assert.Equal(t, "github-acme", NextOwnerInstallationName("Acme", nil))
	assert.Equal(t, "github-acme", NextOwnerInstallationName("Acme", func(string) bool { return false }))

	taken := map[string]bool{"github-acme": true}
	assert.Equal(t, "github-acme (1)", NextOwnerInstallationName("Acme", func(name string) bool { return taken[name] }))

	taken["github-acme (1)"] = true
	assert.Equal(t, "github-acme (2)", NextOwnerInstallationName("Acme", func(name string) bool { return taken[name] }))
}
