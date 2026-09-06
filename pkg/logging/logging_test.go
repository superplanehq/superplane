package logging

import (
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

var _ tackle.Logger = TackleLogger{}

func TestWithFunctionsSetFields(t *testing.T) {
	id := uuid.New()
	rootID := uuid.New()
	logger := log.NewEntry(log.New())

	cases := []struct {
		name  string
		entry *log.Entry
		want  log.Fields
	}{
		{
			name:  "factory",
			entry: WithFactory(logger, models.Factory{ID: id}),
			want:  log.Fields{"factory_id": id},
		},
		{
			name:  "work order",
			entry: WithWorkOrder(logger, models.FactoryWorkOrder{ID: id}),
			want:  log.Fields{"order_id": id},
		},
		{
			name: "event",
			entry: ForEvent(logger, models.CanvasEvent{
				ID:      id,
				NodeID:  "node-1",
				Channel: "default",
			}),
			want: log.Fields{"event_id": id, "node_id": "node-1", "channel": "default"},
		},
		{
			name:  "execution",
			entry: WithExecution(logger, &models.CanvasNodeExecution{ID: id, RootEventID: rootID}),
			want:  log.Fields{"root_event": rootID, "execution": id},
		},
		{
			name:  "node",
			entry: WithNode(logger, models.CanvasNode{NodeID: "node-1"}),
			want:  log.Fields{"node_id": "node-1"},
		},
		{
			name:  "queue item",
			entry: WithQueueItem(logger, models.CanvasNodeQueueItem{ID: id, RootEventID: rootID}),
			want:  log.Fields{"queue_item_id": id, "root_event": rootID},
		},
		{
			name:  "integration",
			entry: WithIntegration(logger, models.Integration{ID: id, AppName: "slack"}),
			want:  log.Fields{"integration_name": "slack", "integration_id": id},
		},
		{
			name:  "webhook",
			entry: WithWebhook(logger, models.Webhook{ID: id}),
			want:  log.Fields{"webhook_id": id},
		},
		{
			name:  "run",
			entry: WithRun(logger, models.CanvasRun{ID: id, WorkflowID: rootID}),
			want:  log.Fields{"run_id": id, "workflow_id": rootID},
		},
		{
			name:  "canvas",
			entry: WithCanvas(logger, models.Canvas{ID: id}),
			want:  log.Fields{"canvas_id": id},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, c.entry.Data)
		})
	}
}

func TestForWrappersUseStandardLogger(t *testing.T) {
	id := uuid.New()

	factoryEntry := ForFactory(models.Factory{ID: id})
	require.Equal(t, log.Fields{"factory_id": id}, factoryEntry.Data)

	runID := uuid.New()
	runEntry := ForRun(models.CanvasRun{ID: runID, WorkflowID: id})
	require.Equal(t, log.Fields{"run_id": runID, "workflow_id": id}, runEntry.Data)
}

func TestTackleLoggerLevels(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	l := NewTackleLogger(logger.WithField("k", "v"))

	l.Errorf("boom %d", 42)
	l.Infof("ok %s", "yes")

	entries := hook.AllEntries()
	require.Len(t, entries, 2)
	require.Equal(t, log.ErrorLevel, entries[0].Level)
	require.Equal(t, "boom 42", entries[0].Message)
	require.Equal(t, "v", entries[0].Data["k"])
	require.Equal(t, log.InfoLevel, entries[1].Level)
	require.Equal(t, "ok yes", entries[1].Message)
}
