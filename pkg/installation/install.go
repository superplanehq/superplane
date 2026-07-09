package installation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type InstallRequest struct {
	Repo           string
	Name           string
	OrganizationID uuid.UUID
	AccountID      uuid.UUID
	InstallParams  map[string]string
	Integrations   map[string]IntegrationMapping
}

type IntegrationMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

	return BuildPreview(repoParam, s.Registry)
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

	user, err := checkInstallPermission(s.AuthService, req.AccountID, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	canvas, resolvedParams, err := s.prepareCanvasForInstall(repo, name, req.InstallParams, req.Integrations, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	console, err := FetchConsole(repo, repo.Ref)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	seedFiles, err := fetchSeedFiles(repo, resolvedParams)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())
	canvasID, err := s.createCanvas(ctx, req.OrganizationID, canvas, seedFiles)
	if err != nil {
		return nil, translateInstallError(err)
	}

	if err := persistInstalledConsole(canvasID, console); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to install console: %v", err)
	}

	return &InstallResult{
		CanvasID:       canvasID,
		OrganizationID: req.OrganizationID.String(),
	}, nil
}

// ─── Install helpers ─────────────────────────────────────────────────────────

func checkInstallPermission(
	authService authorization.Authorization,
	accountID, organizationID uuid.UUID,
) (*models.User, error) {
	user, err := FindActiveUserForAccountInOrganization(accountID, organizationID)
	if err != nil {
		return nil, err
	}

	allowed, err := authService.CheckOrganizationPermission(
		context.Background(),
		user.ID.String(),
		organizationID.String(),
		"canvases",
		"create",
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check canvas create permission: %v", err)
	}

	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "You do not have permission to create apps in this organization")
	}

	return user, nil
}

func (s *Service) prepareCanvasForInstall(
	repo *Repository,
	name string,
	userParams map[string]string,
	integrations map[string]IntegrationMapping,
	organizationID uuid.UUID,
) (*yaml.Canvas, map[string]string, error) {
	canvasBody, resolvedParams, err := fetchAndSubstituteParams(repo, userParams, organizationID)
	if err != nil {
		return nil, nil, err
	}

	canvas, err := yaml.CanvasFromYAML(canvasBody)
	if err != nil {
		return nil, nil, err
	}

	canvas.Metadata.Name = name
	wireIntegrations(canvas, integrations, s.Registry)

	return canvas, resolvedParams, nil
}

func fetchAndSubstituteParams(repo *Repository, userParams map[string]string, organizationID uuid.UUID) ([]byte, map[string]string, error) {
	canvasBody, _, err := fetchRawCanvasFile(repo)
	if err != nil {
		return nil, nil, err
	}

	params, err := FetchParams(repo, repo.Ref)
	if err != nil {
		log.Warnf("failed to load params.json for %s: %v", repo.String(), err)
	}

	if params == nil || len(params.InstallParams) == 0 {
		return canvasBody, nil, nil
	}

	// A nil userParams map means the install bypassed the params wizard
	// ("just take me there"), which falls back to schema defaults. Only
	// enforce required-param validation when the user actually submitted
	// values; otherwise required params without defaults would always fail.
	if userParams != nil {
		if err := ValidateInstallParams(params.InstallParams, userParams); err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
	}

	// Validate secret_picker params against the user-supplied values (and
	// explicit defaults), not the resolved map: ResolveInstallParams fills
	// unset params with placeholder/param-name fallbacks that are not real
	// secret names, so validating those would reject optional pickers left
	// empty.
	if err := ValidateSecretPickerParams(params.InstallParams, userParams, organizationID); err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	resolved := ResolveInstallParams(params.InstallParams, userParams)
	return SubstituteInstallParams(canvasBody, resolved), resolved, nil
}

