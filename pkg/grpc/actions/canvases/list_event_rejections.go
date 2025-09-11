package canvases

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ListEventRejections(ctx context.Context, canvasID string, componentType string, componentID uuid.UUID) (*pb.ListEventRejectionsResponse, error) {
	rejections, err := models.ListEventRejections(componentType, componentID)
	if err != nil {
		log.Errorf("Error finding stage rejections: %v", err)
		return nil, err
	}

	var pbRejections []*pb.EventRejection
	for _, rejection := range rejections {
		pbRejections = append(pbRejections, &pb.EventRejection{
			Id:            rejection.ID.String(),
			EventId:       rejection.EventID.String(),
			ComponentType: rejection.ComponentType,
			ComponentId:   rejection.ComponentID.String(),
			Reason:        actions.RejectionReasonToProto(rejection.Reason),
			Message:       rejection.Message,
			RejectedAt:    timestamppb.New(*rejection.RejectedAt),
		})
	}

	return &pb.ListEventRejectionsResponse{
		Rejections: pbRejections,
	}, nil
}
