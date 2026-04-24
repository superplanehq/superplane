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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.UpdateCanvasVersionResponse, error) {
	return UpdateCanvasVersionWithUsage(
		ctx,
		nil,
		encryptor,
		registry,
		organizationID,
		canvasID,
		versionID,
		pbCanvas,
		autoLayout,
		webhookBaseURL,
		authService,
	)
}

func UpdateCanvasVersionWithUsage(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.UpdateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	organizationUUID := uuid.MustParse(organizationID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
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

	err = usage.EnsureOrganizationWithinLimits(
		ctx,
		usageService,
		organizationID,
		&usagepb.OrganizationState{},
		&usagepb.CanvasState{
			Nodes: int32(len(expandedNodes)),
		},
	)

	if err != nil {
		return nil, err
	}

	requestedVersionID := strings.TrimSpace(versionID)
	if requestedVersionID == "" {
		return nil, status.Error(codes.InvalidArgument, "version id is required")
	}

	versionUUID, err := uuid.Parse(requestedVersionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.TransactionWithContext(ctx, database.DefaultCanvasMutationTimeout, "UpdateCanvasVersion", func(tx *gorm.DB) error {
		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		nodes := injectMetadataIntoNodes(version.Nodes, nodes)

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if version.State == models.CanvasVersionStatePublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		if _, draftErr := models.FindCanvasDraftByVersionInTransaction(tx, canvasUUID, userUUID, version.ID); draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.FailedPrecondition, "version is not your current edit version")
			}
			return draftErr
		}

		now := time.Now()
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.UpdatedAt = &now

		return tx.Save(version).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update canvas version")
		return nil, status.Error(codes.Internal, "failed to update canvas version")
	}

	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), version.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas update RabbitMQ message: %v", err)
	}

	return &pb.UpdateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}

func injectMetadataIntoNodes(versionNodes []models.Node, proposedNodes []models.Node) []models.Node {
	result := make([]models.Node, len(proposedNodes))
	copy(result, proposedNodes)

	for i, proposedNode := range result {
		for _, versionNode := range versionNodes {
			if proposedNode.ID == versionNode.ID {
				result[i].Metadata = versionNode.Metadata
			}
		}
	}

	return result
}
