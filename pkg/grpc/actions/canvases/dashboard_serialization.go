package canvases

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const MaxDashboardPanels = 50
const MaxDashboardPayloadBytes = 1024 * 1024

func serializeCanvasDashboard(dashboard *models.CanvasDashboard) (*pb.CanvasDashboard, error) {
	panels := dashboard.Panels.Data()
	layout := dashboard.Layout.Data()

	pbPanels := make([]*pb.DashboardPanel, 0, len(panels))
	for _, panel := range panels {
		var content *structpb.Value
		if panel.Content != nil {
			value, err := structpb.NewValue(toStructpbCompatible(panel.Content))
			if err != nil {
				return nil, fmt.Errorf("invalid content for panel %q: %w", panel.ID, err)
			}
			content = value
		}
		pbPanels = append(pbPanels, &pb.DashboardPanel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	pbLayout := make([]*pb.DashboardLayoutItem, 0, len(layout))
	for _, item := range layout {
		converted := &pb.DashboardLayoutItem{
			I: item.I,
			X: int32(item.X),
			Y: int32(item.Y),
			W: int32(item.W),
			H: int32(item.H),
		}
		if item.MinW != nil {
			minW := int32(*item.MinW)
			converted.MinW = &minW
		}
		if item.MinH != nil {
			minH := int32(*item.MinH)
			converted.MinH = &minH
		}
		pbLayout = append(pbLayout, converted)
	}

	resp := &pb.CanvasDashboard{
		CanvasId: dashboard.CanvasID.String(),
		Panels:   pbPanels,
		Layout:   pbLayout,
	}
	if !dashboard.UpdatedAt.IsZero() {
		resp.UpdatedAt = timestamppb.New(dashboard.UpdatedAt)
	}
	return resp, nil
}

func deserializeDashboardPanels(in []*pb.DashboardPanel) ([]models.DashboardPanel, error) {
	out := make([]models.DashboardPanel, 0, len(in))
	for _, panel := range in {
		var content map[string]any
		if panel.GetContent() != nil {
			raw := panel.GetContent().AsInterface()
			if raw != nil {
				asMap, ok := raw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("panel %q content must be an object", panel.GetId())
				}
				content = asMap
			}
		}
		if content == nil {
			content = map[string]any{}
		}
		out = append(out, models.DashboardPanel{
			ID:      panel.GetId(),
			Type:    panel.GetType(),
			Content: content,
		})
	}
	return out, nil
}

func deserializeDashboardLayout(in []*pb.DashboardLayoutItem) []models.DashboardLayoutItem {
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
			minW := int(*item.MinW)
			converted.MinW = &minW
		}
		if item.MinH != nil {
			minH := int(*item.MinH)
			converted.MinH = &minH
		}
		out = append(out, converted)
	}
	return out
}

func toStructpbCompatible(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = toStructpbCompatible(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = toStructpbCompatible(item)
		}
		return out
	default:
		return v
	}
}

func encodedDashboardPanelsSize(panels []models.DashboardPanel) (int, error) {
	encoded, err := json.Marshal(panels)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}