func (s *Service) createCanvas(ctx context.Context, organizationID uuid.UUID, canvas *yaml.Canvas, seedFiles []models.RepositorySeedFile) (string, error) {
	nodes, edges, err := canvas.Parse(s.Registry, organizationID.String())
	if err != nil {
		return "", err
	}

	response, err := canvases.CreateCanvasWithSeedFiles(
		ctx,
		s.Registry,
		s.Encryptor,
		s.AuthService,
		s.GitProvider,
		s.WebhooksBaseURL,
		organizationID,
		canvas.Metadata.Name,
		canvas.Metadata.Description,
		nodes,
		edges,
		nil,
		s.UsageService,
		seedFiles,
	)
	if err != nil {
		return "", err
	}

	canvasID := response.GetCanvas().GetMetadata().GetId()
	if canvasID == "" {
		return "", fmt.Errorf("failed to install app: empty canvas ID in response")
	}

	return canvasID, nil
}

// translateInstallError rewrites canvas-creation status errors into messages
// that match the install wizard's vocabulary. Callers see "App" rather than
// "Canvas" because the install flow is a user-facing app installation, even
// though the underlying resource is a canvas.
func translateInstallError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	if st.Code() == codes.AlreadyExists {
		return status.Error(codes.AlreadyExists, "An App with the same name already exists")
	}

	return err
}

// fetchSeedFiles downloads every file in the app repository except the spec
// files (canvas.yaml/console.yaml) and params.json, converting them into
// model rows ready to be persisted alongside the pending canvas repository.
// When resolvedParams is non-nil, {{ install_params.xxx }} placeholders in
// file contents are replaced with the resolved values, matching the
// substitution applied to canvas.yaml.
// Failures are surfaced as InvalidArgument so the install request returns a
// useful 400 instead of leaving a half-installed canvas behind.
func fetchSeedFiles(repo *Repository, resolvedParams map[string]string) ([]models.RepositorySeedFile, error) {
	files, err := FetchRepositoryFiles(repo, repo.Ref)
	if err != nil {
		return nil, fmt.Errorf("fetch repository files: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	seedFiles := make([]models.RepositorySeedFile, 0, len(files))
	for _, file := range files {
		content := file.Content
		if len(resolvedParams) > 0 {
			content = SubstituteInstallParams(content, resolvedParams)
		}

		seedFiles = append(seedFiles, models.RepositorySeedFile{
			Path:    file.Path,
			Content: content,
		})
	}

	return seedFiles, nil
}

// ─── Console persistence ─────────────────────────────────────────────────────

func persistInstalledConsole(canvasID string, console *yaml.Console) error {
	if console == nil {
		return nil
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return fmt.Errorf("invalid canvas id %q: %w", canvasID, err)
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		version, findErr := models.FindLiveCanvasVersionInTransaction(tx, canvasUUID)
		if findErr != nil {
			return findErr
		}

		_, err := models.UpdateCanvasVersionConsoleInTransaction(
			tx,
			version,
			console.Spec.Panels,
			console.Spec.Layout,
		)
		return err
	})
}

// ─── Integration wiring ──────────────────────────────────────────────────────

func wireIntegrations(canvas *yaml.Canvas, mappings map[string]IntegrationMapping, reg *registry.Registry) {
	if canvas.Spec == nil || len(mappings) == 0 {
		return
	}

	componentToIntegration := buildComponentIntegrationMap(reg)

	newNodes := make([]yaml.Node, len(canvas.Nodes()))
	for i, node := range canvas.Spec.Nodes {
		n := node
		integrationName := componentToIntegration[node.Component]
		mapping, ok := mappings[integrationName]
		if !ok {
			newNodes[i] = n
			continue
		}

		n.Integration = &yaml.IntegrationRef{
			ID:   mapping.ID,
			Name: mapping.Name,
		}
		newNodes[i] = n
	}

	canvas.Spec.Nodes = newNodes
}

func buildComponentIntegrationMap(reg *registry.Registry) map[string]string {
	result := make(map[string]string)
	if reg == nil {
		return result
	}

	for _, integration := range reg.ListIntegrations() {
		for _, trigger := range integration.Triggers() {
			result[trigger.Name()] = integration.Name()
		}
		for _, action := range integration.Actions() {
			result[action.Name()] = integration.Name()
		}
	}

	return result
}

func findIntegrationForComponent(node *componentpb.Node, reg *registry.Registry) string {
	if node.Component == "" || reg == nil {
		return ""
	}

	return buildComponentIntegrationMap(reg)[node.Component]
}

// ─── Account / org resolution ────────────────────────────────────────────────

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
