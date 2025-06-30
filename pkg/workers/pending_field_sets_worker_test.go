package workers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__PendingFieldSetsWorkerTest(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	connectionGroup := support.CreateConnectionGroup(t,
		"connection-group-drop",
		r.Canvas,
		r.Source,
		models.MaxConnectionGroupTimeout,
		models.ConnectionGroupTimeoutBehaviorDrop,
	)

	t.Run("field set not timed out -> does nothing", func(t *testing.T) {
		w, _ := NewPendingFieldSetsWorker(
			func() time.Time {
				return time.Now()
			},
		)

		// Create field set
		fields := map[string]string{"version": "v1"}
		fieldSet := support.CreateFieldSet(t, fields, connectionGroup, r.Source)

		// Trigger the worker and verify field sets remains in pending state
		require.NoError(t, w.Tick())
		fieldSet, err := connectionGroup.FindFieldSetByID(fieldSet.ID)
		require.NoError(t, err)
		require.Equal(t, models.ConnectionGroupFieldSetStatePending, fieldSet.State)
	})

	t.Run("field set is timed out -> drop", func(t *testing.T) {
		w, _ := NewPendingFieldSetsWorker(
			func() time.Time {
				return time.Now().Add(25 * time.Hour)
			},
		)

		// Create field set
		fields := map[string]string{"version": "v1"}
		fieldSet := support.CreateFieldSet(t, fields, connectionGroup, r.Source)

		// Trigger the worker and verify field sets is discarded as timed-out
		require.NoError(t, w.Tick())
		fieldSet, err := connectionGroup.FindFieldSetByID(fieldSet.ID)
		require.NoError(t, err)
		require.Equal(t, models.ConnectionGroupFieldSetStateDiscarded, fieldSet.State)
		require.Equal(t, models.ConnectionGroupFieldSetResultTimedOut, fieldSet.Result)
	})

	t.Run("field set is timed out, emit -> emit as timed-out", func(t *testing.T) {
		connectionGroup := support.CreateConnectionGroup(t,
			"connection-group-emit",
			r.Canvas,
			r.Source,
			models.MaxConnectionGroupTimeout,
			models.ConnectionGroupTimeoutBehaviorEmit,
		)

		w, _ := NewPendingFieldSetsWorker(
			func() time.Time {
				return time.Now().Add(25 * time.Hour)
			},
		)

		// Create field set
		fields := map[string]string{"version": "v1"}
		fieldSet := support.CreateFieldSet(t, fields, connectionGroup, r.Source)

		// Trigger the worker and verify field sets remains is processed as timed out
		require.NoError(t, w.Tick())
		fieldSet, err := connectionGroup.FindFieldSetByID(fieldSet.ID)
		require.NoError(t, err)
		require.Equal(t, models.ConnectionGroupFieldSetStateProcessed, fieldSet.State)
		require.Equal(t, models.ConnectionGroupFieldSetResultTimedOut, fieldSet.Result)

		// Verify that a new event was emitted
		events, err := models.ListEventsBySourceID(connectionGroup.ID)
		require.NoError(t, err)
		require.Len(t, events, 1)
		event := events[0]
		assert.Equal(t, connectionGroup.ID, event.SourceID)
		assert.Equal(t, connectionGroup.Name, event.SourceName)
		assert.Equal(t, models.SourceTypeConnectionGroup, event.SourceType)
		var eventData map[string]any
		require.NoError(t, json.Unmarshal(event.Raw, &eventData))
		assert.Equal(t, map[string]any{
			"fields": map[string]any{"version": "v1"},
			"events": map[string]any{"gh": map[string]any{}},
			"result": models.ConnectionGroupFieldSetResultTimedOut,
		}, eventData)
	})
}
