package canvases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// UpdateCanvasLaunchpad replaces the launchpad dashboard for a canvas with the
// provided panels and layout. The launchpad is always-live and not coupled to
// the canvas version lifecycle, so this is a single atomic upsert.
func UpdateCanvasLaunchpad(ctx context.Context, organizationID, canvasID string, panels []*pb.LaunchpadPanel, layout []*pb.LaunchpadLayoutItem) (*pb.UpdateCanvasLaunchpadResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	modelPanels, err := deserializeLaunchpadPanels(panels)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	modelLayout := deserializeLaunchpadLayout(layout)

	if err := validateLaunchpadInput(modelPanels, modelLayout); err != nil {
		return nil, err
	}

	var saved *models.CanvasLaunchpad
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		record, upsertErr := models.UpsertCanvasLaunchpadInTransaction(tx, canvas.ID, modelPanels, modelLayout)
		if upsertErr != nil {
			return upsertErr
		}
		saved = record
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to update canvas launchpad")
		return nil, status.Error(codes.Internal, "failed to update canvas launchpad")
	}

	serialized, err := serializeCanvasLaunchpad(saved)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas launchpad")
	}

	return &pb.UpdateCanvasLaunchpadResponse{Launchpad: serialized}, nil
}

func validateLaunchpadInput(panels []models.LaunchpadPanel, layout []models.LaunchpadLayoutItem) error {
	if len(panels) > MaxLaunchpadPanels {
		return status.Errorf(codes.InvalidArgument, "too many panels (max %d)", MaxLaunchpadPanels)
	}

	panelIDs := make(map[string]struct{}, len(panels))
	for _, panel := range panels {
		if panel.ID == "" {
			return status.Error(codes.InvalidArgument, "panel id is required")
		}
		if panel.Type == "" {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("panel %q type is required", panel.ID))
		}
		if _, exists := panelIDs[panel.ID]; exists {
			return status.Errorf(codes.InvalidArgument, "duplicate panel id %q", panel.ID)
		}
		panelIDs[panel.ID] = struct{}{}
	}

	size, err := encodedPanelsSize(panels)
	if err != nil {
		return status.Error(codes.Internal, "failed to validate panel size")
	}
	if size > MaxLaunchpadPayloadBytes {
		return status.Errorf(codes.InvalidArgument, "panels payload exceeds %d bytes", MaxLaunchpadPayloadBytes)
	}

	layoutIDs := make(map[string]struct{}, len(layout))
	for _, item := range layout {
		if item.I == "" {
			return status.Error(codes.InvalidArgument, "layout item i is required")
		}
		if _, exists := layoutIDs[item.I]; exists {
			return status.Errorf(codes.InvalidArgument, "duplicate layout id %q", item.I)
		}
		layoutIDs[item.I] = struct{}{}

		if _, ok := panelIDs[item.I]; !ok {
			return status.Errorf(codes.InvalidArgument, "layout item %q does not reference any panel", item.I)
		}
		if item.W <= 0 || item.H <= 0 {
			return status.Errorf(codes.InvalidArgument, "layout item %q must have positive width and height", item.I)
		}
		if item.X < 0 || item.Y < 0 {
			return status.Errorf(codes.InvalidArgument, "layout item %q must have non-negative x and y", item.I)
		}
	}

	return nil
}
