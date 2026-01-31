package canvases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
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

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
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

	workflow := models.Workflow{
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
		err := tx.Clauses(clause.Returning{}).Create(&workflow).Error
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

			workflowNode := models.WorkflowNode{
				WorkflowID:    workflow.ID,
				NodeID:        node.ID,
				ParentNodeID:  parentNodeID,
				Name:          node.Name,
				State:         models.WorkflowNodeStateReady,
				Type:          node.Type,
				Ref:           datatypes.NewJSONType(node.Ref),
				Configuration: datatypes.NewJSONType(node.Configuration),
				Metadata:      datatypes.NewJSONType(node.Metadata),
				CreatedAt:     &now,
				UpdatedAt:     &now,
			}

			if err := tx.Create(&workflowNode).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	proto, err := SerializeCanvas(&workflow, false)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
