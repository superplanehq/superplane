package apps

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pbApps "github.com/superplanehq/superplane/pkg/protos/apps"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetAppDashboard(ctx context.Context, organizationID, appID string) (*pbApps.GetAppDashboardResponse, error) {
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
		// Return an empty dashboard if no canvas is associated yet.
		return &pbApps.GetAppDashboardResponse{
			Dashboard: &pbCanvases.CanvasDashboard{
				CanvasId: "",
				Panels:   []*pbCanvases.DashboardPanel{},
				Layout:   []*pbCanvases.DashboardLayoutItem{},
			},
		}, nil
	}

	dashboard, err := models.FindCanvasDashboard(*app.CanvasID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load app dashboard")
	}

	serialized, err := serializeDashboard(dashboard)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize app dashboard")
	}

	return &pbApps.GetAppDashboardResponse{Dashboard: serialized}, nil
}

// serializeDashboard converts a models.CanvasDashboard into the canvases proto message.
func serializeDashboard(d *models.CanvasDashboard) (*pbCanvases.CanvasDashboard, error) {
	panels := d.Panels.Data()
	layout := d.Layout.Data()

	pbPanels := make([]*pbCanvases.DashboardPanel, 0, len(panels))
	for _, panel := range panels {
		var content *structpb.Value
		if panel.Content != nil {
			value, valueErr := structpb.NewValue(panel.Content)
			if valueErr != nil {
				return nil, valueErr
			}
			content = value
		}
		pbPanels = append(pbPanels, &pbCanvases.DashboardPanel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	pbLayout := make([]*pbCanvases.DashboardLayoutItem, 0, len(layout))
	for _, item := range layout {
		converted := &pbCanvases.DashboardLayoutItem{
			I: item.I,
			X: int32(item.X),
			Y: int32(item.Y),
			W: int32(item.W),
			H: int32(item.H),
		}
		if item.MinW != nil {
			v := int32(*item.MinW)
			converted.MinW = &v
		}
		if item.MinH != nil {
			v := int32(*item.MinH)
			converted.MinH = &v
		}
		pbLayout = append(pbLayout, converted)
	}

	resp := &pbCanvases.CanvasDashboard{
		CanvasId: d.CanvasID.String(),
		Panels:   pbPanels,
		Layout:   pbLayout,
	}
	if !d.UpdatedAt.IsZero() {
		resp.UpdatedAt = timestamppb.New(d.UpdatedAt)
	}
	return resp, nil
}
