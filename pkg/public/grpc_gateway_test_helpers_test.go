package public

import (
	"testing"

	"github.com/stretchr/testify/require"
	billingorganizations "github.com/superplanehq/superplane/ee/billing/organizations"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
)

func testGRPCServices(
	t *testing.T,
	authService authorization.Authorization,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	oidcProvider oidc.Provider,
) *grpc.Services {
	t.Helper()

	features := enterprise.NewRegistry()
	features.Register(enterprise.Billing, billingorganizations.New())
	services, err := grpc.NewServices(grpc.ServicesConfig{
		Enterprise:       features,
		BaseURL:          "http://localhost",
		WebhooksBaseURL:  "http://localhost",
		RunnerAPIBaseURL: "http://localhost",
		Encryptor:        encryptor,
		AuthService:      authService,
		Registry:         registry,
		OIDCProvider:     oidcProvider,
		JWTSigner:        jwt.NewSigner("test-runner-registration-secret"),
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
) {
	t.Helper()
	require.NoError(t, server.RegisterGRPCGateway(testGRPCServices(
		t,
		authService,
		registry,
		encryptor,
		oidcProvider,
	)))
}
