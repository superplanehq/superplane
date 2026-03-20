package canvases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ErrDuplicateCanvasName = "duplicate key value violates unique constraint"

func CreateCanvas(ctx context.Context, registry *registry.Registry, organizationID string, pbCanvas *pb.Canvas) (*pb.CreateCanvasResponse, error) {
	return CreateCanvasWithAutoLayout(ctx, registry, organizationID, pbCanvas, nil)
}

func CreateCanvasWithAutoLayout(
	ctx context.Context,
	registry *registry.Registry,
	organizationID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.CreateCanvasResponse, error) {
	return CreateCanvasWithAutoLayoutAndUsage(ctx, nil, registry, organizationID, pbCanvas, autoLayout)
}

func CreateCanvasWithAutoLayoutAndUsage(
	ctx context.Context,
	usageService usage.Service,
	registry *registry.Registry,
	organizationID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.CreateCanvasResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = applyCanvasAutoLayout(nodes, edges, autoLayout, registry)
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
	canvasVersioningEnabled := false
	if !isTemplate {
		canvasVersioningEnabled, err = models.IsCanvasVersioningEnabled(targetOrganizationID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load organization canvas versioning: %v", err)
		}
	}

	if !isTemplate {
		canvasCount, err := models.CountCanvasesByOrganization(organizationID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to count organization canvases: %v", err)
		}

		if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, organizationID, &usagepb.OrganizationState{
			Canvases: int32(canvasCount + 1),
		}, &usagepb.CanvasState{
			Nodes: int32(len(expandedNodes)),
		}); err != nil {
			return nil, err
		}
	}

	liveVersionID := uuid.New()

	canvas := models.Canvas{
		ID:                uuid.New(),
		OrganizationID:    targetOrganizationID,
		LiveVersionID:     &liveVersionID,
		IsTemplate:        isTemplate,
		VersioningEnabled: canvasVersioningEnabled,
		Name:              pbCanvas.Metadata.Name,
		Description:       pbCanvas.Metadata.Description,
		CreatedBy:         &createdBy,
		CreatedAt:         &now,
		UpdatedAt:         &now,
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
				State:         models.CanvasNodeStateReady,
				Type:          node.Type,
				Ref:           datatypes.NewJSONType(node.Ref),
				Configuration: datatypes.NewJSONType(node.Configuration),
				Metadata:      datatypes.NewJSONType(node.Metadata),
				CreatedAt:     &now,
				UpdatedAt:     &now,
			}

			if err := tx.Create(&canvasNode).Error; err != nil {
				return err
			}
		}

		version, err := models.CreatePublishedCanvasVersionInTransaction(
			tx,
			canvas.ID,
			&createdBy,
			expandedNodes,
			edges,
		)
		if err != nil {
			return err
		}
		canvas.LiveVersionID = &version.ID

		return nil
	})

	if err != nil {
		return nil, err
	}

	proto, err := SerializeCanvas(&canvas, false)
	if err != nil {
		return nil, err
	}

	if publishErr := messages.NewCanvasCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishCreated(); publishErr != nil {
		log.Errorf("failed to publish canvas created RabbitMQ message: %v", publishErr)
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
