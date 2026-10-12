package runner

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeConfidenceMCPScript(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not available")
	}

	cmd := exec.Command("node", "--test", "merge_confidence_mcp_test.js")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
