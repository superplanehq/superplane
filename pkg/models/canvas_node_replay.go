package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// Trigger nodes have no Execute path; ReemitTriggerEvent re-runs them.
	ErrReplayTriggerNode = errors.New("trigger nodes cannot be replayed")

	ErrReplaySourceNodeRequired = errors.New("node has multiple incoming edges: source node id is required")

	// Refused up front: an aggregating node whose inputs do not cover every
	// live incoming source would wait forever for one that never arrives.
	ErrReplayMissingInputs = errors.New("replay refused: missing input(s) for distinct incoming source(s)")
)

// Components whose ProcessQueueItem finds-or-creates one execution keyed on
// RootEventID and waits for every distinct incoming source before finishing
// (see pkg/components/merge/merge.go). pkg/registry imports pkg/models, so the
// property cannot be read off the component here; a component missing from
// this list fails the classification test in pkg/components/merge.
var aggregatingComponents = []string{"merge"}

// ReplayInput is one logical input of a replay; N of them are passed as a
// slice. SourceExecutionID, when set, attributes the input via that
// execution's own input event and takes precedence over SourceNodeID, which
// otherwise falls back to edge counting (see ResolveReplaySourceNodeID).
type ReplayInput struct {
	Payload           any
	SourceExecutionID *uuid.UUID
	SourceNodeID      *string
}

func IsAggregatingComponent(name string) bool {
	return slices.Contains(aggregatingComponents, name)
}

// CreateMultiInputNodeReplay enters through the normal queue rather than
// creating an execution directly, so the replay respects LockCanvasNode's
// `ready` gate and rebuilds configuration from the node's live definition. The
// caller owns the transaction and the post-commit publish of the returned
// queue items.
//
// Every input event shares one root, so an aggregating node's find-or-create
// accumulates all N queue items into a single execution instead of N that
// each wait forever. sourceExecutionID is the whole replay's lineage, not any
// individual input's source; it may be nil for a hand-written replay.
func CreateMultiInputNodeReplay(tx *gorm.DB, node *CanvasNode, sourceExecutionID *uuid.UUID, inputs []ReplayInput) (*CanvasRun, []*CanvasNodeQueueItem, error) {
	if len(inputs) == 0 {
		return nil, nil, fmt.Errorf("at least one replay input is required")
	}

	ref := node.Ref.Data()
	if ref.Trigger != nil {
		return nil, nil, ErrReplayTriggerNode
	}

	sourceNodeIDs := make([]string, len(inputs))
	for i, input := range inputs {
		sourceNodeID, err := ResolveReplaySourceNodeID(tx, node, input)
		if err != nil {
			return nil, nil, err
		}
		sourceNodeIDs[i] = sourceNodeID
	}

	if isAggregatingNode(node) {
		incomingSources, err := DistinctIncomingSourceIDs(tx, node)
		if err != nil {
			return nil, nil, fmt.Errorf("find distinct incoming sources: %w", err)
		}

		missing := make([]string, 0)
		for _, source := range incomingSources {
			if !slices.Contains(sourceNodeIDs, source) {
				missing = append(missing, source)
			}
		}

		if len(missing) > 0 {
			return nil, nil, fmt.Errorf("%w: %s", ErrReplayMissingInputs, strings.Join(missing, ", "))
		}
	}

	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, node.WorkflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("find live canvas version: %w", err)
	}

	now := time.Now()
	run := &CanvasRun{
		WorkflowID:              node.WorkflowID,
		NodeID:                  node.NodeID,
		VersionID:               liveVersion.ID,
		State:                   CanvasRunStateStarted,
		IsReplay:                true,
		ReplaySourceExecutionID: sourceExecutionID,
		ReplayPayload:           NewJSONValue(replayPayloadValue(inputs, sourceNodeIDs)),
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, nil, fmt.Errorf("create replay run: %w", err)
	}

	// RunID is set explicitly so CanvasEvent.BeforeCreate does not manufacture
	// a run for these events, and State is Routed so EventRouter's edge-based
	// fan-out never picks them up: they exist only to carry each payload and
	// its attribution into the queue items below.
	events := make([]*CanvasEvent, len(inputs))
	for i, input := range inputs {
		event := &CanvasEvent{
			WorkflowID: node.WorkflowID,
			NodeID:     sourceNodeIDs[i],
			Channel:    "replay",
			Data:       NewJSONValue(input.Payload),
			RunID:      run.ID,
			State:      CanvasEventStateRouted,
			CreatedAt:  &now,
		}

		if err := tx.Create(event).Error; err != nil {
			return nil, nil, fmt.Errorf("create replay input event: %w", err)
		}

		events[i] = event
	}

	rootEventID := events[0].ID

	queueItems := make([]*CanvasNodeQueueItem, len(inputs))
	for i, event := range events {
		queueItem := &CanvasNodeQueueItem{
			WorkflowID:  node.WorkflowID,
			NodeID:      node.NodeID,
			RootEventID: rootEventID,
			RunID:       run.ID,
			EventID:     event.ID,
			CreatedAt:   &now,
		}

		if err := tx.Create(queueItem).Error; err != nil {
			return nil, nil, fmt.Errorf("create replay queue item: %w", err)
		}

		queueItems[i] = queueItem
	}

	return run, queueItems, nil
}

