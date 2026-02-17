package grpc

import (
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"

	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/oidc"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbComponents "github.com/superplanehq/superplane/pkg/protos/components"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	mepb "github.com/superplanehq/superplane/pkg/protos/me"
	organizationPb "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	secretPb "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbServiceAccounts "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	triggerPb "github.com/superplanehq/superplane/pkg/protos/triggers"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	widgetPb "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

//
// Main Entrypoint for the RepositoryHub server.
//

var (
	customFunc recovery.RecoveryHandlerFunc = sentryRecoveryHandler
)

func sentryRecoveryHandler(p any) error {
	log.Errorf("recovered from panic in gRPC handler: %v. Stack: %s", p, debug.Stack())

	hub := sentry.CurrentHub()
	if hub != nil && hub.Client() != nil {
		hub.Recover(p)
		hub.Flush(2 * time.Second)
	}

	return status.Errorf(codes.Internal, "internal server error")
}

func RunServer(baseURL, webhooksBaseURL, basePath string, encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry, oidcProvider oidc.Provider, port int) {
	endpoint := fmt.Sprintf("0.0.0.0:%d", port)
	lis, err := net.Listen("tcp", endpoint)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	//
	// Set up error handler middlewares for the server.
	//
	opts := []recovery.Option{
		recovery.WithRecoveryHandler(customFunc),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(opts...),
			authorization.NewAuthorizationInterceptor(authService).UnaryInterceptor(),
			sanitizeErrorUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(opts...),
		),
	)

	//
	// Initialize health service.
	//
	healthService := &HealthCheckServer{}
	health.RegisterHealthServer(grpcServer, healthService)

	//
	// Initialize services exposed by this server.
	//
	organizationService := NewOrganizationService(authService, registry, oidcProvider, baseURL, webhooksBaseURL)
	organizationPb.RegisterOrganizationsServer(grpcServer, organizationService)

	userService := NewUsersService(authService)
	pbUsers.RegisterUsersServer(grpcServer, userService)

	groupService := NewGroupsService(authService)
	pbGroups.RegisterGroupsServer(grpcServer, groupService)

	roleService := NewRoleService(authService)
	pbRoles.RegisterRolesServer(grpcServer, roleService)

	secretsService := NewSecretService(encryptor, authService)
	secretPb.RegisterSecretsServer(grpcServer, secretsService)

	meService := NewMeService()
	mepb.RegisterMeServer(grpcServer, meService)

	componentService := NewComponentService(registry)
	pbComponents.RegisterComponentsServer(grpcServer, componentService)

	triggerService := NewTriggerService(registry)
	triggerPb.RegisterTriggersServer(grpcServer, triggerService)

	widgetService := NewWidgetService(registry)
	widgetPb.RegisterWidgetsServer(grpcServer, widgetService)

	blueprintService := NewBlueprintService(registry)
	pbBlueprints.RegisterBlueprintsServer(grpcServer, blueprintService)

	canvasService := NewCanvasService(authService, registry, encryptor, webhooksBaseURL+basePath)
	pbCanvases.RegisterCanvasesServer(grpcServer, canvasService)

	integrationService := NewIntegrationService(encryptor, registry)
	integrationpb.RegisterIntegrationsServer(grpcServer, integrationService)

	serviceAccountsService := NewServiceAccountsService(authService)
	pbServiceAccounts.RegisterServiceAccountsServer(grpcServer, serviceAccountsService)

	reflection.Register(grpcServer)

	//
	// Start handling incoming requests
	//
	log.Infof("Starting GRPC on %s.", endpoint)
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}
