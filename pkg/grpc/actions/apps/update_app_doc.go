package apps

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateAppDoc(ctx context.Context, organizationID, appID, path, content string) (*pb.UpdateAppDocResponse, error) {
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

	now := time.Now()
	doc := &models.AppDoc{
		ID:        uuid.New(),
		AppID:     appUUID,
		Path:      path,
		Content:   content,
		UpdatedAt: &now,
	}

	var saved *models.AppDoc
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var upsertErr error
		saved, upsertErr = models.UpsertAppDoc(tx, doc)
		return upsertErr
	})
	if err != nil {
		log.WithError(err).Error("failed to update app doc")
		return nil, status.Error(codes.Internal, "failed to update app doc")
	}

	// TODO(phase-2): Commit file under docs/ to Code Storage.

	return &pb.UpdateAppDocResponse{
		Doc: serializeAppDoc(saved),
	}, nil
}
