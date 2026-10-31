package runner

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalMCPNodeTests(t *testing.T) {
	command := exec.Command("node", "--test", "server_test.js", "owner_test.js", "tasks_test.js", "artifacts_test.js", "flow_tools_test.js", "agent_message_test.js", "stdio_test.js")
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
			if strings.HasPrefix(file.Path, "superplane_mcp/") {
				assert.NotContains(t, file.Path, "task_cli")
			}
			paths[file.Path] = true
		}
		for _, path := range []string{
			"superplane_mcp/server.js",
			"superplane_mcp/modules/owner.js",
			"superplane_mcp/modules/tasks.js",
			"superplane_mcp/modules/artifacts.js",
			"superplane_mcp/modules/plan.js",
			"superplane_mcp/modules/evidence.js",
			"superplane_mcp/modules/links.js",
			"superplane_mcp/modules/merge.js",
			"task_cli.js",
			"bin/superplane",
		} {
			require.True(t, paths[path], path)
		}
		command := files[len(files)-1]
		assert.Equal(t, "bin/superplane", command.Path)
		assert.Equal(t, "0755", command.Mode)
		assert.Len(t, AppendGlobalMCP(env, files), len(files))
	})
}
