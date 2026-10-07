package runner

import (
	"os/exec"
	"testing"
)

func TestTaskArtifactMCPNodeTests(t *testing.T) {
	command := exec.Command("node", "--test", "task_artifact_mcp_test.js")
	command.Dir = "."
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("task artifact MCP tests failed: %v\n%s", err, output)
	}
}
