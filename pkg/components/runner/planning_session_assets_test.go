package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchedPlanningProtocolIncludesRefinementWithoutSkills(t *testing.T) {
	taskDir := t.TempDir()
	for _, file := range AppendPlanningSessionMCPFiles(nil) {
		require.NoError(t, os.WriteFile(filepath.Join(taskDir, file.Path), []byte(file.Content), 0o644))
	}
	// Read the dispatched assets through the same module as all three runners.
	// No workspace skill is installed in this task directory.
	cmd := exec.Command("node", "-e", `process.stdout.write(require(process.argv[1]).analysisProtocol());`, filepath.Join(taskDir, "analysis_protocol.js"))
	protocol, err := cmd.CombinedOutput()
	require.NoError(t, err, string(protocol))
	guidance, err := os.ReadFile("analysis_user_prompt_review.md")
	require.NoError(t, err)
	assert.Contains(t, string(protocol), strings.TrimSpace(string(guidance)))
	assert.Contains(t, string(protocol), "Before your final response, call propose_plan")
	assert.Contains(t, string(protocol), "no scores have been published")
	assert.Contains(t, string(protocol), "The assign_task_owner call does not complete the turn")
	assert.Contains(t, string(protocol), "On a later answer turn with existing scores and no planning changes, do not call propose_plan")
}
