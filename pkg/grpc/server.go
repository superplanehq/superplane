package grpc

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	authorizationProtos "github.com/superplanehq/superplane/pkg/protos/authorization"
	superplaneProtos "github.com/superplanehq/superplane/pkg/protos/superplane"
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

func RunServer(encryptor crypto.Encryptor, port int) {
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
	service := NewDeliveryService(encryptor)
	superplaneProtos.RegisterSuperplaneServer(grpcServer, service)

	authService, err := authorization.NewAuthService()
	if err != nil {
		log.Fatalf("failed to create auth service: %v", err)
	}

	server := NewAuthorizationServer(authService)
	authorizationProtos.RegisterAuthorizationServer(grpcServer, server)

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
