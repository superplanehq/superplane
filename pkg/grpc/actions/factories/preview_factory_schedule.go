package factories

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/triggers/schedule"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PreviewFactorySchedule validates settings without creating or activating a trigger.
func PreviewFactorySchedule(ctx context.Context, organizationID string, req *pb.PreviewFactoryScheduleRequest) (*pb.PreviewFactoryScheduleResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to preview schedule")
	}
	db := database.DB(ctx)
	org, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to preview schedule")
	}
	if err := requireScheduledIntake(org); err != nil {
		return nil, err
	}
	if _, err := findFactory(db, orgID, req.GetFactoryId()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to preview schedule")
	}
	config, err := scheduledIntakeSchedule(req.GetInstallParams())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to preview schedule")
	}
	next, err := schedule.NextTrigger(config, time.Now())
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument(err.Error()), "failed to preview schedule")
	}
	return &pb.PreviewFactoryScheduleResponse{NextRun: timestamppb.New(*next)}, nil
}
