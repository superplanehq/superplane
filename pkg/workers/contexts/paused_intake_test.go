package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SkipPausedIntakeFeed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	intake, err := factory.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceJiraIssues)
	require.NoError(t, err)

	skip, err := SkipPausedIntakeFeed(database.Conn(), otherCanvas.ID)
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = SkipPausedIntakeFeed(database.Conn(), uuid.New())
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = SkipPausedIntakeFeed(database.Conn(), canvas.ID)
	require.NoError(t, err)
	assert.False(t, skip)

	require.NoError(t, intake.SetPaused(database.Conn(), true))
	skip, err = SkipPausedIntakeFeed(database.Conn(), canvas.ID)
	require.NoError(t, err)
	assert.True(t, skip)

	require.NoError(t, intake.SetPaused(database.Conn(), false))
	skip, err = SkipPausedIntakeFeed(database.Conn(), canvas.ID)
	require.NoError(t, err)
	assert.False(t, skip)
}
