package canvases

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var errTemplateCanvasAutoLayout = errors.New("template canvas auto layout failed")

type templateCanvasAutoLayoutError struct {
	cause error
}

func (e *templateCanvasAutoLayoutError) Error() string {
	return e.cause.Error()
}

func (e *templateCanvasAutoLayoutError) Unwrap() []error {
	return []error{errTemplateCanvasAutoLayout, e.cause}
}

// CreatePublishedTemplateCanvasWithoutSetupInTransaction persists a shared template
// as data only. It never runs runtime node setup or caller-org-specific behavior.
func CreatePublishedTemplateCanvasWithoutSetupInTransaction(
	tx *gorm.DB,
	registry *registry.Registry,
	template *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	createdBy *uuid.UUID,
) (*models.Canvas, error) {
	organizationID := models.TemplateOrganizationID.String()

	nodes, edges, err := ParseCanvas(registry, organizationID, template)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, &templateCanvasAutoLayoutError{cause: err}
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: models.TemplateOrganizationID,
		LiveVersionID:  ptrUUID(uuid.New()),
		IsTemplate:     true,
		Name:           template.Metadata.Name,
		Description:    template.Metadata.Description,
		CreatedBy:      createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	if err := tx.Create(canvas).Error; err != nil {
		return nil, err
	}

	if err := persistCanvasNodesWithoutSetupInTransaction(tx, canvas.ID, expandedNodes, &now); err != nil {
		return nil, err
	}

	version, err := models.CreatePublishedCanvasVersionInTransaction(
		tx,
		canvas.ID,
		createdBy,
		expandedNodes,
		edges,
	)
	if err != nil {
		return nil, err
	}

	canvas.LiveVersionID = &version.ID
	canvas.UpdatedAt = version.UpdatedAt

	return canvas, nil
}

func persistCanvasNodesWithoutSetupInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	nodes []models.Node,
	now *time.Time,
) error {
	for _, node := range nodes {
		var parentNodeID *string
		if idx := strings.Index(node.ID, ":"); idx != -1 {
			parent := node.ID[:idx]
			parentNodeID = &parent
		}

		canvasNode := models.CanvasNode{
			WorkflowID:    canvasID,
			NodeID:        node.ID,
			ParentNodeID:  parentNodeID,
			Name:          node.Name,
			State:         models.CanvasNodeStateReady,
			Type:          node.Type,
			Ref:           datatypes.NewJSONType(node.Ref),
			Configuration: datatypes.NewJSONType(node.Configuration),
			Metadata:      datatypes.NewJSONType(node.Metadata),
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := tx.Create(&canvasNode).Error; err != nil {
			return err
		}
	}

	return nil
}
