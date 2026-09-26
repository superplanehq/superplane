package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendInstructions(t *testing.T) {
	t.Run("adds the block at the end", func(t *testing.T) {
		got := AppendInstructions("Fix the login bug.\n", "Keep the change small.")
		assert.Equal(t, "Fix the login bug.\n\n## Instructions\nKeep the change small.", got)
	})

	t.Run("an empty description holds only the block", func(t *testing.T) {
		assert.Equal(t, "## Instructions\nKeep the change small.", AppendInstructions("", "Keep the change small."))
	})

	t.Run("empty instructions leave the description alone", func(t *testing.T) {
		assert.Equal(t, "Fix the login bug.", AppendInstructions("Fix the login bug.", "  \n"))
	})
}

func TestInsertBeforeInstructions(t *testing.T) {
	t.Run("keeps the block above the instructions", func(t *testing.T) {
		description := "Fix the login bug.\n\n## Instructions\nKeep the change small."
		got := InsertBeforeInstructions(description, "### Also\nThe logout button.")
		assert.Equal(t, "Fix the login bug.\n\n### Also\nThe logout button.\n\n## Instructions\nKeep the change small.", got)
	})

	t.Run("appends when there are no instructions", func(t *testing.T) {
		assert.Equal(t, "Fix the login bug.\n\n### Also", InsertBeforeInstructions("Fix the login bug.\n", "### Also"))
	})

	t.Run("ignores an empty block", func(t *testing.T) {
		assert.Equal(t, "Fix the login bug.", InsertBeforeInstructions("Fix the login bug.", ""))
	})
}
