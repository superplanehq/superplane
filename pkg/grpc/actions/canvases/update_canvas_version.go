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
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
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
) (*models.CanvasVersion, error) {
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
		false,
		false,
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
	discardStaging bool,
	commitTarget bool,
) (*models.CanvasVersion, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}
	organizationUUID := uuid.MustParse(organizationID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "failed to apply layout")
	}

	err = usage.EnsureOrganizationWithinLimits(
		ctx,
		usageService,
		organizationID,
		&usagepb.OrganizationState{},
		&usagepb.CanvasState{
			Nodes: int32(len(nodes)),
		},
	)

	if err != nil {
		return nil, err
	}

	requestedVersionID := strings.TrimSpace(versionID)
	if requestedVersionID == "" {
		return nil, grpcerrors.InvalidArgument(nil, "version id is required")
	}

	versionUUID, err := uuid.Parse(requestedVersionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(err, "version not found")
			}
			return err
		}

		nodes := injectMetadataIntoNodes(version.Nodes, nodes)

		if err := ensureVersionIsEditable(userUUID, canvas, version, commitTarget); err != nil {
			return err
		}

		now := time.Now()
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.UpdatedAt = &now

		if err := tx.Save(version).Error; err != nil {
			return err
		}

		if discardStaging {
			return models.DiscardStagedFilesForUser(tx, canvas.ID, userUUID, nil)
		}

		return nil
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		log.WithError(err).Error("failed to update canvas version")
		return nil, grpcerrors.Internal(err, "failed to update canvas version")
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), organizationID).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}

	return version, nil
}

func ensureVersionIsEditable(userID uuid.UUID, canvas *models.Canvas, version *models.CanvasVersion, commitTarget bool) error {
	if commitTarget {
		if models.IsLiveCanvasVersion(nil, canvas, version) {
			return grpcerrors.FailedPrecondition(nil, "live versions are immutable")
		}
		if version.OwnerID == nil || *version.OwnerID != userID {
			return grpcerrors.PermissionDenied(nil, "version owner mismatch")
		}
		return nil
	}

	return grpcerrors.FailedPrecondition(nil, "direct version updates are not supported; stage changes instead")
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
