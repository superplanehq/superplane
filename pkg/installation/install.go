package installation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type InstallRequest struct {
	Repo           string
	Name           string
	OrganizationID uuid.UUID
	AccountID      uuid.UUID
}

type InstallResult struct {
	CanvasID       string
	OrganizationID string
}

type Service struct {
	Registry        *registry.Registry
	Encryptor       crypto.Encryptor
	AuthService     authorization.Authorization
	GitProvider     git.Provider
	WebhooksBaseURL string
	UsageService    usage.Service
}

func (s *Service) Preview(repoParam string) (*Preview, error) {
	repoParam = strings.TrimSpace(repoParam)
	if repoParam == "" {
		return nil, fmt.Errorf("repo query parameter is required")
	}

	return BuildPreview(repoParam)
}

func (s *Service) Install(ctx context.Context, req InstallRequest) (*InstallResult, error) {
	repo, err := ParseRepository(req.Repo)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	user, err := FindActiveUserForAccountInOrganization(req.AccountID, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	allowed, err := s.AuthService.CheckOrganizationPermission(
		user.ID.String(),
		req.OrganizationID.String(),
		"canvases",
		"create",
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check canvas create permission: %v", err)
	}

	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "You do not have permission to create apps in this organization")
	}

	canvas, _, err := FetchCanvas(repo)
	if err != nil {
		return nil, err
	}

	// Pre-parse the optional console.yaml from the same ref. Doing this
	// before CreateCanvas means a malformed console aborts the install
	// without leaving an orphan canvas behind.
	console, err := FetchConsole(repo, repo.Ref)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	canvas.Metadata.Name = name

	ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())
	response, err := canvases.CreateCanvas(
		ctx,
		s.Registry,
		s.Encryptor,
		s.AuthService,
		s.GitProvider,
		s.WebhooksBaseURL,
		req.OrganizationID,
		canvas,
		nil,
		s.UsageService,
	)
	if err != nil {
		return nil, err
	}

	canvasID := ""
	if response != nil && response.Canvas != nil && response.Canvas.Metadata != nil {
		canvasID = response.Canvas.Metadata.Id
	}

	if canvasID == "" {
		return nil, fmt.Errorf("failed to install app")
	}

	if console != nil {
		if err := seedInstalledConsole(ctx, s.GitProvider, s.Registry, s.Encryptor, s.AuthService, s.WebhooksBaseURL, req.OrganizationID, canvasID, console); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to install console: %v", err)
		}
	}

	return &InstallResult{
		CanvasID:       canvasID,
		OrganizationID: req.OrganizationID.String(),
	}, nil
}

func seedInstalledConsole(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	webhookBaseURL string,
	organizationID uuid.UUID,
	canvasID string,
	console *models.DashboardYAML,
) error {
	if console == nil {
		return nil
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return fmt.Errorf("invalid canvas id %q: %w", canvasID, err)
	}

	repository, err := models.FindRepository(organizationID, canvasUUID)
	if err != nil {
		return err
	}

	consoleYAML, err := materialize.BuildConsoleYAMLFromDashboard(console)
	if err != nil {
		return err
	}

	headSHA, err := gitProvider.Head(ctx, repository.RepoID, models.CanvasGitBranchMain)
	if err != nil {
		return err
	}

	commitSHA, err := gitProvider.Commit(ctx, repository.RepoID, git.CommitOptions{
		Branch:          models.CanvasGitBranchMain,
		BaseBranch:      models.CanvasGitBranchMain,
		ExpectedHeadSHA: headSHA,
		Message:         "Install console",
		Author: git.CommitAuthor{
			Name:  "SuperPlane",
			Email: "support@superplane.com",
		},
		Operations: []git.FileOperation{
			{
				Path:      materialize.ConsoleFileName,
				Content:   bytes.NewReader(consoleYAML),
				SizeBytes: int64(len(consoleYAML)),
			},
		},
	})
	if err != nil {
		return err
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		mat := &materialize.Materializer{
			GitProvider:    gitProvider,
			Registry:       registry,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
		}
		_, matErr := mat.MaterializeFromGit(ctx, tx, organizationID, canvasUUID, models.CanvasGitBranchMain, commitSHA, materialize.ModeLive, nil)
		return matErr
	})
}

func FindActiveUserForAccountInOrganization(accountID, organizationID uuid.UUID) (*models.User, error) {
	account, err := models.FindAccountByID(accountID.String())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "account not found")
	}

	user, err := models.FindMaybeDeletedUserByEmailInTransaction(database.Conn(), organizationID.String(), account.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.PermissionDenied, "you are not a member of this organization")
		}

		return nil, status.Error(codes.Internal, "failed to resolve organization membership")
	}

	if user.DeletedAt.Valid {
		return nil, status.Error(codes.PermissionDenied, "you are not a member of this organization")
	}

	return user, nil
}
