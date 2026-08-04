package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestAssigneeDiff(t *testing.T) {
	kept := uuid.New()
	added := uuid.New()
	removed := uuid.New()

	assigned, unassigned := assigneeDiff(
		[]uuid.UUID{kept, removed},
		[]uuid.UUID{kept, added},
	)

	requireLen := func(t *testing.T, refs []factory.UserRef, n int) {
		t.Helper()
		assert.Len(t, refs, n)
	}

	requireLen(t, assigned, 1)
	assert.Equal(t, added, assigned[0].ID)
	requireLen(t, unassigned, 1)
	assert.Equal(t, removed, unassigned[0].ID)
}

func TestAssigneeDiffNoChanges(t *testing.T) {
	assignee := uuid.New()

	assigned, unassigned := assigneeDiff([]uuid.UUID{assignee}, []uuid.UUID{assignee})

	assert.Empty(t, assigned)
	assert.Empty(t, unassigned)
}
