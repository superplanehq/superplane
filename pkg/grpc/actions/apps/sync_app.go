package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func SyncApp(ctx context.Context, organizationID uuid.UUID, appID string) (*pb.SyncAppResponse, error) {
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

	// Mark as syncing
	app.SyncStatus = models.AppSyncStatusSyncing
	if err := models.UpdateApp(database.Conn(), app); err != nil {
		log.Errorf("failed to update sync status for app %s: %v", app.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to update sync status")
	}

	// TODO(phase-2): Pull from Code Storage and materialize dashboard/canvas/docs.
	// For now, mark as ok immediately (stub).
	app.SyncStatus = models.AppSyncStatusOk
	app.SyncError = nil

	if err := models.UpdateApp(database.Conn(), app); err != nil {
		log.Errorf("failed to finalize sync for app %s: %v", app.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to finalize sync")
	}

	return &pb.SyncAppResponse{
		App: serializeApp(app),
	}, nil
}
