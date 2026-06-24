package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DeleteCanvas(ctx context.Context, registry *registry.Registry, organizationID uuid.UUID, id string) (*pb.DeleteCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvas(organizationID, canvasID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	// Perform soft delete on the canvas with name suffix
	// The cleanup worker will handle the actual deletion of nodes and related data
	err = canvas.SoftDelete()
	if err != nil {
		log.Errorf("failed to delete canvas %s: %v", canvas.ID.String(), err)
		return nil, grpcerrors.Internal(err, "failed to delete canvas")
	}

	if err := messages.NewCanvasDeletedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishDeleted(); err != nil {
		log.Errorf("failed to publish canvas deleted RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasResponse{}, nil
}
