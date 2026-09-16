package factories

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func StartFactoryAgentResourceOAuth(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.StartFactoryAgentResourceOAuthRequest,
) (*pb.StartFactoryAgentResourceOAuthResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to start MCP OAuth")
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	connectedBy, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to start MCP OAuth")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}
	if resource.Config.Data().MCPAuth() != models.FactoryAgentResourceAuthOAuth {
		return nil, factoryErrorToStatus(invalidArgument("this connection uses a header, not sign-in"), "failed to start MCP OAuth")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(deps.WebhookBaseURL), "/")
	if baseURL == "" {
		return nil, factoryErrorToStatus(invalidArgument("public SuperPlane URL is missing"), "failed to start MCP OAuth")
	}

	httpClient := mcp.DoerFromCore(deps.Registry.HTTPContext())
	oauthCtx, cancel := mcp.TimeoutContext(ctx)
	defer cancel()

	discovery, err := mcp.Discover(oauthCtx, httpClient, resource.Config.Data().URL)
	if err != nil {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthVendorRejected, mcp.UserFacingOAuthError(err), nil)
		return nil, factoryErrorToStatus(invalidArgument(mcp.UserFacingOAuthError(err)), "failed to start MCP OAuth")
	}

	redirectURI := mcp.CallbackURL(baseURL)
	clientMetadataURL := mcp.ClientMetadataURL(baseURL)
	clientID, useDCR, err := discovery.SelectClientID(clientMetadataURL)
	if err != nil {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthVendorRejected, mcp.UserFacingOAuthError(err), nil)
		return nil, factoryErrorToStatus(invalidArgument(mcp.UserFacingOAuthError(err)), "failed to start MCP OAuth")
	}

	var clientSecret string
	if useDCR {
		registration, err := mcp.RegisterClient(oauthCtx, httpClient, discovery.AuthServer.RegistrationEndpoint, redirectURI, clientMetadataURL)
		if err != nil {
			_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthVendorRejected, mcp.UserFacingOAuthError(err), nil)
			return nil, factoryErrorToStatus(invalidArgument(mcp.UserFacingOAuthError(err)), "failed to start MCP OAuth")
		}
		clientID = registration.ClientID
		clientSecret = registration.ClientSecret
	}

	pkce, err := mcp.NewPKCE()
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}
	state, err := mcp.RandomState()
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}

	if err := persistOAuthStart(ctx, db, deps.Encryptor, resource, discovery, clientID, clientSecret, pkce.Verifier, state, connectedBy); err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}

	authURL, err := mcp.AuthorizationURL(
		discovery.AuthServer.AuthorizationEndpoint,
		clientID,
		redirectURI,
		state,
		discovery.Resource,
		pkce,
	)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to start MCP OAuth")
	}

	return &pb.StartFactoryAgentResourceOAuthResponse{
		AuthorizationUrl: authURL,
		Resource:         serializeFactoryAgentResource(resource),
	}, nil
}

func DisconnectFactoryAgentResourceOAuth(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.DisconnectFactoryAgentResourceOAuthRequest,
) (*pb.DisconnectFactoryAgentResourceOAuthResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to disconnect MCP OAuth")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}

	httpClient := mcp.DoerFromCore(deps.Registry.HTTPContext())
	oauthCtx, cancel := mcp.TimeoutContext(ctx)
	defer cancel()
	revokeStoredOAuthTokens(oauthCtx, httpClient, deps.Encryptor, db, resource)
	if err := resource.DeleteSecrets(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}
	if err := resource.ClearOAuthPending(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}
	if err := resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNotConnected, "", nil); err != nil {
		return nil, factoryErrorToStatus(err, "failed to disconnect MCP OAuth")
	}
	return &pb.DisconnectFactoryAgentResourceOAuthResponse{Resource: serializeFactoryAgentResource(resource)}, nil
}

