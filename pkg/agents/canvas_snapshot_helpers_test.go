package agents

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAppendDraftSnapshotStatus_MarksSnapshotUnavailableWhenDraftLookupFails(t *testing.T) {
	var builder strings.Builder

	source, available := appendDraftSnapshotStatus(&builder, nil, errors.New("database unavailable"))

	assert.False(t, available)
	assert.Empty(t, source)
	assert.Equal(t, "owned_draft: unavailable\nsnapshot_source: unavailable\nnodes: unavailable\n", builder.String())
}

func TestSelectedVersion_ReturnsLiveVersionLoadErrors(t *testing.T) {
	missingVersionID := uuid.New()

	version, err := selectedVersion(&models.Canvas{
		ID:            uuid.New(),
		LiveVersionID: &missingVersionID,
	}, nil, "live")

	require.Error(t, err)
	assert.Nil(t, version)
}
