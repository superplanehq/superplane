package actions

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateTagState(ctx context.Context, req *pb.UpdateTagStateRequest) (*pb.UpdateTagStateResponse, error) {
	if req.Tag == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tag")
	}

	if req.Tag.Name == "" || req.Tag.Value == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing tag name or value")
	}

	state := tagStateFromProto(req.Tag.State)
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		err := models.UpdateTagStateInTransaction(tx, req.Tag.Name, req.Tag.Value, state)
		if err != nil {
			return fmt.Errorf("error updating tags: %v", err)
		}

		//
		// If we are not marking the tag as healthy,
		// we don't need to do anything else,
		// since the pending stage events will take care of that.
		//
		if state != models.TagStateHealthy {
			return nil
		}

		//
		// If we are marking the tag as healthy,
		// we need to move all stage events with this tag
		// that are in waiting(unhealthy) state back to pending state.
		//
		ids, err := models.FindStageEventsByTagInTransaction(tx,
			req.Tag.Name,
			req.Tag.Value,
			models.StageEventStateWaiting,
			models.StageEventStateReasonUnhealthy,
		)

		if err != nil {
			return fmt.Errorf("error finding stage events to update: %v", err)
		}

		return models.UpdateStageEventsInTransaction(tx, ids, models.StageEventStatePending, "")
	})

	if err != nil {
		log.Errorf("Error updating tag state for %v: %v", req, err)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &pb.UpdateTagStateResponse{}, nil
}

func tagStateFromProto(state pb.Tag_State) string {
	switch state {
	case pb.Tag_TAG_STATE_HEALTHY:
		return models.TagStateHealthy
	case pb.Tag_TAG_STATE_UNHEALTHY:
		return models.TagStateUnhealthy
	default:
		return models.TagStateUnknown
	}
}

func tagStateToProto(state string) pb.Tag_State {
	switch state {
	case models.TagStateHealthy:
		return pb.Tag_TAG_STATE_HEALTHY
	case models.TagStateUnhealthy:
		return pb.Tag_TAG_STATE_UNHEALTHY
	default:
		return pb.Tag_TAG_STATE_UNKNOWN
	}
}
