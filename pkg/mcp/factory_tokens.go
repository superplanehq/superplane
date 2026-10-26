package mcp

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// accessTokenRefreshSkew is how early a refresh replaces the access token.
// Providers such as Sentry keep only the current refresh token and the one
// before it. Refreshing on every use rotates past the token we stored.
const accessTokenRefreshSkew = 2 * time.Minute

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
// The refresh token is written first. A provider that rotates tokens has already
// invalidated the previous refresh token by the time this runs.
func StoreOAuthTokens(
	ctx context.Context,
	encryptor crypto.Encryptor,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	tokens *TokenResponse,
) error {
	encryptor = encryption.ForDB(db, encryptor)
	encrypted, err := encryptOAuthTokens(ctx, encryptor, resource, tokens)
	if err != nil {
		return err
	}
	return storeEncryptedOAuthTokens(db, resource, encrypted)
}

type encryptedOAuthTokens struct {
	refresh   []byte
	access    []byte
	expiresAt time.Time
}

func encryptOAuthTokens(ctx context.Context, encryptor crypto.Encryptor, resource *models.FactoryAgentResource, tokens *TokenResponse) (*encryptedOAuthTokens, error) {
	prepared := &encryptedOAuthTokens{expiresAt: time.Now().Add(TokenTTL(tokens))}
	if refresh := strings.TrimSpace(tokens.RefreshToken); refresh != "" {
		var err error
		prepared.refresh, err = EncryptResourceSecret(ctx, encryptor, resource.ID, refresh)
		if err != nil {
			return nil, err
		}
	}
	access, err := EncryptResourceSecret(ctx, encryptor, resource.ID, tokens.AccessToken)
	if err != nil {
		return nil, err
	}
	prepared.access = access
	return prepared, nil
}

func storeEncryptedOAuthTokens(db *gorm.DB, resource *models.FactoryAgentResource, tokens *encryptedOAuthTokens) error {
	if len(tokens.refresh) > 0 {
		if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, tokens.refresh); err != nil {
			return err
		}
	}
	if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, tokens.access); err != nil {
		return err
	}
	metadata := resource.OAuthMetadata.Data()
	metadata.AccessTokenExpiresAt = tokens.expiresAt.UTC().Format(time.RFC3339)
	return resource.SetOAuthMetadata(db, metadata)
}

// DecryptedResourceSecret returns a named secret or an empty string when missing.
func DecryptedResourceSecret(
	ctx context.Context,
	encryptor crypto.Encryptor,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	name string,
) (string, error) {
	encryptor = encryption.ForDB(db, encryptor)
	secret, err := resource.FindSecret(db, name)
	if err != nil {
		if errors.Is(err, models.ErrFactoryAgentResourceSecretNotFound) {
			return "", nil
		}
		return "", err
	}
	return DecryptResourceSecret(ctx, encryptor, resource.ID, secret.Value)
}

func markOAuthConnected(tx *gorm.DB, resource *models.FactoryAgentResource) error {
	if resource.OAuthStatus == models.FactoryAgentResourceOAuthConnected {
		return nil
	}
	return resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthConnected, "", resource.OAuthConnectedBy)
}

func accessTokenExpiresAt(resource *models.FactoryAgentResource) time.Time {
	raw := strings.TrimSpace(resource.OAuthMetadata.Data().AccessTokenExpiresAt)
	if raw == "" {
		return time.Time{}
	}
	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return expiresAt
}

func accessTokenCurrent(expiresAt, now time.Time, minRemaining time.Duration) bool {
	if minRemaining < accessTokenRefreshSkew {
		minRemaining = accessTokenRefreshSkew
	}
	return !expiresAt.IsZero() && expiresAt.After(now.Add(minRemaining))
}

func accessTokenUnexpired(expiresAt, now time.Time) bool {
	return !expiresAt.IsZero() && expiresAt.After(now)
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
