package canvases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func resolveBranchName(branchName string, version *models.CanvasVersion) string {
	if strings.TrimSpace(branchName) != "" {
		return strings.TrimSpace(branchName)
	}
	if version != nil && version.GitBranch != "" {
		return version.GitBranch
	}
	return models.CanvasGitBranchMain
}

func loadBranchForStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
	branchName string,
	versionID string,
) (*models.Canvas, *models.WorkflowBranch, *models.CanvasVersion, uuid.UUID, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, nil, uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, nil, nil, uuid.Nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, nil, uuid.Nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, uuid.Nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	var headVersion *models.CanvasVersion
	if strings.TrimSpace(versionID) != "" {
		versionUUID, parseErr := uuid.Parse(versionID)
		if parseErr != nil {
			return nil, nil, nil, uuid.Nil, grpcerrors.InvalidArgument(parseErr, "invalid version id")
		}
		headVersion, err = models.FindCanvasVersion(canvas.ID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, nil, uuid.Nil, grpcerrors.NotFound(err, "version not found")
			}
			return nil, nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load version")
		}
	}

	resolvedBranchName := resolveBranchName(branchName, headVersion)
	branch, err := findBranchByName(database.DB(ctx), canvas.ID, resolvedBranchName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, uuid.Nil, grpcerrors.NotFound(err, fmt.Sprintf("branch %q not found", resolvedBranchName))
		}
		return nil, nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load branch")
	}

	if headVersion == nil {
		if branch.HeadVersionID == nil {
			return nil, nil, nil, uuid.Nil, grpcerrors.FailedPrecondition(nil, "branch has no commits")
		}
		headVersion, err = models.FindCanvasVersion(canvas.ID, *branch.HeadVersionID)
		if err != nil {
			return nil, nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load branch head")
		}
	}

	userUUID := uuid.MustParse(userID)
	return canvas, branch, headVersion, userUUID, nil
}

func findBranchByName(tx *gorm.DB, canvasID uuid.UUID, branchName string) (*models.WorkflowBranch, error) {
	return models.FindWorkflowBranch(tx, canvasID, branchName)
}
