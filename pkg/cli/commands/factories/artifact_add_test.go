package factories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestArtifactAddCommand_BuildData(t *testing.T) {
	t.Run("markdown from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "PLAN.md")
		require.NoError(t, os.WriteFile(path, []byte("# Plan"), 0o600))

		title := "PLAN.md"
		file := path
		cmd := &cobra.Command{}
		c := &artifactAddCommand{title: &title, file: &file}
		data, typ, err := c.buildData(core.CommandContext{Cmd: cmd}, "markdown")
		require.NoError(t, err)
		assert.Equal(t, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_MARKDOWN, typ)
		assert.Equal(t, "# Plan", data["body"])
		assert.Equal(t, "PLAN.md", data["title"])
	})

	t.Run("markdown requires body or file", func(t *testing.T) {
		cmd := &cobra.Command{}
		c := &artifactAddCommand{}
		_, _, err := c.buildData(core.CommandContext{Cmd: cmd}, "markdown")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--body or --file")
	})

	t.Run("unknown type includes pr", func(t *testing.T) {
		cmd := &cobra.Command{}
		c := &artifactAddCommand{}
		_, _, err := c.buildData(core.CommandContext{Cmd: cmd}, "pr")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown artifact type")
	})

	t.Run("branch requires name", func(t *testing.T) {
		cmd := &cobra.Command{}
		c := &artifactAddCommand{}
		_, _, err := c.buildData(core.CommandContext{Cmd: cmd}, "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--name")
	})

	t.Run("link requires url", func(t *testing.T) {
		cmd := &cobra.Command{}
		c := &artifactAddCommand{}
		_, _, err := c.buildData(core.CommandContext{Cmd: cmd}, "link")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--url")
	})

	t.Run("link builds url and title", func(t *testing.T) {
		url := "https://preview.example.com/pr-42"
		title := "Preview"
		cmd := &cobra.Command{}
		c := &artifactAddCommand{url: &url, title: &title}
		data, typ, err := c.buildData(core.CommandContext{Cmd: cmd}, "link")
		require.NoError(t, err)
		assert.Equal(t, openapi_client.FACTORIESWORKORDERARTIFACTTYPE_TYPE_LINK, typ)
		assert.Equal(t, url, data["url"])
		assert.Equal(t, title, data["title"])
	})

	t.Run("unknown type", func(t *testing.T) {
		cmd := &cobra.Command{}
		c := &artifactAddCommand{}
		_, _, err := c.buildData(core.CommandContext{Cmd: cmd}, "screenshot")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown artifact type")
	})
}
