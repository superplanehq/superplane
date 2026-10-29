package runner

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalMCPNodeTests(t *testing.T) {
	command := exec.Command("node", "--test", "server_test.js", "owner_test.js", "stdio_test.js")
	command.Dir = "superplane_mcp"
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("global runner MCP tests failed: %v\n%s", err, output)
	}
}

func TestAppendGlobalMCP(t *testing.T) {
	t.Run("skips runs without a runner token", func(t *testing.T) {
		assert.Empty(t, AppendGlobalMCP(nil, nil))
	})

	t.Run("ships the server without tests", func(t *testing.T) {
		env := []TaskEnvironmentVariable{{Name: EnvSuperplaneRunnerToken, Value: "token"}}
		files := AppendGlobalMCP(env, nil)

		paths := map[string]bool{}
		for _, file := range files {
			assert.False(t, strings.HasSuffix(file.Path, "_test.js"), file.Path)
			assert.False(t, strings.HasPrefix(file.Path, "task_cli"), file.Path)
			assert.NotEqual(t, "bin/superplane", file.Path)
			paths[file.Path] = true
		}
		for _, path := range []string{
			"superplane_mcp/server.js", "superplane_mcp/modules/owner.js",
		} {
			require.True(t, paths[path], path)
		}
		assert.Len(t, AppendGlobalMCP(env, files), len(files))
	})
}
