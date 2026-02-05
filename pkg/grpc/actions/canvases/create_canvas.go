package canvases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ErrDuplicateCanvasName = "duplicate key value violates unique constraint"

func CreateCanvas(ctx context.Context, registry *registry.Registry, organizationID string, pbCanvas *pb.Canvas) (*pb.CreateCanvasResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if pbCanvas.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas metadata is required")
	}

	if pbCanvas.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	if pbCanvas.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas spec is required")
	}

	if err := actions.CheckForCycles(pbCanvas.Spec.Nodes, pbCanvas.Spec.Edges); err != nil {
		return nil, err
	}

	edges, err := ValidateEdges(pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes := actions.ProtoToNodeDefinitions(pbCanvas.Spec.Nodes)
	nodeValidationErrors := ApplyNodeValidations(registry, organizationID, pbCanvas)
	expandedNodes, err := ExpandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	createdBy := uuid.MustParse(userID)

	now := time.Now()
	targetOrganizationID := uuid.MustParse(organizationID)
	isTemplate := pbCanvas.Metadata.GetIsTemplate()
	if isTemplate {
		targetOrganizationID = models.TemplateOrganizationID
	}

	canvas := models.Canvas{
		ID:             uuid.New(),
		OrganizationID: targetOrganizationID,
		IsTemplate:     isTemplate,
		Name:           pbCanvas.Metadata.Name,
		Description:    pbCanvas.Metadata.Description,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Edges:          datatypes.NewJSONSlice(edges),
		Nodes:          datatypes.NewJSONSlice(expandedNodes),
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Create the workflow record
		//
		err := tx.Clauses(clause.Returning{}).Create(&canvas).Error
		if err != nil {
			if strings.Contains(err.Error(), ErrDuplicateCanvasName) {
				return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
			}
			return err
		}

		//
		// Create the workflow node records (including internal blueprint nodes)
		//
		for _, node := range expandedNodes {
			// Set ParentNodeID for internal nodes (IDs like parent:child)
			var parentNodeID *string
			if idx := strings.Index(node.ID, ":"); idx != -1 {
				parent := node.ID[:idx]
				parentNodeID = &parent
			}

			canvasNode := models.CanvasNode{
				WorkflowID:    canvas.ID,
				NodeID:        node.ID,
				ParentNodeID:  parentNodeID,
				Name:          node.Name,
				Type:          node.Type,
				Ref:           datatypes.NewJSONType(node.Ref),
				Configuration: datatypes.NewJSONType(node.Configuration),
				CreatedAt:     &now,
				UpdatedAt:     &now,
			}

			//
			// If the node has validation errors, set the node to an error state.
			//
			if err, ok := nodeValidationErrors[node.ID]; ok {
				canvasNode.State = models.CanvasNodeStateError
				canvasNode.StateReason = &err
			}

			if err := tx.Create(&canvasNode).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	proto, err := SerializeCanvas(&canvas, false)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