func persistOAuthStart(
	ctx context.Context,
	db *gorm.DB,
	encryptor crypto.Encryptor,
	resource *models.FactoryAgentResource,
	discovery *mcp.Discovery,
	clientID, clientSecret, verifier, state string,
	connectedBy uuid.UUID,
) error {
	if err := resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		AuthorizationEndpoint: discovery.AuthServer.AuthorizationEndpoint,
		TokenEndpoint:         discovery.AuthServer.TokenEndpoint,
		RevocationEndpoint:    discovery.AuthServer.RevocationEndpoint,
		RegistrationEndpoint:  discovery.AuthServer.RegistrationEndpoint,
		ClientID:              clientID,
		Resource:              discovery.Resource,
	}); err != nil {
		return err
	}
	if err := resource.SetOAuthPending(db, state, time.Now().Add(10*time.Minute)); err != nil {
		return err
	}
	encryptedVerifier, err := mcp.EncryptResourceSecret(ctx, encryptor, resource.ID, verifier)
	if err != nil {
		return err
	}
	if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretCodeVerifier, encryptedVerifier); err != nil {
		return err
	}
	if clientSecret != "" {
		encryptedSecret, err := mcp.EncryptResourceSecret(ctx, encryptor, resource.ID, clientSecret)
		if err != nil {
			return err
		}
		if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretClientSecret, encryptedSecret); err != nil {
			return err
		}
	}
	if err := resource.SetOAuthConnector(db, connectedBy); err != nil {
		return err
	}
	return resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNotConnected, "", nil)
}

func FactoryAgentResourceSettingsPath(orgSlug, factoryKey string) string {
	return "/" + strings.Trim(orgSlug, "/") + "/workspaces/" + factoryKey + "/settings/workspace/agent-resources"
}

func CompleteFactoryAgentResourceOAuth(
	ctx context.Context,
	encryptor crypto.Encryptor,
	httpClient mcp.HTTPDoer,
	baseURL, code, state, oauthError string,
) (redirectPath string, statusCode int, message string) {
	db := database.DB(ctx)
	resource, err := models.FindFactoryAgentResourceByOAuthState(db, state)
	if err != nil {
		return "", 404, "agent resource not found"
	}
	factory, err := models.FindFactory(db, resource.OrganizationID, resource.FactoryID)
	if err != nil {
		return "", 404, "workspace not found"
	}
	org, err := models.FindOrganizationByIDInTransaction(db, resource.OrganizationID.String())
	if err != nil {
		return "", 404, "organization not found"
	}
	redirectPath = FactoryAgentResourceSettingsPath(org.Slug, factory.Key)

	if oauthError != "" {
		_ = resource.ClearOAuthPending(db)
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthVendorRejected, oauthError, nil)
		return redirectPath, 302, ""
	}
	if strings.TrimSpace(code) == "" {
		_ = resource.ClearOAuthPending(db)
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, "the authorization server did not return a code", nil)
		return redirectPath, 302, ""
	}

	verifier, err := mcp.DecryptedResourceSecret(ctx, encryptor, db, resource, models.FactoryAgentResourceSecretCodeVerifier)
	if err != nil || verifier == "" {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, "the sign-in session expired", nil)
		return redirectPath, 302, ""
	}
	clientSecret, _ := mcp.DecryptedResourceSecret(ctx, encryptor, db, resource, models.FactoryAgentResourceSecretClientSecret)
	metadata := resource.OAuthMetadata.Data()
	oauthCtx, cancel := mcp.TimeoutContext(ctx)
	defer cancel()
	tokens, err := mcp.ExchangeCode(
		oauthCtx,
		httpClient,
		metadata.TokenEndpoint,
		metadata.ClientID,
		clientSecret,
		mcp.CallbackURL(baseURL),
		code,
		verifier,
		metadata.Resource,
	)
	if err != nil {
		_ = resource.ClearOAuthPending(db)
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, mcp.UserFacingOAuthError(err), nil)
		return redirectPath, 302, ""
	}
	if err := mcp.StoreOAuthTokens(ctx, encryptor, db, resource, tokens); err != nil {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, "SuperPlane could not store the access token", nil)
		return redirectPath, 302, ""
	}
	_ = resource.DeleteSecret(db, models.FactoryAgentResourceSecretCodeVerifier)
	_ = resource.ClearOAuthPending(db)
	_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", resource.OAuthConnectedBy)
	return redirectPath, 302, ""
}

func revokeStoredOAuthTokens(
	ctx context.Context,
	httpClient mcp.HTTPDoer,
	encryptor crypto.Encryptor,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) {
	metadata := resource.OAuthMetadata.Data()
	if metadata.RevocationEndpoint == "" {
		return
	}
	refresh, _ := mcp.DecryptedResourceSecret(ctx, encryptor, db, resource, models.FactoryAgentResourceSecretRefreshToken)
	if refresh != "" {
		mcp.RevokeToken(ctx, httpClient, metadata.RevocationEndpoint, metadata.ClientID, refresh)
	}
}
