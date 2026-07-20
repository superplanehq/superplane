package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func UpdateCanvasPreference(
	ctx context.Context,
	organizationID string,
	userID string,
	req *pb.UpdateCanvasPreferenceRequest,
) (*pb.UpdateCanvasPreferenceResponse, error) {
	if req == nil {
		return nil, grpcerrors.InvalidArgument(nil, "request is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid user id")
	}

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	var preference *models.UserCanvasPreference
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		preference, err = models.SetUserCanvasPreference(
			tx,
			organizationUUID,
			userUUID,
			canvasID,
			req.Starred,
		)
		return err
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update canvas preference")
	}

	return &pb.UpdateCanvasPreferenceResponse{
		Preference: serializeCanvasPreference(preference),
	}, nil
}

func serializeCanvasPreference(preference *models.UserCanvasPreference) *pb.CanvasPreference {
	serialized := &pb.CanvasPreference{
		CanvasId: preference.CanvasID.String(),
		Starred:  preference.StarredAt != nil,
	}

	if preference.StarredAt != nil {
		serialized.StarredAt = timestamppb.New(*preference.StarredAt)
	}

	return serialized
}
