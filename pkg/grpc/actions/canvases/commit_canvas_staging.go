package canvases

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func CommitCanvasStaging(
	ctx context.Context,
	gitProvider gitprovider.Provider,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	branchName string,
	commitMessage string,
	newBranchName string,
	webhookBaseURL string,
	authService authorization.Authorization,
) (*pb.CommitCanvasStagingResponse, error) {
	_ = encryptor
	_ = webhookBaseURL
	_ = authService

	canvas, branch, headVersion, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, branchName, versionID)
	if err != nil {
		return nil, err
	}

	rows, err := models.ListWorkflowStaging(branch.ID, userUUID)
	if err != nil {
		log.WithError(err).WithFields(commitCanvasStagingLogFields(organizationID, canvasID, versionID, branchName, newBranchName, branch.Name, "")).Error("CommitCanvasStaging: failed to load staging")
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	specOps, gitOps := stagedCommitOperations(rows)

	targetBranchName := branch.Name
	if strings.TrimSpace(newBranchName) != "" {
		targetBranchName = strings.TrimSpace(newBranchName)
	}

	stagingBranchID := branch.ID
	parentBranchName := branch.Name
	sourceBranch := branch
	organizationUUID := uuid.MustParse(organizationID)
	message := resolvedStagingCommitMessage(commitMessage)

	var committed *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if targetBranchName != sourceBranch.Name {
			createdBranch, createErr := models.CreateWorkflowBranch(tx, canvas.ID, targetBranchName, nil)
			if createErr != nil {
				return createErr
			}
			sourceBranch = createdBranch
		}

		var createErr error
		committed, createErr = models.CreateCommitOnBranch(tx, models.CreateCommitInput{
			WorkflowID:    canvas.ID,
			BranchName:    targetBranchName,
			OwnerID:       userUUID,
			CommitMessage: message,
			Nodes:         append([]models.Node(nil), headVersion.Nodes...),
			Edges:         append([]models.Edge(nil), headVersion.Edges...),
			ConsolePanels: headVersion.ConsolePanels.Data(),
			ConsoleLayout: headVersion.ConsoleLayout.Data(),
		})
		if createErr != nil {
			return createErr
		}

		if len(specOps) > 0 {
			updated, applyErr := applyRepositorySpecFileOperationsInTransaction(
				ctx,
				tx,
				usageService,
				registry,
				organizationID,
				organizationUUID,
				canvas,
				committed.ID,
				specOps,
			)
			if applyErr != nil {
				return applyErr
			}
			committed = updated
		}

		if gitProvider != nil {
			commitSHA, gitErr := commitWorkflowVersionToGit(ctx, gitProvider, commitWorkflowVersionInput{
				Canvas:           canvas,
				Version:          committed,
				OrganizationID:   organizationID,
				UserID:           userUUID,
				Message:          message,
				BranchName:       targetBranchName,
				ParentBranchName: parentBranchName,
				ExtraGitOps:      gitOps,
			})
			if gitErr != nil {
				return gitErr
			}

			if err := models.UpdateCanvasVersionCommitSHAInTransaction(tx, canvas.ID, committed.ID, commitSHA); err != nil {
				return err
			}
			committed.CommitSHA = commitSHA
		}

		return models.DiscardWorkflowStagingInTransaction(tx, stagingBranchID, userUUID, nil)
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			log.WithError(err).WithFields(commitCanvasStagingLogFields(organizationID, canvasID, versionID, branchName, newBranchName, targetBranchName, parentBranchName)).Error("CommitCanvasStaging: failed to commit staging")
			return nil, err
		}
		log.WithError(err).WithFields(commitCanvasStagingLogFields(organizationID, canvasID, versionID, branchName, newBranchName, targetBranchName, parentBranchName)).Error("CommitCanvasStaging: failed to create commit")
		return nil, grpcerrors.Internal(err, "failed to create commit")
	}

	state, _, err := stagingSummaryForBranch(stagingBranchID, userUUID)
	if err != nil {
		return nil, err
	}

	return &pb.CommitCanvasStagingResponse{
		Version:        SerializeCanvasVersionMetadata(committed, organizationID, nil),
		StagingSummary: state,
	}, nil
}

func stagedCommitOperations(rows []models.WorkflowStaging) (specOps, gitOps []*pb.CanvasRepositoryFileOperation) {
	specContentByPath := map[string]string{}
	for _, row := range rows {
		if IsRepositorySpecFilePath(row.Path) {
			if row.Deleted {
				continue
			}
			specContentByPath[row.Path] = row.Content
			continue
		}

		gitOps = append(gitOps, &pb.CanvasRepositoryFileOperation{
			Path:    row.Path,
			Content: []byte(row.Content),
			Delete:  row.Deleted,
		})
	}

	for _, path := range []string{CanvasYAMLRepositoryPath, ConsoleYAMLRepositoryPath} {
		content, ok := specContentByPath[path]
		if !ok {
			continue
		}
		specOps = append(specOps, &pb.CanvasRepositoryFileOperation{
			Path:    path,
			Content: []byte(content),
		})
	}

	return specOps, gitOps
}

func commitCanvasStagingLogFields(
	organizationID,
	canvasID,
	versionID,
	branchName,
	newBranchName,
	targetBranchName,
	parentBranchName string,
) log.Fields {
	fields := log.Fields{
		"organization_id": organizationID,
		"canvas_id":       canvasID,
		"version_id":      versionID,
		"branch_name":     branchName,
	}
	if newBranchName != "" {
		fields["new_branch_name"] = newBranchName
	}
	if targetBranchName != "" {
		fields["target_branch_name"] = targetBranchName
	}
	if parentBranchName != "" {
		fields["parent_branch_name"] = parentBranchName
	}
	return fields
}

func resolvedStagingCommitMessage(messages ...string) string {
	for _, message := range messages {
		if trimmed := strings.TrimSpace(message); trimmed != "" {
			return trimmed
		}
	}
	return "Update canvas"
}
