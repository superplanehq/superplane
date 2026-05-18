package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteApp(ctx context.Context, organizationID uuid.UUID, appID string) (*pb.DeleteAppResponse, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid app id: %v", err)
	}

	app, err := models.FindApp(organizationID, appUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		return nil, status.Error(codes.Internal, "failed to load app")
	}

	// TODO(phase-2): Delete Code Storage repository via provider API before soft-deleting.

	if err := app.SoftDelete(); err != nil {
		log.Errorf("failed to delete app %s: %v", app.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to delete app")
	}

	return &pb.DeleteAppResponse{}, nil
}
