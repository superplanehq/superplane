package models

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	CanvasOnErrorEventSourceNodeID = "canvas.onError"
	CanvasOnErrorEventChannel      = "default"
	CanvasOnErrorEventType         = "canvas.onError"
)

type OnErrorDispatch struct {
	Event     CanvasEvent
	QueueItem CanvasNodeQueueItem
}

func FindOnErrorNodeID(nodes []Node) string {
	for _, node := range nodes {
		if node.OnError && node.Type != NodeTypeTrigger {
			return node.ID
		}
	}

	return ""
}

func NormalizeOnErrorNodes(nodes []Node) []Node {
	onErrorIndex := -1
	for i := range nodes {
		if !nodes[i].OnError {
			continue
		}

		if nodes[i].Type == NodeTypeTrigger {
			nodes[i].OnError = false
			continue
		}

		if onErrorIndex >= 0 {
			nodes[i].OnError = false
			continue
		}

		onErrorIndex = i
	}

	return nodes
}

func MaybeScheduleCanvasOnErrorInTransaction(
	tx *gorm.DB,
	failedExecution *CanvasNodeExecution,
) (*OnErrorDispatch, error) {
	if failedExecution == nil {
		return nil, nil
	}

	if failedExecution.ParentExecutionID != nil {
		return nil, nil
	}

	if failedExecution.ResultReason != CanvasNodeExecutionResultReasonError {
		return nil, nil
	}

	version, err := FindLiveCanvasVersionInTransaction(tx, failedExecution.WorkflowID)
	if err != nil {
		return nil, err
	}

	onErrorNodeID := FindOnErrorNodeID(version.Nodes)
	if onErrorNodeID == "" {
		return nil, nil
	}

	if onErrorNodeID == failedExecution.NodeID {
		return nil, nil
	}

	onErrorNode, err := FindCanvasNode(tx, failedExecution.WorkflowID, onErrorNodeID)
	if err != nil {
		if errorsIsRecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if onErrorNode.State == CanvasNodeStateError {
		return nil, nil
	}

	failedNode, err := FindCanvasNode(tx, failedExecution.WorkflowID, failedExecution.NodeID)
	if err != nil && !errorsIsRecordNotFound(err) {
		return nil, err
	}

	failedNodeName := failedExecution.NodeID
	if failedNode != nil && failedNode.Name != "" {
		failedNodeName = failedNode.Name
	}

	now := time.Now()
	payload := map[string]any{
		"type":      CanvasOnErrorEventType,
		"timestamp": now,
		"data": map[string]any{
			"failedNodeId":   failedExecution.NodeID,
			"failedNodeName": failedNodeName,
			"errorMessage":   failedExecution.ResultMessage,
			"executionId":    failedExecution.ID.String(),
			"runId":          failedExecution.RunID.String(),
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	event := CanvasEvent{
		WorkflowID:  failedExecution.WorkflowID,
		NodeID:      CanvasOnErrorEventSourceNodeID,
		Channel:     CanvasOnErrorEventChannel,
		Data:        NewJSONValue(json.RawMessage(payloadJSON)),
		ExecutionID: &failedExecution.ID,
		RunID:       failedExecution.RunID,
		State:       CanvasEventStatePending,
		CreatedAt:   &now,
	}

	if err := tx.Create(&event).Error; err != nil {
		return nil, err
	}

	queueItem := CanvasNodeQueueItem{
		WorkflowID:  failedExecution.WorkflowID,
		NodeID:      onErrorNode.NodeID,
		RootEventID: event.ID,
		RunID:       failedExecution.RunID,
		EventID:     event.ID,
		CreatedAt:   &now,
	}

	if err := tx.Create(&queueItem).Error; err != nil {
		return nil, err
	}

	return &OnErrorDispatch{
		Event:     event,
		QueueItem: queueItem,
	}, nil
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
