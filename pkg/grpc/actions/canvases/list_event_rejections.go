package canvases

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

const (
	MinLimit     = 50
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListEventRejections(ctx context.Context, canvasID string, protoTargetType pb.Connection_Type, targetID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListEventRejectionsResponse, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target id")
	}

	targetType := actions.ProtoToConnectionType(protoTargetType)
	if targetType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid target type")
	}

	limit = getLimit(limit)
	result := listAndCountEventRejectionsInParallel(targetType, id, limit, getBefore(before))

	if result.listErr != nil {
		return nil, result.listErr
	}

	if result.countErr != nil {
		return nil, result.countErr
	}

	serialized, err := serializeEventRejections(result.rejections)
	if err != nil {
		return nil, err
	}

	response := &pb.ListEventRejectionsResponse{
		Rejections:    serialized,
		TotalCount:    uint32(result.totalCount),
		HasNextPage:   result.hasNextPage(limit),
		LastTimestamp: result.lastTimestamp(),
	}

	return response, nil
}

type listAndCountEventRejectionsResult struct {
	rejections []models.EventRejection
	totalCount int64
	listErr    error
	countErr   error
}

func (r *listAndCountEventRejectionsResult) hasNextPage(limit uint32) bool {
	return len(r.rejections) == int(limit) && r.totalCount > int64(limit)
}

func (r *listAndCountEventRejectionsResult) lastTimestamp() *timestamppb.Timestamp {
	if len(r.rejections) > 0 {
		lastRejection := r.rejections[len(r.rejections)-1]
		return timestamppb.New(*lastRejection.RejectedAt)
	}

	return nil
}

func listAndCountEventRejectionsInParallel(targetType string, targetID uuid.UUID, limit uint32, beforeTime *time.Time) *listAndCountEventRejectionsResult {
	result := &listAndCountEventRejectionsResult{}
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		result.rejections, result.listErr = models.FilterEventRejections(targetType, targetID, int(limit), beforeTime)
	}()

	go func() {
		defer wg.Done()
		result.totalCount, result.countErr = models.CountEventRejections(targetType, targetID)
	}()

	wg.Wait()
	return result
}

func getLimit(limit uint32) uint32 {
	if limit == 0 {
		return DefaultLimit
	}

	if limit > MaxLimit {
		return MaxLimit
	}

	return limit
}

func getBefore(before *timestamppb.Timestamp) *time.Time {
	if before != nil && before.IsValid() {
		t := before.AsTime()
		return &t
	}

	return nil
}

func serializeEventRejections(in []models.EventRejection) ([]*pb.EventRejection, error) {
	out := []*pb.EventRejection{}
	for _, rejection := range in {
		serialized, err := serializeEventRejection(rejection)
		if err != nil {
			return nil, err
		}
		out = append(out, serialized)
	}
	return out, nil
}

func serializeEventRejection(rejection models.EventRejection) (*pb.EventRejection, error) {
	pbRejection := &pb.EventRejection{
		Id:         rejection.ID.String(),
		TargetType: actions.ConnectionTypeToProto(rejection.TargetType),
		TargetId:   rejection.TargetID.String(),
		Reason:     actions.RejectionReasonToProto(rejection.Reason),
		Message:    rejection.Message,
		RejectedAt: timestamppb.New(*rejection.RejectedAt),
	}

	if rejection.Event != nil {
		event, err := actions.SerializeEvent(*rejection.Event)
		if err != nil {
			return nil, err
		}
		pbRejection.Event = event
	}

	return pbRejection, nil
}
