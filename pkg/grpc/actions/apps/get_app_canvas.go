package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pbApps "github.com/superplanehq/superplane/pkg/protos/apps"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func GetAppCanvas(ctx context.Context, reg *registry.Registry, organizationID, appID string) (*pbApps.GetAppCanvasResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid app_id")
	}

	app, err := models.FindApp(orgUUID, appUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		return nil, status.Error(codes.Internal, "failed to load app")
	}

	if app.CanvasID == nil {
		return &pbApps.GetAppCanvasResponse{Canvas: &pbCanvases.Canvas{}}, nil
	}

	canvas, err := models.FindCanvas(orgUUID, *app.CanvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found for app")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to load canvas author")
		}
	}

	proto, err := canvases.SerializeCanvas(canvas, true, user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pbApps.GetAppCanvasResponse{Canvas: proto}, nil
}
