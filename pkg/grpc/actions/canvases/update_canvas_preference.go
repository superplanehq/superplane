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
	db *gorm.DB,
	canvas *models.Canvas,
	userID string,
	req *pb.UpdateCanvasPreferenceRequest,
) (*pb.UpdateCanvasPreferenceResponse, error) {
	if req == nil {
		return nil, grpcerrors.InvalidArgument(nil, "request is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid user id")
	}

	var preference *models.UserCanvasPreference
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		preference, err = models.SetUserCanvasPreference(
			tx,
			canvas.OrganizationID,
			userUUID,
			canvas.ID,
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
	if preference == nil {
		return nil
	}

	serialized := &pb.CanvasPreference{
		CanvasId: preference.CanvasID.String(),
		Starred:  preference.StarredAt != nil,
	}

	if preference.StarredAt != nil {
		serialized.StarredAt = timestamppb.New(*preference.StarredAt)
	}

	return serialized
}
