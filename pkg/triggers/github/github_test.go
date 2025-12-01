package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBranchDeletionEvent(t *testing.T) {
	assert.True(t, isBranchDeletionEvent("push", map[string]any{"deleted": true}))
	assert.False(t, isBranchDeletionEvent("push", map[string]any{"deleted": false}))
	assert.False(t, isBranchDeletionEvent("push", map[string]any{}))
	assert.False(t, isBranchDeletionEvent("pull_request", map[string]any{}))
}
