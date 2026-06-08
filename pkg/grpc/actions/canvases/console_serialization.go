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

// panelTypeFromModel maps the storage/YAML-level lowercase panel type string
// (the source of truth defined in `pkg/models/console_yml.go`) to its proto
// enum representation. Used only at the wire boundary; the rest of the
// backend continues to work with the string form.
//
// Returns an error when the stored string is not a known panel type. That
// case shouldn't happen in practice because `ValidateConsoleContent` rejects
// unknown types on import, but we surface the mismatch rather than silently
// shipping `TYPE_UNSPECIFIED`.
func panelTypeFromModel(modelType string) (pb.Console_Panel_Type, error) {
	switch modelType {
	case models.ConsolePanelTypeMarkdown:
		return pb.Console_Panel_MARKDOWN, nil
	case models.ConsolePanelTypeNode:
		return pb.Console_Panel_NODE, nil
	case models.ConsolePanelTypeNodes:
		return pb.Console_Panel_NODES, nil
	case models.ConsolePanelTypeTable:
		return pb.Console_Panel_TABLE, nil
	case models.ConsolePanelTypeChart:
		return pb.Console_Panel_CHART, nil
	case models.ConsolePanelTypeNumber:
		return pb.Console_Panel_NUMBER, nil
	default:
		return pb.Console_Panel_TYPE_UNSPECIFIED, fmt.Errorf("unknown panel type %q", modelType)
	}
}

// panelTypeToModel is the inverse of `panelTypeFromModel`. It is fail-closed:
// `TYPE_UNSPECIFIED` and any unknown enum value (including future values an
// older server might receive from a newer client) return an error so the
// caller can reject the request with `InvalidArgument` instead of silently
// dropping the panel into an unvalidated state.
func panelTypeToModel(protoType pb.Console_Panel_Type) (string, error) {
	switch protoType {
	case pb.Console_Panel_MARKDOWN:
		return models.ConsolePanelTypeMarkdown, nil
	case pb.Console_Panel_NODE:
		return models.ConsolePanelTypeNode, nil
	case pb.Console_Panel_NODES:
		return models.ConsolePanelTypeNodes, nil
	case pb.Console_Panel_TABLE:
		return models.ConsolePanelTypeTable, nil
	case pb.Console_Panel_CHART:
		return models.ConsolePanelTypeChart, nil
	case pb.Console_Panel_NUMBER:
		return models.ConsolePanelTypeNumber, nil
	case pb.Console_Panel_TYPE_UNSPECIFIED:
		return "", fmt.Errorf("panel type is required")
	default:
		return "", fmt.Errorf("unsupported panel type %d", protoType)
	}
}

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
		panelType, err := panelTypeFromModel(panel.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid type for panel %q: %w", panel.ID, err)
		}
		pbPanels = append(pbPanels, &pb.Console_Panel{
			Id:      panel.ID,
			Type:    panelType,
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
		panelType, err := panelTypeToModel(panel.GetType())
		if err != nil {
			return nil, fmt.Errorf("panel %q: %w", panel.GetId(), err)
		}
		out = append(out, models.ConsolePanel{
			ID:      panel.GetId(),
			Type:    panelType,
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
