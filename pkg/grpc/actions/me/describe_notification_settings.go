package me

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"gorm.io/gorm"
)

func DescribeNotificationSettings(ctx context.Context) (*pb.DescribeNotificationSettingsResponse, error) {
	orgID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := models.FindUserNotificationSettings(database.DB(ctx), orgID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.DescribeNotificationSettingsResponse{
			Settings: defaultNotificationSettingsProto(),
		}, nil
	}
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe notification settings")
	}

	return &pb.DescribeNotificationSettingsResponse{
		Settings: serializeNotificationSettings(settings),
	}, nil
}
