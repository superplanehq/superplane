package runner

import (
	"os"
	"os/exec"
	"path/filepath"
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
	assert.Contains(t, string(protocol), "Ask only when two valid readings materially change the outcome")
	assert.Contains(t, string(protocol), "'## Scope', '## Approach', '## Acceptance', and '## Risks'")
	assert.Contains(t, string(protocol), "integer from 1 through 3")
	assert.Contains(t, string(protocol), "propose_update")
}
