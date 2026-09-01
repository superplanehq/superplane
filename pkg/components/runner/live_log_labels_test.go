package runner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveLogPreviewUsesFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `echo "hello"`, LiveLogPreview("\n  echo \"hello\"\nworld"))
	assert.Equal(t, "", LiveLogPreview("  \n\t"))

	clone := `git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${REPO}.git"`
	assert.Equal(t, clone, LiveLogPreview(clone))

	prompt := "You are writing an implementation plan for a SuperPlane Work Order. The repository is the working tree."
	assert.Equal(t, prompt, LiveLogPreview(prompt))

	long := strings.Repeat("a", liveLogPreviewMaxRunes+10)
	assert.Equal(t, strings.Repeat("a", liveLogPreviewMaxRunes), LiveLogPreview(long))
}
