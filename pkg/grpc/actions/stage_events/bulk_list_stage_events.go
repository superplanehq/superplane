package stageevents

import (
	"context"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func BulkListStageEvents(ctx context.Context, canvasID string, stages []*pb.StageEventItemRequest, limitPerStage int32, before *timestamppb.Timestamp, pbStates []pb.StageEvent_State, pbStateReasons []pb.StageEvent_StateReason) (*pb.BulkListStageEventsResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	states, err := validateStageEventStates(pbStates)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	stateReasons, err := validateStageEventStateReasons(pbStateReasons)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var beforeTime *time.Time
	if before != nil && before.IsValid() {
		t := before.AsTime()
		beforeTime = &t
	}

	stageIdentifiers := make([]string, 0, len(stages))
	for _, stageItem := range stages {
		stageIdentifiers = append(stageIdentifiers, stageItem.StageIdOrName)
	}

	stagesByIdentifier, err := models.BulkFindStagesByCanvasIDAndIdentifiers(canvasUUID, stageIdentifiers)
	if err != nil {
		return nil, err
	}

	var stageIDs []uuid.UUID
	stageIDMap := make(map[string]uuid.UUID)

	for _, stageItem := range stages {
		stage, found := stagesByIdentifier[stageItem.StageIdOrName]
		if !found {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}
		stageIDs = append(stageIDs, stage.ID)
		stageIDMap[stageItem.StageIdOrName] = stage.ID
	}

	validatedLimit := validateLimit(int(limitPerStage))

	eventsByStageID, err := models.BulkListStageEventsByCanvasIDAndMultipleStages(canvasUUID, stageIDs, validatedLimit, beforeTime, states, stateReasons)
	if err != nil {
		return nil, err
	}
	results := make([]*pb.StageEventItemResult, 0, len(stages))
	for _, stageItem := range stages {
		stageID := stageIDMap[stageItem.StageIdOrName]
		stageEvents := eventsByStageID[stageID.String()]

		serializedEvents, err := serializeStageEvents(stageEvents)
		if err != nil {
			return nil, err
		}

		result := &pb.StageEventItemResult{
			StageId: stageID.String(),
			Events:  serializedEvents,
		}

		results = append(results, result)
	}

	response := &pb.BulkListStageEventsResponse{
		Results: results,
	}

	return response, nil
}
