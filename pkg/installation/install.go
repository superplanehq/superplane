package installation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

	// Resolve the ref by trying to fetch the canvas file.
	// We need the ref for params.json and console.yaml too.
	var canvasBody []byte
	if repo.Ref == "" {
		for _, ref := range defaultRefs {
			body, fetchErr := fetchURL(rawFileURL(repo, ref, canvasFileName))
			if fetchErr == nil {
				repo.Ref = ref
				canvasBody = body
				break
			}
		}
		if canvasBody == nil {
			return nil, fmt.Errorf("canvas.yaml not found on main or master branch")
		}
	} else {
		var fetchErr error
		canvasBody, fetchErr = fetchURL(rawFileURL(repo, repo.Ref, canvasFileName))
		if fetchErr != nil {
			return nil, fetchErr
		}
	}

	// Substitute install params in raw YAML before parsing.
	// If the template has params, we always need to substitute something
	// so the YAML is valid ({{ install_params.xxx }} is not valid YAML).
	params, _ := FetchParams(repo, repo.Ref)
	if params != nil && len(params.InstallParams) > 0 {
		if len(req.InstallParams) > 0 {
			// Wizard flow: user provided values, validate and substitute.
			resolved := ResolveInstallParams(params.InstallParams, req.InstallParams)
			if err := ValidateInstallParams(params.InstallParams, resolved); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "%v", err)
			}
			canvasBody = SubstituteInstallParams(canvasBody, resolved)
		} else {
			// One-click flow: no params sent, substitute with defaults/placeholders.
			defaults := make(map[string]string, len(params.InstallParams))
			for _, p := range params.InstallParams {
				if p.Default != "" {
					defaults[p.Name] = p.Default
				} else if p.Placeholder != "" {
					defaults[p.Name] = p.Placeholder
				} else {
					defaults[p.Name] = p.Name
				}
			}
			canvasBody = SubstituteInstallParams(canvasBody, defaults)
		}
	}

	canvas, err := parseCanvasYAML(canvasBody)
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

	// Wire integration instances into canvas nodes.
	if len(req.Integrations) > 0 {
		wireIntegrations(canvas, req.Integrations, s.Registry)
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

	for _, node := range canvas.Spec.Nodes {
		if node.Component == "" {
			continue
		}

		integrationName := findIntegrationForComponent(node, reg)
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

// findIntegrationForComponent returns the integration name that owns
// the given node's component (trigger or action). Returns "" if not found.
func findIntegrationForComponent(node *componentpb.Node, reg *registry.Registry) string {
	if node.Component == "" {
		return ""
	}

	for _, integration := range reg.ListIntegrations() {
		for _, trigger := range integration.Triggers() {
			if trigger.Name() == node.Component {
				return integration.Name()
			}
		}

		for _, action := range integration.Actions() {
			if action.Name() == node.Component {
				return integration.Name()
			}
		}
	}

	return ""
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
