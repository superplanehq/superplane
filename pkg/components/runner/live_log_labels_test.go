package runner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveLogTextPreservesInteriorNewlinesAndTrimsOuterBlankLines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "echo \"hello\"\nworld", LiveLogText("\n  echo \"hello\"\nworld\n\n"))
	assert.Equal(t, "", LiveLogText("  \n\t"))

	clone := `git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${REPO}.git"`
	assert.Equal(t, clone, LiveLogText(clone))

	prompt := "You are writing an implementation plan for a SuperPlane Work Order.\n\nThe repository is the working tree."
	assert.Equal(t, prompt, LiveLogText(prompt))

	long := strings.Repeat("a", liveLogTextMaxRunes+10)
	assert.Equal(t, strings.Repeat("a", liveLogTextMaxRunes), LiveLogText(long))
}

func TestLiveLogFirstLineReturnsFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `echo "hello"`, LiveLogFirstLine("\n  echo \"hello\"\nworld"))
	assert.Equal(t, "", LiveLogFirstLine("  \n\t"))

	prompt := "You are writing an implementation plan for a SuperPlane Work Order.\n\nThe repository is the working tree."
	assert.Equal(t, "You are writing an implementation plan for a SuperPlane Work Order.", LiveLogFirstLine(prompt))
}
