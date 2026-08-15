package canvases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

const (
	// Mirrors webhook ingress' MaxEventSize, so a replay cannot store an event
	// the ingestion path would have rejected.
	MaxReplayPayloadSize = 512 * 1024

	MaxReplayInputs = 32
)

// ReplayNode replays a single canvas node in isolation from the graph. It
// creates a run and queue items but no execution: NodeQueueWorker creates
// that once the node's `ready` gate lets the item through, so callers find it
// by polling ListNodeExecutions on the returned run id.
//
// Inputs come from explicitInputs, else from sourceExecutionID's history,
// else from the legacy hand-written payload. sourceExecutionID is threaded
// through whichever branch wins, so lineage is recorded even when the inputs
// did not come from history.
func ReplayNode(
	ctx context.Context,
	db *gorm.DB,
	canvas *models.Canvas,
	nodeID string,
	sourceExecutionID *uuid.UUID,
	payload map[string]any,
	sourceNodeID *string,
	explicitInputs []models.ReplayInput,
) (*pb.ReplayNodeResponse, error) {
	node, err := canvas.FindNode(nodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, fmt.Sprintf("canvas node '%s' not found", nodeID))
		}

		return nil, err
	}

	// Resolved before the branch that consumes it, so lineage recorded from a
	// request that also carries `inputs` is scoped to this canvas too.
	var sourceExecution *models.CanvasNodeExecution
	if sourceExecutionID != nil {
		sourceExecution, err = models.FindNodeExecutionInTransaction(db, canvas.ID, *sourceExecutionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, grpcerrors.NotFound(err, "source execution not found")
			}

			return nil, err
		}
	}

	var inputs []models.ReplayInput
	switch {
	case len(explicitInputs) > 0:
		if err := validateSuppliedReplayInputs(db, node, explicitInputs); err != nil {
			return nil, err
		}

		inputs = explicitInputs
	case sourceExecution != nil:
		inputs, _, err = resolveReplayInputsFromHistory(db, node, sourceExecution)
		if err != nil {
			if errors.Is(err, models.ErrReplayMissingInputs) {
				return nil, grpcerrors.InvalidArgument(err, err.Error())
			}
			return nil, err
		}
	default:
		inputs = []models.ReplayInput{{Payload: payload, SourceNodeID: sourceNodeID}}
		if err := validateSuppliedReplayInputs(db, node, inputs); err != nil {
			return nil, err
		}
	}

	var run *models.CanvasRun
	var queueItems []*models.CanvasNodeQueueItem
	err = db.Transaction(func(tx *gorm.DB) error {
		run, queueItems, err = models.CreateMultiInputNodeReplay(tx, node, sourceExecutionID, inputs)
		return err
	})

	if err != nil {
		switch {
		case errors.Is(err, models.ErrReplayTriggerNode),
			errors.Is(err, models.ErrReplaySourceNodeRequired),
			errors.Is(err, models.ErrReplayMissingInputs):
			return nil, grpcerrors.InvalidArgument(err, err.Error())
		default:
			return nil, err
		}
	}

	queueItemIDs := make([]string, len(queueItems))
	for i, queueItem := range queueItems {
		queueItemIDs[i] = queueItem.ID.String()

		clone := *queueItem
		if err := messages.NewCanvasQueueItemMessage(clone).PublishCreated(); err != nil {
			log.Errorf("failed to publish replay queue item created RabbitMQ message: %v", err)
		}
	}

	return &pb.ReplayNodeResponse{
		RunId:        run.ID.String(),
		QueueItemIds: queueItemIDs,
	}, nil
}

