package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// EncryptResourceSecret encrypts a workspace MCP secret with the resource ID as AAD.
func EncryptResourceSecret(ctx context.Context, encryptor crypto.Encryptor, resourceID uuid.UUID, plaintext string) ([]byte, error) {
	return encryptor.Encrypt(ctx, []byte(plaintext), []byte(resourceID.String()))
}

// DecryptResourceSecret decrypts a workspace MCP secret with the resource ID as AAD.
func DecryptResourceSecret(ctx context.Context, encryptor crypto.Encryptor, resourceID uuid.UUID, value []byte) (string, error) {
	plain, err := encryptor.Decrypt(ctx, value, []byte(resourceID.String()))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// StoreOAuthTokens encrypts and persists the access token and optional refresh token.
func StoreOAuthTokens(
	ctx context.Context,
	encryptor crypto.Encryptor,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	tokens *TokenResponse,
) error {
	access, err := EncryptResourceSecret(ctx, encryptor, resource.ID, tokens.AccessToken)
	if err != nil {
		return err
	}
	if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, access); err != nil {
		return err
	}
	if strings.TrimSpace(tokens.RefreshToken) == "" {
		return nil
	}
	refresh, err := EncryptResourceSecret(ctx, encryptor, resource.ID, tokens.RefreshToken)
	if err != nil {
		return err
	}
	return resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, refresh)
}

// DecryptedResourceSecret returns a named secret or an empty string when missing.
func DecryptedResourceSecret(
	ctx context.Context,
	encryptor crypto.Encryptor,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	name string,
) (string, error) {
	secret, err := resource.FindSecret(db, name)
	if err != nil {
		if errors.Is(err, models.ErrFactoryAgentResourceSecretNotFound) {
			return "", nil
		}
		return "", err
	}
	return DecryptResourceSecret(ctx, encryptor, resource.ID, secret.Value)
}

// MintFactoryAgentResourceAccessToken refreshes the OAuth access token for a workspace MCP server.
func MintFactoryAgentResourceAccessToken(
	ctx context.Context,
	encryptor crypto.Encryptor,
	httpClient HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) (string, error) {
	if resource.Config.Data().MCPAuth() != models.FactoryAgentResourceAuthOAuth {
		return "", nil
	}
	unlock := lockFactoryAgentResourceRefresh(resource.ID)
	defer unlock()

	refresh, err := DecryptedResourceSecret(ctx, encryptor, db, resource, models.FactoryAgentResourceSecretRefreshToken)
	if err != nil {
		return "", err
	}
	if refresh == "" {
		if err := resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, "sign in again", nil); err != nil {
			return "", err
		}
		return "", fmt.Errorf("MCP connection %s is not signed in", resource.Name)
	}
	metadata := resource.OAuthMetadata.Data()
	clientSecret, _ := DecryptedResourceSecret(ctx, encryptor, db, resource, models.FactoryAgentResourceSecretClientSecret)
	oauthCtx, cancel := TimeoutContext(ctx)
	defer cancel()
	tokens, err := RefreshAccessToken(oauthCtx, httpClient, metadata.TokenEndpoint, metadata.ClientID, clientSecret, refresh, metadata.Resource)
	if err != nil {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthNeedsReconnect, UserFacingOAuthError(err), nil)
		return "", err
	}
	if err := StoreOAuthTokens(ctx, encryptor, db, resource, tokens); err != nil {
		return "", err
	}
	if resource.OAuthStatus != models.FactoryAgentResourceOAuthConnected {
		_ = resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", resource.OAuthConnectedBy)
	}
	return tokens.AccessToken, nil
}

// UserFacingOAuthError returns a short error for the MCP servers page.
func UserFacingOAuthError(err error) string {
	if err == nil {
		return "the MCP server refused the SuperPlane OAuth client"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "the MCP server refused the SuperPlane OAuth client"
	}
	return message
}

var factoryAgentResourceRefreshLocks sync.Map

func lockFactoryAgentResourceRefresh(id uuid.UUID) func() {
	value, _ := factoryAgentResourceRefreshLocks.LoadOrStore(id.String(), &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}
