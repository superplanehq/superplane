package changesets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RerunNodeSetup(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		componentCanvasNode("recovered", "Recovered", "noop", map[string]any{}),
		componentCanvasNode("broken", "Broken", "missingcomponent", map[string]any{}),
		componentCanvasNode("untouched", "Untouched", "missingcomponent", map[string]any{}),
	}, nil)
	staleReason := "setup failed earlier"
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "recovered").
		Updates(map[string]any{"state": models.CanvasNodeStateError, "state_reason": staleReason}).Error)

	require.NoError(t, RerunNodeSetup(t.Context(), db, canvas, []string{"recovered", "broken"}, canvasPublisherOptions(r)))

	recovered, err := models.FindCanvasNode(db, canvas.ID, "recovered")
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, recovered.State)
	assert.Nil(t, recovered.StateReason)

	broken, err := models.FindCanvasNode(db, canvas.ID, "broken")
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateError, broken.State)
	require.NotNil(t, broken.StateReason)
	assert.Contains(t, *broken.StateReason, "missingcomponent not registered")

	untouched, err := models.FindCanvasNode(db, canvas.ID, "untouched")
	require.NoError(t, err)
	assert.NotEqual(t, models.CanvasNodeStateError, untouched.State, "only the requested nodes run Setup again")
}

func Test__CanvasPublisherFactoryContextOnlyForFactoryApps(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	assert.Nil(t, (&CanvasPublisher{tx: db, canvas: canvas}).factoryContext())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas.FactoryID = &factory.ID
	assert.NotNil(t, (&CanvasPublisher{tx: db, canvas: canvas}).factoryContext())
}

func Test__RerunNodeSetupRejectsInvalidOptions(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		componentCanvasNode("node-a", "Node A", "noop", map[string]any{}),
	}, nil)
	options := canvasPublisherOptions(r)
	options.Registry = nil

	assert.Error(t, RerunNodeSetup(t.Context(), database.Conn(), canvas, []string{"node-a"}, options))
}
