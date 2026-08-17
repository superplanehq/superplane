package factories

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DescribeNotificationSettings(
	ctx context.Context,
	organizationID string,
) (*pb.DescribeNotificationSettingsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe notification settings")
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
		return nil, factoryErrorToStatus(err, "failed to describe notification settings")
	}

	return &pb.DescribeNotificationSettingsResponse{
		Settings: serializeNotificationSettings(settings),
	}, nil
}
