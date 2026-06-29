package canvases

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
)

func SerializeCanvas(
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (*pb.Canvas, error) {
	var createdBy *pb.UserRef
	if user != nil {
		createdBy = &pb.UserRef{Id: user.ID.String(), Name: user.Name}
	}

	canvasFolderID := ""
	if canvas.CanvasFolderID != nil {
		canvasFolderID = canvas.CanvasFolderID.String()
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:             canvas.ID.String(),
			OrganizationId: canvas.OrganizationID.String(),
			Name:           canvas.Name,
			Description:    canvas.Description,
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:      createdBy,
			FolderId:       canvasFolderID,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: actions.NodesToProto(liveVersion.Nodes),
			Edges: actions.EdgesToProto(liveVersion.Edges),
		},
		Status: status,
	}, nil
}

func ParseCanvas(registry *registry.Registry, orgID string, canvas *pb.Canvas) ([]models.Node, []models.Edge, error) {
	if canvas.Metadata == nil {
		return nil, nil, grpcerrors.InvalidArgument(nil, "canvas metadata is required")
	}

	if canvas.Metadata.Name == "" {
		return nil, nil, grpcerrors.InvalidArgument(nil, "canvas name is required")
	}

	if canvas.Spec == nil {
		return nil, nil, grpcerrors.InvalidArgument(nil, "canvas spec is required")
	}

	// Allow empty canvases
	if len(canvas.Spec.Nodes) == 0 {
		return []models.Node{}, []models.Edge{}, nil
	}

	nodeIDs := make(map[string]bool)
	nodeTypeByID := make(map[string]componentpb.Node_Type)
	nodeValidationErrors := make(map[string]string)

	for i, node := range canvas.Spec.Nodes {
		if node.Id == "" {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("node %d: id is required", i))
		}

		if node.Name == "" {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("node %s: name is required", node.Id))
		}

		if nodeIDs[node.Id] {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("node %s: duplicate node id", node.Id))
		}

		nodeIDs[node.Id] = true
		nodeTypeByID[node.Id] = node.Type
		if err := validateNodeRef(registry, orgID, node); err != nil {
			nodeValidationErrors[node.Id] = err.Error()
		}
	}

	// Find shadowed names within connected components
	nodes := actions.ProtoToNodes(canvas.Spec.Nodes)
	nodeWarnings := actions.FindShadowedNameWarnings(registry, canvas.Spec.Nodes, canvas.Spec.Edges)
	nodesByID := make(map[string]models.Node, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	for nodeID, errs := range expressionvalidation.ValidateCanvasExpressions(registry, canvas.Spec.Nodes) {
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		joined := strings.Join(msgs, "\n")
		if existing, ok := nodeValidationErrors[nodeID]; ok {
			nodeValidationErrors[nodeID] = existing + "\n" + joined
		} else {
			nodeValidationErrors[nodeID] = joined
		}
	}

	for i, edge := range canvas.Spec.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: source_id and target_id are required", i))
		}

		if edge.Channel == "" {
			edge.Channel = "default"
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: source node %s not found", i, edge.SourceId))
		}

		if !nodeIDs[edge.TargetId] {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: target node %s not found", i, edge.TargetId))
		}

		if nodeTypeByID[edge.SourceId] == componentpb.Node_TYPE_WIDGET {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: widget nodes cannot be used as source nodes", i))
		}

		if nodeTypeByID[edge.TargetId] == componentpb.Node_TYPE_WIDGET {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: widget nodes cannot be used as target nodes", i))
		}

		if err := changesets.ValidateSourceNodeOutputChannel(
			registry,
			nodesByID[edge.SourceId],
			edge.Channel,
		); err != nil {
			return nil, nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("edge %d: %v", i, err))
		}
	}

	// Convert proto nodes to models, adding validation errors and warnings where applicable
	edges := actions.ProtoToEdges(canvas.Spec.Edges)
	for i := range nodes {
		if errorMsg, hasError := nodeValidationErrors[nodes[i].ID]; hasError {
			nodes[i].ErrorMessage = &errorMsg
		} else {
			nodes[i].ErrorMessage = nil
		}

		if warningMsg, hasWarning := nodeWarnings[nodes[i].ID]; hasWarning {
			nodes[i].WarningMessage = &warningMsg
		} else {
			nodes[i].WarningMessage = nil
		}
	}

	//
	// Check for cycles in the canvas
	//
	if err := changesets.CheckForCycles(nodes, edges); err != nil {
		return nil, nil, grpcerrors.InvalidArgument(err, "invalid canvas graph")
	}

	return nodes, edges, nil
}

func validateNodeRef(registry *registry.Registry, organizationID string, node *componentpb.Node) error {
	if node.Component == "" {
		return fmt.Errorf("component name is required")
	}

	parts := strings.SplitN(node.Component, ".", 2)
	if len(parts) > 2 {
		return fmt.Errorf("invalid component name: %s", node.Component)
	}

	configurable, err := registry.FindConfigurableComponent(node.Component)
	if err != nil {
		return err
	}

	if len(parts) > 1 {
		err := validateIntegration(organizationID, node.Integration, node.Component)
		if err != nil {
			return err
		}
	}

	return configuration.ValidateConfiguration(configurable.Configuration(), node.Configuration.AsMap())
}

func validateIntegration(organizationID string, ref *componentpb.IntegrationRef, component string) error {
	if ref == nil || ref.Id == nil {
		return fmt.Errorf("integration is required")
	}

	integrationID, err := uuid.Parse(*ref.Id)
	if err != nil {
		return fmt.Errorf("invalid integration ID: %v", err)
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %v", err)
	}

	integration, err := models.FindIntegration(orgID, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found or does not belong to this organization")
	}

	if !integration.HasCapabilityEnabled(component) {
		return fmt.Errorf("%s is not enabled for integration %s", component, integration.InstallationName)
	}

	return nil
}

func serializeCanvas(
	ctx context.Context,
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (*pb.Canvas, error) {
	var proto *pb.Canvas
	err := telemetry.RunSpan(ctx, "canvases.serialize", func(ctx context.Context) error {
		var serErr error
		proto, serErr = SerializeCanvas(canvas, liveVersion, user, status)
		return serErr
	})
	if err != nil {
		return nil, err
	}

	return proto, nil
}
