package agents

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendDraftSnapshotStatus_MarksSnapshotUnavailableWhenDraftLookupFails(t *testing.T) {
	var builder strings.Builder

	source, available := appendDraftSnapshotStatus(&builder, nil, errors.New("database unavailable"))

	assert.False(t, available)
	assert.Empty(t, source)
	assert.Equal(t, "owned_draft: unavailable\nsnapshot_source: unavailable\nnodes: unavailable\n", builder.String())
}
