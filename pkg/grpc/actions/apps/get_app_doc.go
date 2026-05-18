package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func GetAppDoc(ctx context.Context, organizationID, appID, path string) (*pb.GetAppDocResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid app_id")
	}

	if path == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	_, err = models.FindApp(orgUUID, appUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "app not found")
		}
		return nil, status.Error(codes.Internal, "failed to load app")
	}

	doc, err := models.FindAppDocByPath(appUUID, path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "doc not found")
		}
		return nil, status.Error(codes.Internal, "failed to load doc")
	}

	return &pb.GetAppDocResponse{
		Doc: serializeAppDoc(doc),
	}, nil
}
