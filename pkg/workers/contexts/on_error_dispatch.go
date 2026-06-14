package contexts

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers/onerror"
	"gorm.io/gorm"
)

// DispatchOnError fans an errored execution out to every On Error trigger node
// in the same canvas by emitting a root event on each of them. The emitted
// events are collected through onNewEvents, so the caller publishes them after
// its transaction commits (and the EventRouter starts a new run for each).
//
// On Error nodes are not wired through edges, so this is what connects a failed
// execution to them. The event payload describes which node errored and carries
// the payloads emitted by every upstream node in the failed run.
//
// Dispatch failures must never roll back or mask the original execution
// failure, so any error here is logged and swallowed.
func DispatchOnError(tx *gorm.DB, execution *models.CanvasNodeExecution, onNewEvents func([]models.CanvasEvent)) {
	if err := dispatchOnError(tx, execution, onNewEvents); err != nil {
		log.WithError(err).Warnf("failed to dispatch onError event for execution %s", execution.ID)
	}
}

func dispatchOnError(tx *gorm.DB, execution *models.CanvasNodeExecution, onNewEvents func([]models.CanvasEvent)) error {
	nodes, err := models.FindCanvasNodesInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return err
	}

	onErrorNodes := onErrorNodesFrom(nodes)
	if len(onErrorNodes) == 0 {
		return nil
	}

	//
	// Loop prevention: do not react to errors that happen inside a run that was
	// itself started by an On Error node.
	//
	rootEvent, err := models.FindCanvasEventInTransaction(tx, execution.RootEventID)
	if err != nil {
		return err
	}

	for i := range onErrorNodes {
		if onErrorNodes[i].NodeID == rootEvent.NodeID {
			return nil
		}
	}

	payload, err := buildOnErrorPayload(tx, execution, nodes, rootEvent)
	if err != nil {
		return err
	}

	for i := range onErrorNodes {
		eventCtx := NewEventContext(tx, &onErrorNodes[i], onNewEvents)
		if err := eventCtx.Emit(onerror.PayloadType, payload); err != nil {
			return err
		}
	}

	return nil
}

func buildOnErrorPayload(tx *gorm.DB, execution *models.CanvasNodeExecution, nodes []models.CanvasNode, rootEvent *models.CanvasEvent) (map[string]any, error) {
	erroredNode := findNode(nodes, execution.NodeID)

	builder := NewNodeConfigurationBuilder(tx, execution.WorkflowID).
		WithNodeID(execution.NodeID).
		WithRootEvent(&execution.RootEventID).
		WithPreviousExecution(&execution.ID)

	payloads, err := builder.BuildExecutionMessageChain()
	if err != nil {
		//
		// The error itself matters more than the upstream payloads. If the
		// message chain can't be resolved, still deliver the error with an
		// empty payloads map.
		//
		log.WithError(err).Warnf("failed to build message chain for execution %s", execution.ID)
		payloads = map[string]any{}
	}

	nodeInfo := map[string]any{
		"id":        execution.NodeID,
		"component": componentNameForNode(erroredNode),
	}
	if erroredNode != nil {
		nodeInfo["name"] = erroredNode.Name
	}

	return map[string]any{
		"node": nodeInfo,
		"error": map[string]any{
			"reason":  execution.ResultReason,
			"message": execution.ResultMessage,
		},
		"run": map[string]any{
			"id": execution.RunID.String(),
		},
		"root":     buildRootInfo(nodes, rootEvent),
		"payloads": payloads,
	}, nil
}

// buildRootInfo describes the event that started the failed run, so On Error
// handlers can display what originally triggered the chain.
func buildRootInfo(nodes []models.CanvasNode, rootEvent *models.CanvasEvent) map[string]any {
	rootNode := findNode(nodes, rootEvent.NodeID)

	nodeInfo := map[string]any{
		"id":        rootEvent.NodeID,
		"component": componentNameForNode(rootNode),
	}
	if rootNode != nil {
		nodeInfo["name"] = rootNode.Name
	}

	return map[string]any{
		"node":    nodeInfo,
		"payload": rootEvent.Data.Data(),
	}
}

func onErrorNodesFrom(nodes []models.CanvasNode) []models.CanvasNode {
	var result []models.CanvasNode
	for i := range nodes {
		ref := nodes[i].Ref.Data()
		if ref.Trigger != nil && ref.Trigger.Name == onerror.TriggerName {
			result = append(result, nodes[i])
		}
	}

	return result
}

func findNode(nodes []models.CanvasNode, nodeID string) *models.CanvasNode {
	for i := range nodes {
		if nodes[i].NodeID == nodeID {
			return &nodes[i]
		}
	}

	return nil
}

func componentNameForNode(node *models.CanvasNode) string {
	if node == nil {
		return ""
	}

	ref := node.Ref.Data()
	switch {
	case ref.Component != nil && ref.Component.Name != "":
		return ref.Component.Name
	case ref.Trigger != nil && ref.Trigger.Name != "":
		return ref.Trigger.Name
	default:
		return ""
	}
}
