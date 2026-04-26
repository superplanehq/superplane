package canvases

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvases(canvases []models.Canvas) ([]*pb.Canvas, error) {
	//
	// Get all users with a single query, to avoid N+1 queries.
	//
	userIDs := []uuid.UUID{}
	for _, canvas := range canvases {
		if canvas.CreatedBy != nil {
			userIDs = append(userIDs, *canvas.CreatedBy)
		}
	}

	users, err := models.FindMaybeDeletedUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[string]models.User, len(users))
	for _, user := range users {
		usersByID[user.ID.String()] = user
	}

	//
	// Serialize all canvases now
	//
	protoCanvases := make([]*pb.Canvas, len(canvases))
	for i, canvas := range canvases {
		var user *models.User
		if canvas.CreatedBy != nil {
			u, _ := usersByID[canvas.CreatedBy.String()]
			user = &u
		}

		protoCanvas, err := SerializeCanvas(&canvas, false, user)
		if err != nil {
			return nil, err
		}

		protoCanvases[i] = protoCanvas
	}

	return protoCanvases, nil
}

func SerializeCanvas(canvas *models.Canvas, includeStatus bool, user *models.User) (*pb.Canvas, error) {
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
	if err != nil {
		return nil, err
	}

	changeManagementEnabled, err := isChangeManagementEnabledForCanvas(canvas)
	if err != nil {
		return nil, err
	}

	serializedNodes, err := serializeCanvasNodes(canvas.ID, liveVersion.Nodes)
	if err != nil {
		return nil, err
	}

	var createdBy *pb.UserRef
	if user != nil {
		createdBy = &pb.UserRef{Id: user.ID.String(), Name: user.Name}
	}

	if !includeStatus {
		return &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:             canvas.ID.String(),
				OrganizationId: canvas.OrganizationID.String(),
				Name:           canvas.Name,
				Description:    canvas.Description,
				CreatedAt:      timestamppb.New(*canvas.CreatedAt),
				UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
				CreatedBy:      createdBy,
				IsTemplate:     canvas.IsTemplate,
			},
			Spec: &pb.Canvas_Spec{
				Nodes:            serializedNodes,
				Edges:            actions.EdgesToProto(liveVersion.Edges),
				ChangeManagement: serializeChangeManagement(changeManagementEnabled, canvas.EffectiveChangeRequestApprovers()),
			},
			Status: nil,
		}, nil
	}

	// Fetch last executions per node
	lastExecutions, err := models.FindLastExecutionPerNode(canvas.ID)
	if err != nil {
		return nil, err
	}

	executionIDs := make([]string, len(lastExecutions))
	for i, execution := range lastExecutions {
		executionIDs[i] = execution.ID.String()
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs)
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(lastExecutions, childExecutions)
	if err != nil {
		return nil, err
	}

	// Fetch last events per node
	lastEvents, err := models.FindLastEventPerNode(canvas.ID)
	if err != nil {
		return nil, err
	}

	serializedEvents, err := SerializeCanvasEvents(lastEvents)
	if err != nil {
		return nil, err
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
			IsTemplate:     canvas.IsTemplate,
		},
		Spec: &pb.Canvas_Spec{
			Nodes:            serializedNodes,
			Edges:            actions.EdgesToProto(liveVersion.Edges),
			ChangeManagement: serializeChangeManagement(changeManagementEnabled, canvas.EffectiveChangeRequestApprovers()),
		},
		Status: &pb.Canvas_Status{
			LastExecutions: serializedExecutions,
			LastEvents:     serializedEvents,
		},
	}, nil
}

func serializeChangeManagement(
	enabled bool,
	approvers []models.CanvasChangeRequestApprover,
) *pb.Canvas_ChangeManagement {
	cm := &pb.Canvas_ChangeManagement{
		Enabled:   enabled,
		Approvals: make([]*pb.Canvas_ChangeManagement_Approver, 0, len(approvers)),
	}

	for _, approver := range approvers {
		cm.Approvals = append(cm.Approvals, &pb.Canvas_ChangeManagement_Approver{
			Type:     canvasChangeRequestApproverTypeToProto(approver.Type),
			UserId:   approver.User,
			RoleName: approver.Role,
		})
	}

	return cm
}

func canvasChangeRequestApproverTypeToProto(value string) pb.Canvas_ChangeManagement_Approver_Type {
	switch value {
	case models.CanvasChangeRequestApproverTypeAnyone:
		return pb.Canvas_ChangeManagement_Approver_TYPE_ANYONE
	case models.CanvasChangeRequestApproverTypeUser:
		return pb.Canvas_ChangeManagement_Approver_TYPE_USER
	case models.CanvasChangeRequestApproverTypeRole:
		return pb.Canvas_ChangeManagement_Approver_TYPE_ROLE
	default:
		return pb.Canvas_ChangeManagement_Approver_TYPE_UNSPECIFIED
	}
}

func serializeCanvasNodes(canvasID uuid.UUID, nodes []models.Node) ([]*componentpb.Node, error) {
	serialized := actions.NodesToProto(nodes)
	if len(serialized) == 0 {
		return serialized, nil
	}

	canvasNodes, err := models.FindCanvasNodes(canvasID)
	if err != nil {
		return nil, err
	}

	pausedByID := make(map[string]bool, len(canvasNodes))
	for _, node := range canvasNodes {
		pausedByID[node.NodeID] = node.State == models.CanvasNodeStatePaused
	}

	for _, node := range serialized {
		if paused, ok := pausedByID[node.Id]; ok {
			node.Paused = paused
		}
	}

	return serialized, nil
}

func ParseCanvas(registry *registry.Registry, orgID string, canvas *pb.Canvas) ([]models.Node, []models.Edge, error) {
	if canvas.Metadata == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas metadata is required")
	}

	if canvas.Metadata.Name == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	if canvas.Spec == nil {
		return nil, nil, status.Error(codes.InvalidArgument, "canvas spec is required")
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
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %d: id is required", i)
		}

		if node.Name == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %s: name is required", node.Id)
		}

		if nodeIDs[node.Id] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "node %s: duplicate node id", node.Id)
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

	for i, edge := range canvas.Spec.Edges {
		if edge.SourceId == "" || edge.TargetId == "" {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source_id and target_id are required", i)
		}

		if !nodeIDs[edge.SourceId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: source node %s not found", i, edge.SourceId)
		}

		if !nodeIDs[edge.TargetId] {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: target node %s not found", i, edge.TargetId)
		}

		if nodeTypeByID[edge.SourceId] == componentpb.Node_TYPE_WIDGET {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as source nodes", i)
		}

		if nodeTypeByID[edge.TargetId] == componentpb.Node_TYPE_WIDGET {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: widget nodes cannot be used as target nodes", i)
		}

		if err := changesets.ValidateSourceNodeOutputChannel(
			registry,
			nodesByID[edge.SourceId],
			edge.Channel,
		); err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "edge %d: %v", i, err)
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
		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
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
		err := validateIntegration(organizationID, node.Integration)
		if err != nil {
			return err
		}
	}

	return configuration.ValidateConfiguration(configurable.Configuration(), node.Configuration.AsMap())
}

func validateIntegration(organizationID string, ref *componentpb.IntegrationRef) error {
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

	_, err = models.FindIntegration(orgID, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found or does not belong to this organization")
	}

	return nil
}
