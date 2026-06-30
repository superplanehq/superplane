package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func PublishCanvasVersion(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	canvasID string,
	versionID string,
	commitMessage string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.PublishCanvasVersionResponse, error) {
	_ = encryptor
	_ = reg
	_ = gitProv
	_ = webhookBaseURL
	_ = authService

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	organizationUUID := uuid.MustParse(organizationID)
	userUUID := uuid.MustParse(userID)

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	publishedVersion, err := mergeBranchToMainInTransaction(
		ctx,
		organizationUUID,
		canvasUUID,
		versionUUID,
		userUUID,
		commitMessage,
	)
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, err
	}

	if err := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); err != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", err)
	}
	if err := messages.NewCanvasVersionUpdatedMessage(canvas.ID.String(), publishedVersion.ID.String()).PublishVersionUpdated(); err != nil {
		log.Errorf("failed to publish canvas version updated RabbitMQ message: %v", err)
	}

	return &pb.PublishCanvasVersionResponse{
		Version: SerializeCanvasVersion(publishedVersion, organizationID, nil, canvas),
	}, nil
}

func mergeBranchToMainInTransaction(
	ctx context.Context,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	sourceVersionUUID uuid.UUID,
	userUUID uuid.UUID,
	commitMessage string,
) (*models.CanvasVersion, error) {
	_ = ctx
	var publishedVersion *models.CanvasVersion

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		sourceVersion, findErr := models.FindCanvasVersionForUpdateInTransaction(tx, canvasUUID, sourceVersionUUID)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(findErr, "version not found")
			}
			return findErr
		}

		if sourceVersion.GitBranch == models.CanvasGitBranchMain {
			publishedVersion = sourceVersion
			return nil
		}

		branch, branchErr := models.FindWorkflowBranch(tx, canvasUUID, sourceVersion.GitBranch)
		if branchErr != nil {
			return branchErr
		}

		hasStaging, err := models.HasWorkflowStagingInTransaction(tx, branch.ID, userUUID)
		if err != nil {
			return err
		}
		if hasStaging {
			return grpcerrors.FailedPrecondition(nil, "branch has uncommitted staged changes")
		}

		created, createErr := models.CreateCommitOnBranch(tx, models.CreateCommitInput{
			WorkflowID:    canvasUUID,
			BranchName:    models.CanvasGitBranchMain,
			OwnerID:       userUUID,
			CommitMessage: resolvedMergeCommitMessage(commitMessage, sourceVersion.GitBranch),
			Nodes:         append([]models.Node(nil), sourceVersion.Nodes...),
			Edges:         append([]models.Edge(nil), sourceVersion.Edges...),
			ConsolePanels: sourceVersion.ConsolePanels.Data(),
			ConsoleLayout: sourceVersion.ConsoleLayout.Data(),
		})
		if createErr != nil {
			return createErr
		}

		if err := models.DeleteWorkflowBranch(tx, canvasUUID, sourceVersion.GitBranch); err != nil {
			return err
		}

		publishedVersion = created
		return nil
	})

	if err != nil {
		return nil, err
	}

	return publishedVersion, nil
}

func resolvedMergeCommitMessage(requestedMessage, branchName string) string {
	if trimmed := strings.TrimSpace(requestedMessage); trimmed != "" {
		return trimmed
	}
	return "Merge branch '" + branchName + "'"
}

func publishDraftVersionInTransaction(
	ctx context.Context,
	encryptor crypto.Encryptor,
	reg *registry.Registry,
	gitProv gitprovider.Provider,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasUUID uuid.UUID,
	versionUUID uuid.UUID,
	userUUID uuid.UUID,
	authService authorization.Authorization,
	webhookBaseURL string,
) (*models.CanvasVersion, error) {
	return mergeBranchToMainInTransaction(ctx, organizationUUID, canvasUUID, versionUUID, userUUID, "")
}
