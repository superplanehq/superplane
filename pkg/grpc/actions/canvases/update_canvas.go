package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateCanvas(
	_ context.Context,
	organizationID string,
	id string,
	name *string,
	description *string,
	canvasVersioningEnabled *bool,
) (*pb.UpdateCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID)
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

	changed := false
	if name != nil {
		nextName := strings.TrimSpace(*name)
		if nextName == "" {
			return nil, status.Error(codes.InvalidArgument, "canvas name is required")
		}

		if canvas.Name != nextName {
			canvas.Name = nextName
			changed = true
		}
	}

	if description != nil && canvas.Description != *description {
		canvas.Description = *description
		changed = true
	}

	if canvasVersioningEnabled != nil && canvas.CanvasVersioningEnabled != *canvasVersioningEnabled {
		canvas.CanvasVersioningEnabled = *canvasVersioningEnabled
		changed = true
	}

	if changed {
		now := time.Now()
		canvas.UpdatedAt = &now

		if saveErr := database.Conn().Save(canvas).Error; saveErr != nil {
			if strings.Contains(saveErr.Error(), ErrDuplicateCanvasName) {
				return nil, status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
			}
			log.Errorf("failed to update canvas %s metadata: %v", canvas.ID.String(), saveErr)
			return nil, status.Error(codes.Internal, "failed to update canvas")
		}
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String()).Publish(true); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	serializedCanvas, serializeErr := SerializeCanvas(canvas, false)
	if serializeErr != nil {
		log.Errorf("failed to serialize canvas %s after update: %v", canvas.ID.String(), serializeErr)
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasResponse{
		Canvas: serializedCanvas,
	}, nil
}
