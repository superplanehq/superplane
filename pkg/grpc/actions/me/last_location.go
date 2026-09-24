package me

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// DescribeLastLocation returns the calling user's saved "resume where you
// left off" location for the current organization, if one was saved.
func DescribeLastLocation(ctx context.Context) (*pb.DescribeLastLocationResponse, error) {
	orgID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	location, err := models.FindUserLastLocation(database.DB(ctx), orgID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.DescribeLastLocationResponse{}, nil
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe last location")
	}

	return &pb.DescribeLastLocationResponse{
		LastLocation: serializeLastLocation(location),
	}, nil
}

// SaveLastLocation records the last in-app screen the calling user visited
// in the current organization. The path must be a safe, relative,
// same-origin path; anything else is rejected as an invalid argument.
func SaveLastLocation(ctx context.Context, req *pb.SaveLastLocationRequest) (*pb.SaveLastLocationResponse, error) {
	orgID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil || req.GetPath() == "" {
		return nil, grpcerrors.InvalidArgument(nil, "path is required")
	}

	var location *models.UserLastLocation
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		location, err = models.SetUserLastLocation(tx, orgID, userID, req.GetPath())
		return err
	})
	if errors.Is(err, models.ErrUserLastLocationPathInvalid) {
		return nil, grpcerrors.InvalidArgument(err, "path must be a relative in-app path")
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to save last location")
	}

	return &pb.SaveLastLocationResponse{
		LastLocation: serializeLastLocation(location),
	}, nil
}

func serializeLastLocation(location *models.UserLastLocation) *pb.LastLocation {
	if location == nil {
		return nil
	}

	return &pb.LastLocation{
		Path:      location.Path,
		UpdatedAt: timestamppb.New(location.UpdatedAt),
	}
}
