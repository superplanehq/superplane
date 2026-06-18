package grpc

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	agentsActions "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/oidc"
	pbActions "github.com/superplanehq/superplane/pkg/protos/actions"
	pbAgents "github.com/superplanehq/superplane/pkg/protos/agents"
	pbCanvasFolders "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbIntegrations "github.com/superplanehq/superplane/pkg/protos/integrations"
	pbMe "github.com/superplanehq/superplane/pkg/protos/me"
	pbOrganizations "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pbSecrets "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbServiceAccounts "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	pbTriggers "github.com/superplanehq/superplane/pkg/protos/triggers"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWidgets "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

type Services struct {
	Users           pbUsers.UsersServer
	Groups          pbGroups.GroupsServer
	Roles           pbRoles.RolesServer
	Organizations   pbOrganizations.OrganizationsServer
	Integrations    pbIntegrations.IntegrationsServer
	Secrets         pbSecrets.SecretsServer
	Me              pbMe.MeServer
	Actions         pbActions.ActionsServer
	Triggers        pbTriggers.TriggersServer
	Widgets         pbWidgets.WidgetsServer
	Canvases        pbCanvases.CanvasesServer
	CanvasFolders   pbCanvasFolders.CanvasFoldersServer
	ServiceAccounts pbServiceAccounts.ServiceAccountsServer
	Agents          pbAgents.AgentsServer
}

type ServicesConfig struct {
	BaseURL         string
	WebhooksBaseURL string
	Encryptor       crypto.Encryptor
	AuthService     authorization.Authorization
	Registry        *registry.Registry
	OIDCProvider    oidc.Provider
	GitProvider     git.Provider
	AgentService    agentsActions.AgentsService
	UsageService    usage.Service
}

func NewServices(cfg ServicesConfig) (*Services, error) {
	if cfg.UsageService == nil {
		return nil, fmt.Errorf("usage service is required")
	}

	chain := NewUnaryChain(cfg.AuthService)

	return &Services{
		Users: WrapUsersServer(
			NewUsersService(cfg.AuthService),
			chain,
		),
		Groups: WrapGroupsServer(
			NewGroupsService(cfg.AuthService),
			chain,
		),
		Roles: WrapRolesServer(
			NewRoleService(cfg.AuthService),
			chain,
		),
		Organizations: WrapOrganizationsServer(
			NewOrganizationService(
				cfg.AuthService,
				cfg.Registry,
				cfg.OIDCProvider,
				cfg.BaseURL,
				cfg.WebhooksBaseURL,
				cfg.UsageService,
			),
			chain,
		),
		Integrations: WrapIntegrationsServer(
			NewIntegrationService(cfg.Encryptor, cfg.Registry),
			chain,
		),
		Secrets: WrapSecretsServer(
			NewSecretService(cfg.Encryptor, cfg.AuthService),
			chain,
		),
		Me: WrapMeServer(
			NewMeService(cfg.AuthService),
			chain,
		),
		Actions: WrapActionsServer(
			NewActionService(cfg.Registry),
			chain,
		),
		Triggers: WrapTriggersServer(
			NewTriggerService(cfg.Registry),
			chain,
		),
		Widgets: WrapWidgetsServer(
			NewWidgetService(cfg.Registry),
			chain,
		),
		Canvases: WrapCanvasesServer(
			NewCanvasService(
				cfg.AuthService,
				cfg.Registry,
				cfg.Encryptor,
				cfg.GitProvider,
				cfg.WebhooksBaseURL,
				cfg.UsageService,
			),
			chain,
		),
		CanvasFolders: WrapCanvasFoldersServer(
			NewCanvasFolderService(),
			chain,
		),
		ServiceAccounts: WrapServiceAccountsServer(
			NewServiceAccountsService(cfg.AuthService),
			chain,
		),
		Agents: WrapAgentsServer(
			NewAgentsService(cfg.AgentService),
			chain,
		),
	}, nil
}
