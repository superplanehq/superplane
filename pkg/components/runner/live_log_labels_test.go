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
	long := strings.Repeat("a", 100)
	assert.Equal(t, strings.Repeat("a", liveLogPreviewMaxRunes), LiveLogPreview(long))
}
