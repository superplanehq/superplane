package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CanvasDismissAgentSuggestion(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	locked, err := models.LockCanvasForUpdate(database.Conn(), r.Organization.ID, canvas.ID)
	require.NoError(t, err)

	err = locked.DismissAgentSuggestion(database.Conn(), "add-ci")
	require.NoError(t, err)
	assert.Equal(t, []string{"add-ci"}, []string(locked.DismissedAgentSuggestionIDs))

	reloaded, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	t.Logf("reloaded dismissals=%#v", reloaded.DismissedAgentSuggestionIDs)
	assert.Equal(t, []string{"add-ci"}, []string(reloaded.DismissedAgentSuggestionIDs))
}
