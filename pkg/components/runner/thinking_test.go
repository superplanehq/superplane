package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeThinkingLevel(t *testing.T) {
	t.Parallel()

	got, err := NormalizeThinkingLevel(" High ")
	require.NoError(t, err)
	assert.Equal(t, ThinkingLevelHigh, got)

	got, err = NormalizeThinkingLevel("")
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = NormalizeThinkingLevel("xhigh")
	require.Error(t, err)
}

func TestOverlayThinkingLevel(t *testing.T) {
	t.Parallel()

	got, ok := OverlayThinkingLevel("high", "")
	assert.False(t, ok)
	assert.Equal(t, "high", got)

	got, ok = OverlayThinkingLevel("high", "default")
	assert.True(t, ok)
	assert.Empty(t, got)

	got, ok = OverlayThinkingLevel("", "medium")
	assert.True(t, ok)
	assert.Equal(t, ThinkingLevelMedium, got)
}

func TestPromptNodeCommandOmitsEmptyThinking(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-prompt.txt" 'sonnet'`,
		PromptNodeCommand("01-prompt.txt", "sonnet", ""),
	)
	assert.Equal(
		t,
		`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-prompt.txt" 'sonnet' 'high'`,
		PromptNodeCommand("01-prompt.txt", "sonnet", "high"),
	)
}
