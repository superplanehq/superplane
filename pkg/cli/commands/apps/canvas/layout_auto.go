package canvas

import (
	"encoding/json"
	"fmt"

	canvasmodels "github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	canvaseslayout "github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func applyAutoLayoutToCanvasResource(resource *canvasmodels.Canvas, autoLayout *openapi_client.CanvasesCanvasAutoLayout) error {
	if autoLayout == nil || resource == nil || resource.Spec == nil {
		return nil
	}

	nodes, edges, err := openapiSpecToModelGraph(resource.Spec)
	if err != nil {
		return err
	}

	layoutedNodes, layoutedEdges, err := canvaseslayout.ApplyLayout(nodes, edges, openapiAutoLayoutToProto(autoLayout))
	if err != nil {
		return err
	}

	payload, err := json.Marshal(struct {
		Nodes []models.Node `json:"nodes"`
		Edges []models.Edge `json:"edges"`
	}{
		Nodes: layoutedNodes,
		Edges: layoutedEdges,
	})
	if err != nil {
		return fmt.Errorf("marshal layouted spec: %w", err)
	}

	var decoded struct {
		Nodes []openapi_client.SuperplaneComponentsNode `json:"nodes"`
		Edges []openapi_client.SuperplaneComponentsEdge `json:"edges"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return fmt.Errorf("decode layouted spec: %w", err)
	}
	resource.Spec.SetNodes(decoded.Nodes)
	resource.Spec.SetEdges(decoded.Edges)
	return nil
}

func openapiSpecToModelGraph(spec *openapi_client.CanvasesCanvasSpec) ([]models.Node, []models.Edge, error) {
	if spec == nil {
		return []models.Node{}, []models.Edge{}, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal canvas spec: %w", err)
	}

	var decoded struct {
		Nodes []models.Node `json:"nodes"`
		Edges []models.Edge `json:"edges"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, nil, fmt.Errorf("decode canvas spec: %w", err)
	}
	return decoded.Nodes, decoded.Edges, nil
}

func openapiAutoLayoutToProto(autoLayout *openapi_client.CanvasesCanvasAutoLayout) *pb.CanvasAutoLayout {
	if autoLayout == nil {
		return nil
	}

	result := &pb.CanvasAutoLayout{}
	switch autoLayout.GetAlgorithm() {
	case openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL:
		result.Algorithm = pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL
	default:
		result.Algorithm = pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED
	}

	switch autoLayout.GetScope() {
	case openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS:
		result.Scope = pb.CanvasAutoLayout_SCOPE_FULL_CANVAS
	case openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT:
		result.Scope = pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT
	default:
		result.Scope = pb.CanvasAutoLayout_SCOPE_UNSPECIFIED
	}

	result.NodeIds = append([]string(nil), autoLayout.GetNodeIds()...)
	return result
}
