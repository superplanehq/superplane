package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	return CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		nil,
		nil,
		registry,
		organizationID,
		pbCanvas,
		autoLayout,
		"",
		nil,
	)
}

func CreateCanvasWithAutoLayoutAndUsage(
	ctx context.Context,
	usageService usage.Service,
	registry *registry.Registry,
	organizationID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.CreateCanvasResponse, error) {
	return CreateCanvasWithAutoLayoutAndUsageAndSetup(
		ctx,
		usageService,
		nil,
		registry,
		organizationID,
		pbCanvas,
		autoLayout,
		"",
		nil,
	)
}

func CreateCanvasWithAutoLayoutAndUsageAndSetup(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CreateCanvasResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	createdBy := uuid.MustParse(userID)
	var err error
	isTemplate := pbCanvas.Metadata.GetIsTemplate()
	if isTemplate {
		var canvas *models.Canvas
		err = database.Conn().Transaction(func(tx *gorm.DB) error {
			canvas, err = CreatePublishedTemplateCanvasWithoutSetupInTransaction(
				tx,
				registry,
				pbCanvas,
				autoLayout,
				&createdBy,
			)
			if err != nil {
				if strings.Contains(err.Error(), ErrDuplicateCanvasName) {
					return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
				}
				if errors.Is(err, errTemplateCanvasAutoLayout) {
					return status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", err)
				}
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return createCanvasResponse(canvas, organizationID)
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", err)
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	targetOrganizationID := uuid.MustParse(organizationID)
	changeManagementEnabled := false
	changeManagementEnabled, err = models.IsChangeManagementEnabled(targetOrganizationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load organization change management setting: %v", err)
	}

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
	canvas := &models.Canvas{
		ID:                      uuid.New(),
		OrganizationID:          targetOrganizationID,
		LiveVersionID:           ptrUUID(uuid.New()),
		IsTemplate:              false,
		ChangeManagementEnabled: changeManagementEnabled,
		Name:                    pbCanvas.Metadata.Name,
		Description:             pbCanvas.Metadata.Description,
		CreatedBy:               &createdBy,
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	organizationUUID := uuid.MustParse(organizationID)
	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Create the workflow record
		//
		err := tx.Clauses(clause.Returning{}).Create(canvas).Error
		if err != nil {
			if strings.Contains(err.Error(), ErrDuplicateCanvasName) {
				return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
			}
			return err
		}

		//
		// Create the workflow node records (including internal blueprint nodes)
		//
		if err := createCanvasNodesInTransaction(
			ctx,
			tx,
			encryptor,
			registry,
			organizationUUID,
			canvas.ID,
			expandedNodes,
			authService,
			webhookBaseURL,
		); err != nil {
			return err
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
		canvas.UpdatedAt = version.UpdatedAt

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createCanvasResponse(canvas, organizationID)
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func createCanvasResponse(canvas *models.Canvas, creatorOrganizationID string) (*pb.CreateCanvasResponse, error) {
	if publishErr := messages.NewCanvasCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishCreated(); publishErr != nil {
		log.Errorf("failed to publish canvas created RabbitMQ message: %v", publishErr)
	}

	userOrganizationID := canvas.OrganizationID.String()
	if canvas.IsTemplate {
		userOrganizationID = creatorOrganizationID
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		var err error
		user, err = models.FindMaybeDeletedUserByID(userOrganizationID, canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	proto, err := SerializeCanvas(canvas, false, user)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}

func createCanvasNodesInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	nodes []models.Node,
	authService authorization.Authorization,
	webhookBaseURL string,
) error {
	existingNodes := []models.CanvasNode{}
	nodesByID := make(map[string]*models.Node, len(nodes))
	for i := range nodes {
		nodesByID[nodes[i].ID] = &nodes[i]
	}

	canSetupNodes := encryptor != nil && authService != nil

	for _, node := range nodes {
		canvasNode, nodeLevelErrorMessage, err := upsertNode(tx, existingNodes, node, canvasID)
		if err != nil {
			return err
		}

		if nodeLevelErrorMessage != nil {
			setParentNodeError(canvasNode, node.ID, nodesByID, nodeLevelErrorMessage)
		}
		syncCreatedCanvasNode(nodesByID, canvasNode)

		if !canSetupNodes || canvasNode.State != models.CanvasNodeStateReady {
			continue
		}

		if err := setupNode(ctx, tx, encryptor, reg, canvasNode, organizationUUID, authService, webhookBaseURL); err != nil {
			if saveErr := markNodeSetupError(tx, canvasNode, err); saveErr != nil {
				return saveErr
			}

			errorMsg := err.Error()
			setParentNodeError(canvasNode, node.ID, nodesByID, &errorMsg)
		}

		syncCreatedCanvasNode(nodesByID, canvasNode)
	}

	return nil
}

func syncCreatedCanvasNode(nodesByID map[string]*models.Node, canvasNode *models.CanvasNode) {
	node, ok := nodesByID[canvasNode.NodeID]
	if !ok {
		return
	}

	node.Metadata = canvasNode.Metadata.Data()
	if canvasNode.StateReason == nil || strings.TrimSpace(*canvasNode.StateReason) == "" {
		node.ErrorMessage = nil
		return
	}

	errorMsg := *canvasNode.StateReason
	node.ErrorMessage = &errorMsg
}
