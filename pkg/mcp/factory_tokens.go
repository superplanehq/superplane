package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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

const factoryAgentResourceRefreshLockPrefix = "factory-mcp-oauth-refresh:"

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
	if refresh := strings.TrimSpace(tokens.RefreshToken); refresh != "" {
		encrypted, err := EncryptResourceSecret(ctx, encryptor, resource.ID, refresh)
		if err != nil {
			return err
		}
		if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, encrypted); err != nil {
			return err
		}
	}
	access, err := EncryptResourceSecret(ctx, encryptor, resource.ID, tokens.AccessToken)
	if err != nil {
		return err
	}
	if err := resource.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, access); err != nil {
		return err
	}
	metadata := resource.OAuthMetadata.Data()
	metadata.AccessTokenExpiresAt = time.Now().Add(TokenTTL(tokens)).UTC().Format(time.RFC3339)
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

// MintFactoryAgentResourceAccessToken returns a usable OAuth access token for a workspace MCP server.
// It reuses the stored access token until shortly before expiry, and refreshes under a database lock.
func MintFactoryAgentResourceAccessToken(
	ctx context.Context,
	encryptor crypto.Encryptor,
	httpClient HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) (string, error) {
	return mintFactoryAgentResourceAccessToken(ctx, encryptor, httpClient, db, resource, accessTokenRefreshSkew)
}

// MintFactoryAgentResourceAccessTokenWithLifetime returns a token that remains
// valid for at least minRemaining. Agent runs pass the execution timeout
// because they copy the token into fixed headers for the whole run.
func MintFactoryAgentResourceAccessTokenWithLifetime(
	ctx context.Context,
	encryptor crypto.Encryptor,
	httpClient HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	minRemaining time.Duration,
) (string, error) {
	if minRemaining < accessTokenRefreshSkew {
		minRemaining = accessTokenRefreshSkew
	}
	return mintFactoryAgentResourceAccessToken(ctx, encryptor, httpClient, db, resource, minRemaining)
}

func mintFactoryAgentResourceAccessToken(
	ctx context.Context,
	encryptor crypto.Encryptor,
	httpClient HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
	minRemaining time.Duration,
) (string, error) {
	if resource.Config.Data().MCPAuth() != models.FactoryAgentResourceAuthOAuth {
		return "", nil
	}
	unlock := lockFactoryAgentResourceRefresh(resource.ID)
	defer unlock()

	var accessToken string
	var refreshFailure error
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockFactoryAgentResourceRefreshTx(tx, resource.ID); err != nil {
			return err
		}
		if err := reloadFactoryAgentResource(tx, resource); err != nil {
			return err
		}

		now := time.Now()
		expiresAt := accessTokenExpiresAt(resource)
		currentAccess, err := DecryptedResourceSecret(ctx, encryptor, tx, resource, models.FactoryAgentResourceSecretAccessToken)
		if err != nil {
			return err
		}
		if currentAccess != "" && accessTokenCurrent(expiresAt, now, minRemaining) {
			accessToken = currentAccess
			return markOAuthConnected(tx, resource)
		}

		refresh, err := DecryptedResourceSecret(ctx, encryptor, tx, resource, models.FactoryAgentResourceSecretRefreshToken)
		if err != nil {
			return err
		}
		if refresh == "" {
			if err := resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthNeedsReconnect, "sign in again", nil); err != nil {
				return err
			}
			refreshFailure = fmt.Errorf("MCP connection %s is not signed in", resource.Name)
			return nil
		}

		metadata := resource.OAuthMetadata.Data()
		clientSecret, _ := DecryptedResourceSecret(ctx, encryptor, tx, resource, models.FactoryAgentResourceSecretClientSecret)
		oauthCtx, cancel := TimeoutContext(ctx)
		defer cancel()
		tokens, err := RefreshAccessToken(oauthCtx, httpClient, metadata.TokenEndpoint, metadata.ClientID, clientSecret, refresh, metadata.Resource)
		if err != nil {
			minted, err := finishRefreshError(ctx, encryptor, tx, resource, refresh, currentAccess, expiresAt, err)
			if err != nil {
				return err
			}
			accessToken = minted.accessToken
			refreshFailure = minted.failure
			return nil
		}
		if err := StoreOAuthTokens(ctx, encryptor, tx, resource, tokens); err != nil {
			return err
		}
		if err := markOAuthConnected(tx, resource); err != nil {
			return err
		}
		accessToken = tokens.AccessToken
		return nil
	})
	if err != nil {
		return "", err
	}
	if refreshFailure != nil {
		return "", refreshFailure
	}
	return accessToken, nil
}

type mintedToken struct {
	accessToken string
	failure     error
}

func finishRefreshError(
	ctx context.Context,
	encryptor crypto.Encryptor,
	tx *gorm.DB,
	resource *models.FactoryAgentResource,
	refresh string,
	currentAccess string,
	expiresAt time.Time,
	cause error,
) (mintedToken, error) {
	var endpointErr *TokenEndpointError
	if !errors.As(cause, &endpointErr) || !endpointErr.RejectsGrant() {
		if currentAccess != "" && accessTokenUnexpired(expiresAt, time.Now()) {
			return mintedToken{accessToken: currentAccess}, nil
		}
		return mintedToken{}, cause
	}

	latestRefresh, err := DecryptedResourceSecret(ctx, encryptor, tx, resource, models.FactoryAgentResourceSecretRefreshToken)
	if err != nil {
		return mintedToken{}, err
	}
	if latestRefresh != refresh {
		return adoptRotatedAccessToken(ctx, encryptor, tx, resource, cause)
	}

	if err := resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthNeedsReconnect, UserFacingOAuthError(cause), nil); err != nil {
		return mintedToken{}, err
	}
	return mintedToken{failure: cause}, nil
}

func adoptRotatedAccessToken(
	ctx context.Context,
	encryptor crypto.Encryptor,
	tx *gorm.DB,
	resource *models.FactoryAgentResource,
	cause error,
) (mintedToken, error) {
	latestAccess, err := DecryptedResourceSecret(ctx, encryptor, tx, resource, models.FactoryAgentResourceSecretAccessToken)
	if err != nil {
		return mintedToken{}, err
	}
	if latestAccess == "" {
		return mintedToken{}, cause
	}
	if err := markOAuthConnected(tx, resource); err != nil {
		return mintedToken{}, err
	}
	return mintedToken{accessToken: latestAccess}, nil
}

func markOAuthConnected(tx *gorm.DB, resource *models.FactoryAgentResource) error {
	if resource.OAuthStatus == models.FactoryAgentResourceOAuthConnected {
		return nil
	}
	return resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthConnected, "", resource.OAuthConnectedBy)
}

func reloadFactoryAgentResource(tx *gorm.DB, resource *models.FactoryAgentResource) error {
	var fresh models.FactoryAgentResource
	err := tx.Where("id = ?", resource.ID).First(&fresh).Error
	if err != nil {
		return err
	}
	resource.OAuthStatus = fresh.OAuthStatus
	resource.OAuthError = fresh.OAuthError
	resource.OAuthMetadata = fresh.OAuthMetadata
	resource.OAuthConnectedBy = fresh.OAuthConnectedBy
	resource.OAuthConnectedAt = fresh.OAuthConnectedAt
	return nil
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

func lockFactoryAgentResourceRefreshTx(tx *gorm.DB, id uuid.UUID) error {
	return tx.Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		factoryAgentResourceRefreshLockPrefix+id.String(),
	).Error
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
