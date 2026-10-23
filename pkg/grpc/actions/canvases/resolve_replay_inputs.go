package canvases

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

// attributedReplayInput pairs a recovered input with the source node the write
// path would attribute it to.
type attributedReplayInput struct {
	sourceNodeID string
	input        models.ReplayInput
}

// ResolveReplayInputs reports the inputs a replay would use, reusing the
// write path's own resolver and live-source lookup so what is shown and what
// would launch cannot diverge.
//
// A live source with no surviving payload is returned with an empty payload
// rather than omitted, so the caller sees a slot to fill - expired when
// retention erased an input of the replayed execution, missing otherwise. A
// recovered payload whose source no longer feeds the node is detached and
// must never be replayed: merge waits only on live sources.
func ResolveReplayInputs(ctx context.Context, db *gorm.DB, canvas *models.Canvas, nodeID string, sourceExecutionID uuid.UUID) (*pb.ResolveReplayInputsResponse, error) {
	node, err := canvas.FindNode(nodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, fmt.Sprintf("canvas node '%s' not found", nodeID))
		}

		return nil, err
	}

	sourceExecution, err := models.FindNodeExecutionInTransaction(db, canvas.ID, sourceExecutionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "source execution not found")
		}

		return nil, err
	}

	if sourceExecution.NodeID != node.NodeID {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("source execution does not belong to canvas node '%s'", nodeID))
	}

	recovered, expired, err := resolveReplayInputsFromHistory(db, node, sourceExecution)
	if err != nil {
		if errors.Is(err, models.ErrReplayMissingInputs) {
			return nil, grpcerrors.InvalidArgument(err, err.Error())
		}
		return nil, err
	}

	liveSources, err := models.DistinctIncomingSourceIDs(db, node)
	if err != nil {
		return nil, err
	}

	attributed := make([]attributedReplayInput, 0, len(recovered))
	for _, input := range recovered {
		sourceNodeID, err := models.ResolveReplaySourceNodeID(db, node, input)
		if err != nil {
			if errors.Is(err, models.ErrReplaySourceNodeRequired) {
				return nil, grpcerrors.InvalidArgument(err, err.Error())
			}
			return nil, err
		}

		attributed = append(attributed, attributedReplayInput{sourceNodeID: sourceNodeID, input: input})
	}

	resolved := make([]*pb.ResolvedReplayInput, 0, len(liveSources)+len(attributed))
	for _, source := range liveSources {
		found := false
		for _, entry := range attributed {
			if entry.sourceNodeID != source {
				continue
			}

			payload, err := replayInputPayloadStruct(entry.input)
			if err != nil {
				return nil, err
			}

			resolved = append(resolved, &pb.ResolvedReplayInput{
				SourceNodeId: source,
				Payload:      payload,
				Status:       pb.ResolvedReplayInput_STATUS_RECOVERED,
			})
			found = true
		}

		if !found {
			resolved = append(resolved, &pb.ResolvedReplayInput{
				SourceNodeId: source,
				Status:       uncoveredSourceStatus(expired),
			})
		}
	}

	for _, entry := range attributed {
		if slices.Contains(liveSources, entry.sourceNodeID) {
			continue
		}

		payload, err := replayInputPayloadStruct(entry.input)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, &pb.ResolvedReplayInput{
			SourceNodeId: entry.sourceNodeID,
			Payload:      payload,
			Status:       pb.ResolvedReplayInput_STATUS_DETACHED,
		})
	}

	return &pb.ResolveReplayInputsResponse{Inputs: resolved}, nil
}

// A tombstone carries no attribution - the event that held it is gone - so
// every empty slot is reported expired once the execution has one. Warning
// about a source that was merely added later is the harmless direction; the
// reverse would promise a faithful replay we cannot deliver.
func uncoveredSourceStatus(expired bool) pb.ResolvedReplayInput_Status {
	if expired {
		return pb.ResolvedReplayInput_STATUS_EXPIRED
	}

	return pb.ResolvedReplayInput_STATUS_MISSING
}

func replayInputPayloadStruct(input models.ReplayInput) (*structpb.Struct, error) {
	if input.Payload == nil {
		return newStructpbStruct(map[string]any{})
	}

	payload, ok := input.Payload.(map[string]any)
	if !ok {
		return nil, grpcerrors.InvalidArgument(nil, "recovered replay input payload is not a JSON object")
	}

	return newStructpbStruct(payload)
}
