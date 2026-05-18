package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pbApps "github.com/superplanehq/superplane/pkg/protos/apps"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateAppDashboard(
	ctx context.Context,
	organizationID, appID string,
	panels []*pbCanvases.DashboardPanel,
	layout []*pbCanvases.DashboardLayoutItem,
) (*pbApps.UpdateAppDashboardResponse, error) {
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
		return nil, status.Error(codes.FailedPrecondition, "app has no associated canvas")
	}

	modelPanels, deserializeErr := deserializeDashboardPanels(panels)
	if deserializeErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", deserializeErr)
	}
	modelLayout := deserializeDashboardLayout(layout)

	var saved *models.CanvasDashboard
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		record, upsertErr := models.UpsertCanvasDashboardInTransaction(tx, *app.CanvasID, modelPanels, modelLayout)
		if upsertErr != nil {
			return upsertErr
		}
		saved = record
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to update app dashboard")
		return nil, status.Error(codes.Internal, "failed to update app dashboard")
	}

	// TODO(phase-2): Commit dashboard/dashboard.yaml to Code Storage.

	serialized, err := serializeDashboard(saved)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize app dashboard")
	}

	return &pbApps.UpdateAppDashboardResponse{Dashboard: serialized}, nil
}

func deserializeDashboardPanels(in []*pbCanvases.DashboardPanel) ([]models.DashboardPanel, error) {
	out := make([]models.DashboardPanel, 0, len(in))
	for _, panel := range in {
		content := map[string]any{}
		if panel.GetContent() != nil {
			raw := panel.GetContent().AsInterface()
			if raw != nil {
				asMap, ok := raw.(map[string]any)
				if !ok {
					return nil, errors.New("panel content must be an object")
				}
				content = asMap
			}
		}
		out = append(out, models.DashboardPanel{
			ID:      panel.GetId(),
			Type:    panel.GetType(),
			Content: content,
		})
	}
	return out, nil
}

func deserializeDashboardLayout(in []*pbCanvases.DashboardLayoutItem) []models.DashboardLayoutItem {
	out := make([]models.DashboardLayoutItem, 0, len(in))
	for _, item := range in {
		converted := models.DashboardLayoutItem{
			I: item.GetI(),
			X: int(item.GetX()),
			Y: int(item.GetY()),
			W: int(item.GetW()),
			H: int(item.GetH()),
		}
		if item.MinW != nil {
			v := int(*item.MinW)
			converted.MinW = &v
		}
		if item.MinH != nil {
			v := int(*item.MinH)
			converted.MinH = &v
		}
		out = append(out, converted)
	}
	return out
}
