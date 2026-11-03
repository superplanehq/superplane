package workers

import (
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers/messages"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// Persist each completion event in the transaction that finishes its source run.
// EventRouter also polls pending events, so delivery survives a lost notification.
func (d *RunCallbackDispatcher) dispatchRunFinishedTriggers() error {
	if d.run.State != models.CanvasRunStateFinished {
		return nil
	}
	nodes, err := models.FindCanvasNodesInTransaction(d.tx, d.run.WorkflowID)
	if err != nil {
		return err
	}
	for i := range nodes {
		node := &nodes[i]
		ref := node.Ref.Data().Trigger
		if ref == nil || ref.Name != messages.OnRunFinishedTriggerName || node.NodeID == d.run.NodeID {
			continue
		}
		var config messages.OnRunFinishedConfiguration
		if err := mapstructure.Decode(node.Configuration.Data(), &config); err != nil {
			return err
		}
		entryNodeID := strings.TrimSpace(config.SourceEntryNodeID)
		if entryNodeID == "" || entryNodeID != d.run.NodeID {
			continue
		}
		payload := map[string]any{"runId": d.run.ID.String(), "result": d.run.Result, "entryNodeId": d.run.NodeID}
		if err := contexts.NewEventContext(d.tx, node, nil, d.eventCollector).
			EmitDeduplicated(messages.RunFinishedPayloadType, payload, "run-finished:"+d.run.ID.String()); err != nil {
			return err
		}
	}
	return nil
}
