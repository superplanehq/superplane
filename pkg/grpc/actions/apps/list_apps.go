package apps

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListApps(ctx context.Context, organizationID string) (*pb.ListAppsResponse, error) {
	appList, err := models.ListApps(organizationID)
	if err != nil {
		log.Errorf("failed to list apps for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list apps")
	}

	return &pb.ListAppsResponse{
		Apps: serializeApps(appList),
	}, nil
}
