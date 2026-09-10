package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__HasActiveCanvasRun__RecognizesUnfinishedStates(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "schedule", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	run, err := models.CreateCanvasRunInTransaction(
		database.DB(t.Context()),
		canvas.ID,
		"schedule",
		models.CanvasRunStatePending,
		"",
	)
	require.NoError(t, err)

	tests := []struct {
		state  string
		active bool
	}{
		{state: models.CanvasRunStatePending, active: true},
		{state: models.CanvasRunStateStarted, active: true},
		{state: models.CanvasRunStateCancelling, active: true},
		{state: models.CanvasRunStateFinished, active: false},
	}

	for _, test := range tests {
		t.Run(test.state, func(t *testing.T) {
			require.NoError(t, database.DB(t.Context()).Model(run).Update("state", test.state).Error)

			active, err := models.HasActiveCanvasRun(database.DB(t.Context()), canvas.ID)
			require.NoError(t, err)
			assert.Equal(t, test.active, active)
		})
	}
}

func Test__HasActiveCanvasRun__ScopesRunsToCanvas(t *testing.T) {
	r := support.Setup(t)
	canvasWithActiveRun, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "schedule", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)
	canvasWithoutRuns, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "schedule", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	_, err := models.CreateCanvasRunInTransaction(
		database.DB(t.Context()),
		canvasWithActiveRun.ID,
		"schedule",
		models.CanvasRunStateStarted,
		"",
	)
	require.NoError(t, err)

	active, err := models.HasActiveCanvasRun(database.DB(t.Context()), canvasWithoutRuns.ID)
	require.NoError(t, err)
	assert.False(t, active)
}

func Test__HasActiveCanvasRun__SerializesConcurrentRunChecks(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "schedule", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	firstReady := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- database.Conn().Transaction(func(tx *gorm.DB) error {
			active, err := models.HasActiveCanvasRun(tx, canvas.ID)
			if err != nil {
				return err
			}
			if active {
				return fmt.Errorf("expected no active run")
			}

			_, err = models.CreateCanvasRunInTransaction(
				tx,
				canvas.ID,
				"schedule",
				models.CanvasRunStateStarted,
				"",
			)
			if err != nil {
				return err
			}

			close(firstReady)
			<-releaseFirst
			return nil
		})
	}()

	select {
	case <-firstReady:
	case err := <-firstDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		close(releaseFirst)
		t.Fatal("timed out waiting for the first run check")
	}

	var concurrentActive bool
	concurrentErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL lock_timeout = '200ms'").Error; err != nil {
			return err
		}

		var err error
		concurrentActive, err = models.HasActiveCanvasRun(tx, canvas.ID)
		return err
	})

	close(releaseFirst)
	require.NoError(t, <-firstDone)

	var postgresErr *pgconn.PgError
	require.ErrorAs(t, concurrentErr, &postgresErr)
	assert.Equal(t, "55P03", postgresErr.Code)
	assert.False(t, concurrentActive)

	active, err := models.HasActiveCanvasRun(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	assert.True(t, active)
}
