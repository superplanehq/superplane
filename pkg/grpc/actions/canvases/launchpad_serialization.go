package canvases

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MaxLaunchpadPanels caps the number of panels per canvas to prevent unbounded
// growth of the JSONB document. The grid would not be usable beyond this anyway.
const MaxLaunchpadPanels = 50

// MaxLaunchpadPayloadBytes caps the encoded size of `panels` to keep a single
// upsert from inflating the table or transferring too much data on read.
const MaxLaunchpadPayloadBytes = 1024 * 1024 // 1 MiB

func serializeCanvasLaunchpad(launchpad *models.CanvasLaunchpad) (*pb.CanvasLaunchpad, error) {
	panels := launchpad.Panels.Data()
	layout := launchpad.Layout.Data()

	pbPanels := make([]*pb.LaunchpadPanel, 0, len(panels))
	for _, panel := range panels {
		var content *structpb.Value
		if panel.Content != nil {
			value, err := structpb.NewValue(toStructpbCompatible(panel.Content))
			if err != nil {
				return nil, fmt.Errorf("invalid content for panel %q: %w", panel.ID, err)
			}
			content = value
		}
		pbPanels = append(pbPanels, &pb.LaunchpadPanel{
			Id:      panel.ID,
			Type:    panel.Type,
			Content: content,
		})
	}

	pbLayout := make([]*pb.LaunchpadLayoutItem, 0, len(layout))
	for _, item := range layout {
		converted := &pb.LaunchpadLayoutItem{
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
		if item.AutoHeight != nil {
			autoHeight := *item.AutoHeight
			converted.AutoHeight = &autoHeight
		}
		pbLayout = append(pbLayout, converted)
	}

	resp := &pb.CanvasLaunchpad{
		CanvasId: launchpad.CanvasID.String(),
		Panels:   pbPanels,
		Layout:   pbLayout,
	}
	if !launchpad.UpdatedAt.IsZero() {
		resp.UpdatedAt = timestamppb.New(launchpad.UpdatedAt)
	}
	return resp, nil
}

func deserializeLaunchpadPanels(in []*pb.LaunchpadPanel) ([]models.LaunchpadPanel, error) {
	out := make([]models.LaunchpadPanel, 0, len(in))
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
		out = append(out, models.LaunchpadPanel{
			ID:      panel.GetId(),
			Type:    panel.GetType(),
			Content: content,
		})
	}
	return out, nil
}

func deserializeLaunchpadLayout(in []*pb.LaunchpadLayoutItem) []models.LaunchpadLayoutItem {
	out := make([]models.LaunchpadLayoutItem, 0, len(in))
	for _, item := range in {
		converted := models.LaunchpadLayoutItem{
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
		if item.AutoHeight != nil {
			autoHeight := *item.AutoHeight
			converted.AutoHeight = &autoHeight
		}
		out = append(out, converted)
	}
	return out
}

// toStructpbCompatible normalizes a value so it can be passed to
// structpb.NewValue, which accepts only a small set of Go types. This is
// needed because Content is decoded from JSONB into Go's native types but
// structpb expects float64 for numeric values.
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

func encodedPanelsSize(panels []models.LaunchpadPanel) (int, error) {
	encoded, err := json.Marshal(panels)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}
