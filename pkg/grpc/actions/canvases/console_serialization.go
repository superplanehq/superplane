package canvases

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ConsolePanelsToProto(panels []models.ConsolePanel) ([]*pb.Console_Panel, error) {
	if len(panels) == 0 {
		return nil, nil
	}

	result := make([]*pb.Console_Panel, 0, len(panels))
	for _, panel := range panels {
		content, err := newStructpbValue(panel.Content)
		if err != nil {
			return nil, fmt.Errorf("panel %q: %w", panel.ID, err)
		}

		result = append(result, &pb.Console_Panel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	return result, nil
}

func ConsoleLayoutToProto(layout []models.ConsoleLayoutItem) []*pb.Console_LayoutItem {
	if len(layout) == 0 {
		return nil
	}

	result := make([]*pb.Console_LayoutItem, 0, len(layout))
	for _, item := range layout {
		protoItem := &pb.Console_LayoutItem{
			I: item.I,
			X: int32(item.X),
			Y: int32(item.Y),
			W: int32(item.W),
			H: int32(item.H),
		}
		if item.MinW != nil {
			minW := int32(*item.MinW)
			protoItem.MinW = &minW
		}
		if item.MinH != nil {
			minH := int32(*item.MinH)
			protoItem.MinH = &minH
		}
		result = append(result, protoItem)
	}

	return result
}

func SerializeCanvasSpecFromVersion(version *models.CanvasVersion) (*pb.Canvas_Spec, error) {
	if version == nil {
		return &pb.Canvas_Spec{}, nil
	}

	panels := version.ConsolePanels.Data()
	if panels == nil {
		panels = []models.ConsolePanel{}
	}
	layout := version.ConsoleLayout.Data()
	if layout == nil {
		layout = []models.ConsoleLayoutItem{}
	}

	protoPanels, err := ConsolePanelsToProto(panels)
	if err != nil {
		return nil, err
	}

	return &pb.Canvas_Spec{
		Nodes:  actions.NodesToProto(version.Nodes),
		Edges:  actions.EdgesToProto(version.Edges),
		Panels: protoPanels,
		Layout: ConsoleLayoutToProto(layout),
	}, nil
}
