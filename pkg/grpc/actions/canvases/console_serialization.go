package canvases

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MaxConsolePanels and MaxConsolePayloadBytes are re-exported from the
// models package so existing gRPC tests/callers keep working unchanged.
const (
	MaxConsolePanels       = models.MaxConsolePanels
	MaxConsolePayloadBytes = models.MaxConsolePayloadBytes
)

func serializeConsole(version *models.CanvasVersion) (*pb.Console, error) {
	panels := version.ConsolePanels.Data()
	layout := version.ConsoleLayout.Data()

	pbPanels := make([]*pb.Console_Panel, 0, len(panels))
	for _, panel := range panels {
		var content *structpb.Value
		if panel.Content != nil {
			value, err := newStructpbValue(panel.Content)
			if err != nil {
				return nil, fmt.Errorf("invalid content for panel %q: %w", panel.ID, err)
			}
			content = value
		}
		pbPanels = append(pbPanels, &pb.Console_Panel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	pbLayout := make([]*pb.Console_LayoutItem, 0, len(layout))
	for _, item := range layout {
		converted := &pb.Console_LayoutItem{
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

	resp := &pb.Console{
		CanvasId:  version.WorkflowID.String(),
		VersionId: version.ID.String(),
		Panels:    pbPanels,
		Layout:    pbLayout,
	}

	if version.UpdatedAt != nil && !version.UpdatedAt.IsZero() {
		resp.UpdatedAt = timestamppb.New(*version.UpdatedAt)
	}

	return resp, nil
}

func deserializeConsolePanels(in []*pb.Console_Panel) ([]models.ConsolePanel, error) {
	out := make([]models.ConsolePanel, 0, len(in))
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
		out = append(out, models.ConsolePanel{
			ID:      panel.GetId(),
			Type:    panel.GetType(),
			Content: content,
		})
	}
	return out, nil
}

func deserializeConsoleLayout(in []*pb.Console_LayoutItem) []models.ConsoleLayoutItem {
	out := make([]models.ConsoleLayoutItem, 0, len(in))
	for _, item := range in {
		converted := models.ConsoleLayoutItem{
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
