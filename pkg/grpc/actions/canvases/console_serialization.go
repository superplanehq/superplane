package canvases

import (
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func serializeConsolePanels(version *models.CanvasVersion) []*pb.Console_Panel {
	if version == nil {
		return nil
	}

	panels := version.ConsolePanels.Data()
	if panels == nil {
		panels = []models.ConsolePanel{}
	}

	protoPanels := make([]*pb.Console_Panel, 0, len(panels))
	for _, panel := range panels {
		content, err := newStructpbValue(panel.Content)
		if err != nil {
			content = nil
		}

		protoPanels = append(protoPanels, &pb.Console_Panel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	return protoPanels
}

func serializeConsoleLayout(version *models.CanvasVersion) []*pb.Console_LayoutItem {
	if version == nil {
		return nil
	}

	layout := version.ConsoleLayout.Data()
	if layout == nil {
		layout = []models.ConsoleLayoutItem{}
	}

	protoLayout := make([]*pb.Console_LayoutItem, 0, len(layout))
	for _, item := range layout {
		layoutItem := &pb.Console_LayoutItem{
			I: item.I,
			X: int32(item.X),
			Y: int32(item.Y),
			W: int32(item.W),
			H: int32(item.H),
		}
		if item.MinW != nil {
			minW := int32(*item.MinW)
			layoutItem.MinW = &minW
		}
		if item.MinH != nil {
			minH := int32(*item.MinH)
			layoutItem.MinH = &minH
		}
		protoLayout = append(protoLayout, layoutItem)
	}

	return protoLayout
}

func serializeCanvasSpecFromVersion(version *models.CanvasVersion) *pb.Canvas_Spec {
	if version == nil {
		return nil
	}

	return &pb.Canvas_Spec{
		Nodes:  actions.NodesToProto(version.Nodes),
		Edges:  actions.EdgesToProto(version.Edges),
		Panels: serializeConsolePanels(version),
		Layout: serializeConsoleLayout(version),
	}
}
