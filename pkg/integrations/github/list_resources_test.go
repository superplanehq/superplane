package github

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__toIntegrationResources__usesFullName(t *testing.T) {
	fullName := "acme/web"
	shortName := "web"
	id := int64(42)

	resources := toIntegrationResources([]*github.Repository{
		{ID: &id, Name: &shortName, FullName: &fullName},
	})

	require.Len(t, resources, 1)
	assert.Equal(t, "repository", resources[0].Type)
	assert.Equal(t, "acme/web", resources[0].Name)
	assert.Equal(t, "42", resources[0].ID)
}

func Test__toIntegrationResources__fallsBackToName(t *testing.T) {
	shortName := "web"
	id := int64(7)

	resources := toIntegrationResources([]*github.Repository{
		{ID: &id, Name: &shortName},
	})

	require.Len(t, resources, 1)
	assert.Equal(t, "web", resources[0].Name)
}
