package grpc

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	adminRunnerActions "github.com/superplanehq/superplane/pkg/grpc/actions/admin/runners"
	agentsActions "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/oidc"
	pbActions "github.com/superplanehq/superplane/pkg/protos/actions"
	pbAdminRunners "github.com/superplanehq/superplane/pkg/protos/admin/runners"
	pbAgents "github.com/superplanehq/superplane/pkg/protos/agents"
	pbAPIKeys "github.com/superplanehq/superplane/pkg/protos/api_keys"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbFactories "github.com/superplanehq/superplane/pkg/protos/factories"
	pbFiles "github.com/superplanehq/superplane/pkg/protos/files"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbMe "github.com/superplanehq/superplane/pkg/protos/me"
	pbOrganizations "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecrets "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbTriggers "github.com/superplanehq/superplane/pkg/protos/triggers"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWidgets "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
)

type Services struct {
	Users         pbUsers.UsersServer
	Groups        pbGroups.GroupsServer
	Roles         pbRoles.RolesServer
	Organizations pbOrganizations.OrganizationsServer
	Integrations  pbIntegrations.IntegrationsServer
	Secrets       pbSecrets.SecretsServer
	Me            pbMe.MeServer
	Actions       pbActions.ActionsServer
	Triggers      pbTriggers.TriggersServer
	Widgets       pbWidgets.WidgetsServer
	Canvases      pbCanvases.CanvasesServer
	Factories     pbFactories.FactoriesServer
	Files         pbFiles.FilesServer
	APIKeys       pbAPIKeys.ApiKeysServer
	Agents        pbAgents.AgentsServer
	AdminRunners  pbAdminRunners.RunnersServer
}

type ServicesConfig struct {
	BaseURL          string
	WebhooksBaseURL  string
	RunnerAPIBaseURL string
	Encryptor        crypto.Encryptor
	AuthService      authorization.Authorization
	Registry         *registry.Registry
	OIDCProvider     oidc.Provider
	AgentService     agentsActions.AgentsService
	JWTSigner        *jwt.Signer

	// Entitlements and EnterpriseAccessControl default to Community mode, so
	// a missing value never grants Enterprise features.
	Entitlements            licensing.Entitlements
	EnterpriseAccessControl EnterpriseAccessControl
}

func NewServices(cfg ServicesConfig) (*Services, error) {
	if cfg.JWTSigner == nil {
		return nil, errors.New("JWT signer is required")
	}

	entitlements := cfg.Entitlements
	if entitlements == nil {
		entitlements = licensing.Community
	}

	var accessControl EnterpriseAccessControl = communityAccessControl{}
	if cfg.EnterpriseAccessControl != nil {
		accessControl = cfg.EnterpriseAccessControl
	}

	return &Services{
		Users:  NewUsersService(cfg.AuthService),
		Groups: NewGroupsService(cfg.AuthService, accessControl),
		Roles:  NewRoleService(cfg.AuthService, entitlements, accessControl),
		Organizations: NewOrganizationService(
			cfg.AuthService,
			cfg.Registry,
			cfg.OIDCProvider,
			cfg.BaseURL,
			cfg.WebhooksBaseURL,
		),
		Integrations: NewIntegrationService(cfg.Encryptor, cfg.Registry),
		Secrets:      NewSecretService(cfg.Encryptor, cfg.AuthService),
		Me:           NewMeService(cfg.AuthService),
		Actions:      NewActionService(cfg.Registry),
		Triggers:     NewTriggerService(cfg.Registry),
		Widgets:      NewWidgetService(cfg.Registry),
		Canvases: NewCanvasService(
			cfg.AuthService,
			cfg.Registry,
			cfg.Encryptor,
			cfg.WebhooksBaseURL,
		),
		Factories: NewFactoryService(
			cfg.Registry,
			cfg.Encryptor,
			cfg.AuthService,
			cfg.WebhooksBaseURL,
		),
		Files:   NewFilesService(cfg.AuthService),
		APIKeys: NewAPIKeysService(cfg.AuthService),
		Agents:  NewAgentsService(cfg.AgentService),
		AdminRunners: NewAdminRunnersService(
			adminRunnerActions.NewService(
				cfg.JWTSigner,
				cfg.RunnerAPIBaseURL,
			),
		),
	}, nil
}
