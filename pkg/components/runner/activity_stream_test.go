package runner

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivityStreamScript(t *testing.T) {
	cmd := exec.Command("node", "--test", "activity_stream_test.js")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
