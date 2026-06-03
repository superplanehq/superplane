package canvases

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CreateDraftBranch(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	displayName string,
) (*pb.CreateDraftBranchResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}
	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	repository, err := models.FindRepository(orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "repository not found: %v", err)
	}
	if repository.Status != models.RepositoryStatusReady {
		return nil, status.Error(codes.FailedPrecondition, "repository is not ready")
	}

	userUUID := uuid.MustParse(userID)
	branchName, err := uniqueDraftBranchName(canvasUUID, userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate draft branch name: %v", err)
	}

	mainHead, err := gitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read main branch head: %v", err)
	}

	if err := gitProvider.CreateBranch(ctx, repository.RepoID, branchName, models.CanvasGitBranchMain); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create draft branch: %v", err)
	}

	label := strings.TrimSpace(displayName)
	if label == "" {
		label, err = nextDraftDisplayName(canvasUUID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate draft name: %v", err)
		}
	}

	var branch *models.CanvasDraftBranch
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		branch = &models.CanvasDraftBranch{
			CanvasID:       canvasUUID,
			OrganizationID: orgUUID,
			BranchName:     branchName,
			DisplayName:    label,
			OwnerID:        &userUUID,
			CreatedBy:      &userUUID,
			TipSHA:         mainHead,
		}
		if createErr := models.CreateDraftBranchInTransaction(tx, branch); createErr != nil {
			return createErr
		}

		mat := &materialize.DraftMaterializer{GitProvider: gitProvider, Registry: registry}
		_, matErr := mat.MaterializeDraft(ctx, tx, orgUUID, canvasUUID, branchName, mainHead, &userUUID)
		return matErr
	})
	if err != nil {
		_ = gitProvider.DeleteBranch(ctx, repository.RepoID, branchName)
		return nil, status.Errorf(codes.Internal, "failed to create draft branch: %v", err)
	}

	return &pb.CreateDraftBranchResponse{
		Branch: serializeDraftBranch(branch, organizationID, nil),
	}, nil
}

// uniqueDraftBranchName returns a draft branch name that does not yet exist for
// the canvas. The first draft for a user keeps the default name so CLI and
// change-request defaults stay aligned; subsequent drafts get a unique suffix.
func uniqueDraftBranchName(canvasID, userID uuid.UUID) (string, error) {
	base := materialize.DefaultDraftBranchName(userID)
	candidate := base
	for attempt := 0; attempt < 50; attempt++ {
		existing, err := models.FindDraftBranch(canvasID, candidate)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%s", base, uuid.NewString()[:8])
	}

	return "", fmt.Errorf("could not generate a unique draft branch name after multiple attempts")
}

var draftDisplayNamePattern = regexp.MustCompile(`^Draft #(\d+)$`)

// nextDraftDisplayName returns a sequential, human-friendly draft name such as
// "Draft #1", "Draft #2". It picks the next number after the highest existing
// "Draft #N" label so names stay distinct even after some drafts are deleted.
func nextDraftDisplayName(canvasID uuid.UUID) (string, error) {
	branches, err := models.ListDraftBranchesForCanvas(canvasID)
	if err != nil {
		return "", err
	}

	highest := 0
	for _, branch := range branches {
		matches := draftDisplayNamePattern.FindStringSubmatch(branch.DisplayName)
		if matches == nil {
			continue
		}
		if n, convErr := strconv.Atoi(matches[1]); convErr == nil && n > highest {
			highest = n
		}
	}

	return fmt.Sprintf("Draft #%d", highest+1), nil
}
