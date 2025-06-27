package connectiongroups

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListConnectionGroupFieldSets(ctx context.Context, req *pb.ListConnectionGroupFieldSetsRequest) (*pb.ListConnectionGroupFieldSetsResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)

	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = canvas.FindConnectionGroupByName(req.IdOrName)
	} else {
		connectionGroup, err = canvas.FindConnectionGroupByID(uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "connection group not found")
		}

		return nil, err
	}

	events, err := connectionGroup.ListFieldSets()
	if err != nil {
		return nil, err
	}

	serialized, err := serializeConnectionGroupFieldSets(events)
	if err != nil {
		return nil, err
	}

	response := &pb.ListConnectionGroupFieldSetsResponse{
		FieldSets: serialized,
	}

	return response, nil
}

func serializeConnectionGroupFieldSets(in []models.ConnectionGroupFieldSet) ([]*pb.ConnectionGroupFieldSet, error) {
	out := []*pb.ConnectionGroupFieldSet{}
	for _, i := range in {
		e, err := serializeConnectionGroupFieldSet(i)
		if err != nil {
			return nil, err
		}

		out = append(out, e)
	}

	return out, nil
}

// TODO: very inefficient way of querying the events for the field set that we should fix later
func serializeConnectionGroupFieldSet(in models.ConnectionGroupFieldSet) (*pb.ConnectionGroupFieldSet, error) {
	fieldSet := pb.ConnectionGroupFieldSet{
		Id:        in.ID.String(),
		Fields:    []*pb.KeyValuePair{},
		Hash:      in.FieldSetHash,
		State:     fieldSetStateToProto(in.State),
		Result:    fieldSetResultToProto(in.Result),
		CreatedAt: timestamppb.New(*in.CreatedAt),
	}

	for k, v := range in.FieldSet.Data() {
		fieldSet.Fields = append(fieldSet.Fields, &pb.KeyValuePair{
			Name:  k,
			Value: v,
		})
	}

	//
	// Add events
	//
	events, err := in.FindEvents()
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		fieldSet.Events = append(fieldSet.Events, &pb.ConnectionGroupEvent{
			Id:         event.ID.String(),
			SourceId:   event.SourceID.String(),
			SourceName: event.SourceName,
			SourceType: actions.ConnectionTypeToProto(event.SourceType),
			ReceivedAt: timestamppb.New(*event.ReceivedAt),
		})
	}

	return &fieldSet, nil
}

func fieldSetStateToProto(state string) pb.ConnectionGroupFieldSet_State {
	switch state {
	case models.ConnectionGroupFieldSetStatePending:
		return pb.ConnectionGroupFieldSet_STATE_PENDING
	case models.ConnectionGroupFieldSetStateProcessed:
		return pb.ConnectionGroupFieldSet_STATE_PROCESSED
	default:
		return pb.ConnectionGroupFieldSet_STATE_UNKNOWN
	}
}

func fieldSetResultToProto(result string) pb.ConnectionGroupFieldSet_Result {
	switch result {
	case models.ConnectionGroupFieldSetResultTimedOut:
		return pb.ConnectionGroupFieldSet_RESULT_TIMED_OUT
	case models.ConnectionGroupFieldSetResultReceivedAll:
		return pb.ConnectionGroupFieldSet_RESULT_RECEIVED_ALL
	default:
		return pb.ConnectionGroupFieldSet_RESULT_NONE
	}
}
