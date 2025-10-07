package grpc

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	pbBlueprints "github.com/superplanehq/superplane/pkg/protos/blueprints"
	canvasPb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	integrationPb "github.com/superplanehq/superplane/pkg/protos/integrations"
	mepb "github.com/superplanehq/superplane/pkg/protos/me"
	organizationPb "github.com/superplanehq/superplane/pkg/protos/organizations"
	pbPrimitives "github.com/superplanehq/superplane/pkg/protos/primitives"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	secretPb "github.com/superplanehq/superplane/pkg/protos/secrets"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	pbWorkflows "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

//
// Main Entrypoint for the RepositoryHub server.
//

var (
	customFunc recovery.RecoveryHandlerFunc
)

func RunServer(encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry, port int) {
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
	service := NewCanvasService(encryptor, authService, registry)
	canvasPb.RegisterSuperplaneServer(grpcServer, service)

	organizationService := NewOrganizationService(authService)
	organizationPb.RegisterOrganizationsServer(grpcServer, organizationService)

	userService := NewUsersService(authService)
	pbUsers.RegisterUsersServer(grpcServer, userService)

	groupService := NewGroupsService(authService)
	pbGroups.RegisterGroupsServer(grpcServer, groupService)

	roleService := NewRoleService(authService)
	pbRoles.RegisterRolesServer(grpcServer, roleService)

	secretsService := NewSecretService(encryptor, authService)
	secretPb.RegisterSecretsServer(grpcServer, secretsService)

	integrationsService := NewIntegrationService(encryptor, authService, registry)
	integrationPb.RegisterIntegrationsServer(grpcServer, integrationsService)

	meService := NewMeService()
	mepb.RegisterMeServer(grpcServer, meService)

	primitiveService := NewPrimitiveService(registry)
	pbPrimitives.RegisterPrimitivesServer(grpcServer, primitiveService)
	blueprintService := NewBlueprintService(registry)
	pbBlueprints.RegisterBlueprintsServer(grpcServer, blueprintService)
	workflowService := NewWorkflowService(registry)
	pbWorkflows.RegisterWorkflowsServer(grpcServer, workflowService)

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
