package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Client__ownerAndName(t *testing.T) {
	client := &Client{owner: "acme"}

	t.Run("short name uses integration owner", func(t *testing.T) {
		owner, name := client.ownerAndName("web")
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "web", name)
	})

	t.Run("owner/repo is split", func(t *testing.T) {
		owner, name := client.ownerAndName("other/api")
		assert.Equal(t, "other", owner)
		assert.Equal(t, "api", name)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		owner, name := client.ownerAndName("  acme/web  ")
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "web", name)
	})

	t.Run("invalid multi-slash falls back to short name", func(t *testing.T) {
		owner, name := client.ownerAndName("a/b/c")
		assert.Equal(t, "acme", owner)
		assert.Equal(t, "a/b/c", name)
	})
}

func Test__repositoryRefersTo(t *testing.T) {
	assert.True(t, repositoryRefersTo("web", "web"))
	assert.True(t, repositoryRefersTo("web", "acme/web"))
	assert.False(t, repositoryRefersTo("web", "acme/api"))
	assert.False(t, repositoryRefersTo("", "web"))
}
