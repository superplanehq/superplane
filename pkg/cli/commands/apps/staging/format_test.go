package staging

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatStagedPathLineWithoutColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	line := formatStagedPathLine(os.Stdout, "canvas.yaml")
	require.Equal(t, "M canvas.yaml", line)
}

func TestFormatStagedPathLineWithColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")

	var buf bytes.Buffer
	line := formatStagedPathLine(&buf, "canvas.yaml")
	require.Equal(t, "\x1b[32mM\x1b[0m canvas.yaml", line)
}
