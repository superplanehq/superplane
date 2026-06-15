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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
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
	InstallParams  map[string]string
	Integrations   map[string]IntegrationMapping // integration type name → instance to wire
}

// IntegrationMapping identifies a specific integration instance to wire
// into canvas nodes of the corresponding integration type.
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

	canvas, console, err := s.resolveCanvasFromRepo(repo, name, req)
	if err != nil {
		return nil, err
	}

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

	if err := persistInstalledConsole(canvasID, console); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to install console: %v", err)
	}

	return &InstallResult{
		CanvasID:       canvasID,
		OrganizationID: req.OrganizationID.String(),
	}, nil
}

// resolveCanvasFromRepo fetches, parametrizes, and prepares a canvas for creation.
func (s *Service) resolveCanvasFromRepo(
	repo *Repository,
	name string,
	req InstallRequest,
) (*pb.Canvas, *models.ConsoleYAML, error) {
	canvasBody, _, err := fetchRawCanvasFile(repo)
	if err != nil {
		return nil, nil, err
	}

	canvasBody, err = applyInstallParams(canvasBody, repo, req.InstallParams)
	if err != nil {
		return nil, nil, err
	}

	canvas, err := parseCanvasYAML(canvasBody)
	if err != nil {
		return nil, nil, err
	}

	console, err := FetchConsole(repo, repo.Ref)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	canvas.Metadata.Name = name

	if len(req.Integrations) > 0 {
		wireIntegrations(canvas, req.Integrations, s.Registry)
	}

	return canvas, console, nil
}

// applyInstallParams validates and substitutes template parameters in raw YAML.
// Returns the substituted YAML bytes, ready for parsing.
func applyInstallParams(canvasBody []byte, repo *Repository, userParams map[string]string) ([]byte, error) {
	params, err := FetchParams(repo, repo.Ref)
	if err != nil {
		log.Warnf("failed to load params.json for %s: %v", repo.String(), err)
	}

	if params == nil || len(params.InstallParams) == 0 {
		return canvasBody, nil
	}

	if userParams != nil {
		if err := ValidateInstallParams(params.InstallParams, userParams); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return SubstituteInstallParams(canvasBody, ResolveInstallParams(params.InstallParams, userParams)), nil
	}

	return SubstituteInstallParams(canvasBody, DefaultParamValues(params.InstallParams)), nil
}

// persistInstalledConsole writes the optional console for a freshly created
// canvas. A nil console is a no-op (the repo did not ship a console.yaml).
//
// Note: this runs after canvases.CreateCanvas, in its own transaction. If the
// upsert fails, the canvas already exists; the user can re-import the console
// from the UI. We accept that trade-off to avoid changing CreateCanvas's
// signature just for this side-effect.
func persistInstalledConsole(canvasID string, console *models.ConsoleYAML) error {
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

// wireIntegrations sets the Integration ref on each canvas node whose
// component belongs to an integration type present in the mapping.
func wireIntegrations(canvas *pb.Canvas, mappings map[string]IntegrationMapping, reg *registry.Registry) {
	if canvas.Spec == nil {
		return
	}

	componentToIntegration := buildComponentIntegrationMap(reg)

	for _, node := range canvas.Spec.Nodes {
		if node.Component == "" {
			continue
		}

		integrationName := componentToIntegration[node.Component]
		if integrationName == "" {
			continue
		}

		mapping, ok := mappings[integrationName]
		if !ok {
			continue
		}

		node.Integration = &componentpb.IntegrationRef{
			Id:   &mapping.ID,
			Name: &mapping.Name,
		}
	}
}

// buildComponentIntegrationMap creates a lookup from component name to
// integration name using the registry. Built once, used for all nodes.
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

// findIntegrationForComponent returns the integration name that owns
// the given node's component (trigger or action). Returns "" if not found.
func findIntegrationForComponent(node *componentpb.Node, reg *registry.Registry) string {
	if node.Component == "" || reg == nil {
		return ""
	}

	result := buildComponentIntegrationMap(reg)
	return result[node.Component]
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
