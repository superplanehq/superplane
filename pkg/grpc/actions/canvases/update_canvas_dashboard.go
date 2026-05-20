package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateCanvasDashboard(ctx context.Context, organizationID, canvasID string, panels []*pb.DashboardPanel, layout []*pb.DashboardLayoutItem) (*pb.UpdateCanvasDashboardResponse, error) {
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

	modelPanels, err := deserializeDashboardPanels(panels)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	modelLayout := deserializeDashboardLayout(layout)

	if err := validateDashboardInput(modelPanels, modelLayout); err != nil {
		return nil, err
	}

	var saved *models.CanvasDashboard
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		record, upsertErr := models.UpsertCanvasDashboardInTransaction(tx, canvas.ID, modelPanels, modelLayout)
		if upsertErr != nil {
			return upsertErr
		}
		saved = record
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to update canvas dashboard")
		return nil, status.Error(codes.Internal, "failed to update canvas dashboard")
	}

	serialized, err := serializeCanvasDashboard(saved)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas dashboard")
	}

	return &pb.UpdateCanvasDashboardResponse{Dashboard: serialized}, nil
}

func validateDashboardInput(panels []models.DashboardPanel, layout []models.DashboardLayoutItem) error {
	if err := models.ValidateDashboardContent(panels, layout); err != nil {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return nil
}
