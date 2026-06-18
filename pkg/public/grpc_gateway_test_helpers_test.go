package public

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

func testGRPCServices(
	t *testing.T,
	authService authorization.Authorization,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	oidcProvider oidc.Provider,
	gitProvider git.Provider,
	usageService usage.Service,
) *grpc.Services {
	t.Helper()

	if usageService == nil {
		usageService = &fakePublicUsageService{}
	}

	services, err := grpc.NewServices(grpc.ServicesConfig{
		BaseURL:         "http://localhost",
		WebhooksBaseURL: "http://localhost",
		Encryptor:       encryptor,
		AuthService:     authService,
		Registry:        registry,
		OIDCProvider:    oidcProvider,
		GitProvider:     gitProvider,
		UsageService:    usageService,
	})
	require.NoError(t, err)
	return services
}

func registerTestGRPCGateway(
	t *testing.T,
	server *Server,
	authService authorization.Authorization,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	oidcProvider oidc.Provider,
	gitProvider git.Provider,
	usageService usage.Service,
) {
	t.Helper()
	require.NoError(t, server.RegisterGRPCGateway(testGRPCServices(
		t,
		authService,
		registry,
		encryptor,
		oidcProvider,
		gitProvider,
		usageService,
	)))
}
