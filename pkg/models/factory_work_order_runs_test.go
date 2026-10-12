package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryWorkOrder__Close__LeavesSharedCreatorRunning(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "intake", "start")
	source, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)

	closing, err := factory.CreateWorkOrder(db, "Closing", "", &r.User, nil, &source.ID)
	require.NoError(t, err)
	_, err = factory.CreateWorkOrder(db, "Other", "", &r.User, nil, &source.ID)
	require.NoError(t, err)

	otherWork, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	ownedWork, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(otherWork).Update("parent_run_id", source.ID).Error)
	require.NoError(t, db.Model(ownedWork).Updates(map[string]any{
		"parent_run_id": source.ID,
		"input": models.NewJSONValue(map[string]any{
			"work_order": map[string]any{"id": closing.ID.String()},
		}),
	}).Error)

	_, err = closing.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	source, err = models.FindUnscopedCanvasRun(db, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, source.State)
	otherWork, err = models.FindUnscopedCanvasRun(db, otherWork.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, otherWork.State)
	ownedWork, err = models.FindUnscopedCanvasRun(db, ownedWork.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, ownedWork.State)
}

func Test__FactoryWorkOrder__Close__CancelsExclusiveCreator(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "intake", "start")
	source, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Only", "", &r.User, nil, &source.ID)
	require.NoError(t, err)

	_, err = order.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
	require.NoError(t, err)

	source, err = models.FindUnscopedCanvasRun(db, source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, source.State)
}
