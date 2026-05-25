package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/canvasstorage"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteCanvas(ctx context.Context, registry *registry.Registry, organizationID uuid.UUID, id string, storage canvasstorage.Provider) (*pb.DeleteCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(organizationID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, templateErr := models.FindCanvasTemplate(canvasID); templateErr == nil {
				return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
			}
		}
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	if err := deleteCanvasRepository(ctx, canvas, storage); err != nil {
		return nil, err
	}

	// Perform soft delete on the canvas with name suffix
	// The cleanup worker will handle the actual deletion of nodes and related data
	err = canvas.SoftDelete()
	if err != nil {
		log.Errorf("failed to delete canvas %s: %v", canvas.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to delete canvas")
	}

	if err := messages.NewCanvasDeletedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishDeleted(); err != nil {
		log.Errorf("failed to publish canvas deleted RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasResponse{}, nil
}

func deleteCanvasRepository(ctx context.Context, canvas *models.Canvas, storage canvasstorage.Provider) error {
	if storage == nil {
		return nil
	}

	repository, err := models.FindCanvasRepository(canvas.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return status.Error(codes.Internal, "failed to load canvas repository")
	}

	if err := storage.DeleteRepository(ctx, canvasRepositoryRef(repository)); err != nil {
		return canvasStorageStatusError(err)
	}

	return nil
}