// ReplayInputsFromRun reads a replay run's pinned ReplayPayload back into
// inputs, so a replay can itself be replayed. It tolerates all three shapes
// ReplayPayload has been written in: a bare payload, a list of
// {payload, sourceNodeId} pairs, and - refused rather than silently collapsed
// into one input - a bare list of payloads carrying no attribution.
// Per-input SourceExecutionID stays nil: resolving through it would attribute
// every input to the same source node.
func ReplayInputsFromRun(tx *gorm.DB, node *CanvasNode, run *CanvasRun, sourceExecutionID uuid.UUID) ([]ReplayInput, error) {
	list, isList := run.ReplayPayload.Data().([]any)
	if !isList {
		return []ReplayInput{{
			Payload:           run.ReplayPayload.Data(),
			SourceExecutionID: &sourceExecutionID,
		}}, nil
	}

	if inputs, ok := attributedReplayInputs(list); ok {
		return inputs, nil
	}

	incomingSources, err := DistinctIncomingSourceIDs(tx, node)
	if err != nil {
		return nil, fmt.Errorf("find distinct incoming sources: %w", err)
	}

	if len(incomingSources) == 0 {
		return nil, fmt.Errorf("%w: replayed run has no source attribution recorded", ErrReplayMissingInputs)
	}

	return nil, fmt.Errorf("%w: %s", ErrReplayMissingInputs, strings.Join(incomingSources, ", "))
}

// DistinctIncomingSourceIDs lets the read side label inputs against the same
// live-source list the write path enforces, so what is shown and what is
// launchable cannot diverge. Mirrors process_queue_context.go's
// DistinctIncomingSources, reimplemented here because pkg/models must not
// import pkg/workers/contexts.
func DistinctIncomingSourceIDs(tx *gorm.DB, node *CanvasNode) ([]string, error) {
	_, edges, err := FindLiveCanvasSpecInTransaction(tx, node.WorkflowID)
	if err != nil {
		return nil, err
	}

	sources := make([]string, 0)
	for _, edge := range edges {
		if edge.TargetID == node.NodeID && !slices.Contains(sources, edge.SourceID) {
			sources = append(sources, edge.SourceID)
		}
	}

	return sources, nil
}

// ResolveReplaySourceNodeID takes the source node from the original
// execution's own input event when replaying from history; a hand-written
// payload falls back to edge counting. Exported so the read side attributes an
// input exactly the way a launch would.
func ResolveReplaySourceNodeID(tx *gorm.DB, node *CanvasNode, input ReplayInput) (string, error) {
	if input.SourceExecutionID != nil {
		sourceExecution, err := FindNodeExecutionInTransaction(tx, node.WorkflowID, *input.SourceExecutionID)
		if err != nil {
			return "", fmt.Errorf("find source execution: %w", err)
		}

		if sourceExecution.EventID != uuid.Nil {
			sourceEvent, err := FindCanvasEventInTransaction(tx, sourceExecution.EventID)
			if err == nil {
				return sourceEvent.NodeID, nil
			}

			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return "", fmt.Errorf("find source execution's input event: %w", err)
			}
			// Event row gone (retention) - fall through to the explicit
			// override below, same as the uuid.Nil case.
		}

		if input.SourceNodeID != nil {
			return *input.SourceNodeID, nil
		}

		return "", fmt.Errorf("source execution %s has no retrievable input event; source node id is required", *input.SourceExecutionID)
	}

	if input.SourceNodeID != nil {
		return *input.SourceNodeID, nil
	}

	_, edges, err := FindLiveCanvasSpecInTransaction(tx, node.WorkflowID)
	if err != nil {
		return "", fmt.Errorf("find live canvas spec: %w", err)
	}

	incoming := make([]Edge, 0)
	for _, edge := range edges {
		if edge.TargetID == node.NodeID {
			incoming = append(incoming, edge)
		}
	}

	switch len(incoming) {
	case 0:
		return node.NodeID, nil
	case 1:
		return incoming[0].SourceID, nil
	default:
		return "", ErrReplaySourceNodeRequired
	}
}

func isAggregatingNode(node *CanvasNode) bool {
	ref := node.Ref.Data()
	return ref.Component != nil && IsAggregatingComponent(ref.Component.Name)
}

// Persisted shape of one entry in a multi-input CanvasRun.ReplayPayload.
// ReplayPayload is never serialized outward, so this shape can change without
// an API contract change.
type replayPayloadPair struct {
	Payload      any    `json:"payload"`
	SourceNodeID string `json:"sourceNodeId"`
}

// A single input stays a bare payload - readers still rely on that shape.
// Several are pinned as attributed pairs rather than a bare list, which
// ReplayInputsFromRun could not expand again.
func replayPayloadValue(inputs []ReplayInput, sourceNodeIDs []string) any {
	if len(inputs) == 1 {
		return inputs[0].Payload
	}

	pairs := make([]replayPayloadPair, len(inputs))
	for i, input := range inputs {
		pairs[i] = replayPayloadPair{Payload: input.Payload, SourceNodeID: sourceNodeIDs[i]}
	}
	return pairs
}

// Reports false for anything that is not a full list of attributed pairs,
// including an empty list: a replay run always pins at least one input, so an
// empty one carries no attribution to replay from.
func attributedReplayInputs(list []any) ([]ReplayInput, bool) {
	if len(list) == 0 {
		return nil, false
	}

	inputs := make([]ReplayInput, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}

		sourceNodeID, ok := entry["sourceNodeId"].(string)
		if !ok || sourceNodeID == "" {
			return nil, false
		}

		payload, ok := entry["payload"]
		if !ok {
			return nil, false
		}

		inputs = append(inputs, ReplayInput{Payload: payload, SourceNodeID: &sourceNodeID})
	}

	return inputs, true
}