// Only inputs the caller supplied are checked: history-derived inputs are
// already-stored events, and a source that has since stopped feeding the node
// is reported as detached by ResolveReplayInputs rather than refused here.
func validateSuppliedReplayInputs(db *gorm.DB, node *models.CanvasNode, inputs []models.ReplayInput) error {
	if len(inputs) > MaxReplayInputs {
		return grpcerrors.InvalidArgument(nil, fmt.Sprintf("replay refused: %d inputs exceeds the maximum of %d", len(inputs), MaxReplayInputs))
	}

	var liveSources []string
	for _, input := range inputs {
		if input.Payload != nil {
			if _, isObject := input.Payload.(map[string]any); !isObject {
				return grpcerrors.InvalidArgument(nil, "replay refused: input payload must be a JSON object")
			}
		}

		encoded, err := json.Marshal(input.Payload)
		if err != nil {
			return grpcerrors.InvalidArgument(err, "replay refused: input payload is not serializable")
		}

		if len(encoded) > MaxReplayPayloadSize {
			return grpcerrors.InvalidArgument(nil, fmt.Sprintf("replay refused: input payload of %d bytes exceeds the maximum of %d bytes", len(encoded), MaxReplayPayloadSize))
		}

		if input.SourceNodeID == nil {
			continue
		}

		if liveSources == nil {
			liveSources, err = models.DistinctIncomingSourceIDs(db, node)
			if err != nil {
				return err
			}
		}

		if !slices.Contains(liveSources, *input.SourceNodeID) {
			return grpcerrors.InvalidArgument(nil, fmt.Sprintf("replay refused: '%s' is not an incoming source of canvas node '%s'", *input.SourceNodeID, node.NodeID))
		}
	}

	return nil
}

// Reading history stays here rather than in models, which only ever resolves
// the inputs it is handed. The second return reports whether the execution
// consumed inputs retention has since erased, which the read path surfaces as
// expired rather than merely missing.
func resolveReplayInputsFromHistory(db *gorm.DB, node *models.CanvasNode, sourceExecution *models.CanvasNodeExecution) ([]models.ReplayInput, bool, error) {
	if sourceExecution.RunID != uuid.Nil {
		run, err := models.FindCanvasRunInTransaction(db, node.WorkflowID, sourceExecution.RunID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, grpcerrors.Internal(err, "failed to load source execution's run")
		}

		if err == nil && run.IsReplay {
			// A replay's payload is pinned on the run rather than tied to an
			// event, so a replay-of-a-replay survives retention.
			inputs, err := models.ReplayInputsFromRun(db, node, run, sourceExecution.ID)
			return inputs, false, err
		}
	}

	consumedEvents, err := models.ListConsumedEventsForExecution(db, sourceExecution.ID)
	if err != nil {
		return nil, false, err
	}

	if len(consumedEvents) == 0 {
		return nil, false, grpcerrors.InvalidArgument(nil, "source execution has no surviving consumed events to replay from")
	}

	eventIDs := make([]string, 0, len(consumedEvents))
	for _, consumedEvent := range consumedEvents {
		if consumedEvent.EventID != nil {
			eventIDs = append(eventIDs, consumedEvent.EventID.String())
		}
	}

	events, err := models.FindCanvasEvents(db, eventIDs)
	if err != nil {
		return nil, false, err
	}

	eventsByID := make(map[string]models.CanvasEvent, len(events))
	for _, event := range events {
		eventsByID[event.ID.String()] = event
	}

	expired := false
	inputs := make([]models.ReplayInput, 0, len(consumedEvents))
	for _, consumedEvent := range consumedEvents {
		if consumedEvent.EventID == nil {
			// Tombstone left by ON DELETE SET NULL: the execution did consume
			// an input here, retention erased it.
			expired = true
			continue
		}

		event, ok := eventsByID[consumedEvent.EventID.String()]
		if !ok {
			// Retention deleted the event between the two autocommitted reads
			// above, so the tombstone landed after we read the link row.
			// CreateMultiInputNodeReplay refuses the replay if skipping this
			// input leaves a live source uncovered.
			expired = true
			continue
		}

		// SourceExecutionID stays nil: resolving through it would collapse
		// all N inputs onto the replayed execution's single source node,
		// while each event already carries its own in NodeID.
		nodeID := event.NodeID
		inputs = append(inputs, models.ReplayInput{
			Payload:      event.Data.Data(),
			SourceNodeID: &nodeID,
		})
	}

	if len(inputs) == 0 {
		return nil, expired, grpcerrors.InvalidArgument(nil, "source execution has no surviving consumed events to replay from")
	}

	return inputs, expired, nil
}
