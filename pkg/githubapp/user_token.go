package githubapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const userAccessTokenRefreshMargin = 5 * time.Minute

var ErrUserAuthorizationNotUsable = errors.New("GitHub authorization is missing, expired, or for another app")

// UserAccessToken is a usable token for one member.
type UserAccessToken struct {
	Token string
	Login string
}

// SaveUserAuthorization stores an encrypted grant for the account.
func SaveUserAuthorization(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	accountID uuid.UUID,
	appID int64,
	user *GitHubUser,
	tokens *UserTokens,
) error {
	encryptedAccessToken, encryptedRefreshToken, err := encryptUserTokens(ctx, tx, encryptor, accountID, tokens)
	if err != nil {
		return err
	}
	return models.SaveAccountGitHubAuthorization(tx, &models.AccountGitHubAuthorization{
		AccountID:             accountID,
		GitHubAppID:           appID,
		GitHubUserID:          user.UserIDString(),
		GitHubLogin:           user.Login,
		EncryptedAccessToken:  encryptedAccessToken,
		AccessTokenExpiresAt:  tokens.AccessTokenExpiresAt,
		EncryptedRefreshToken: encryptedRefreshToken,
		RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt,
	})
}

// ResolveUserAccessToken returns a token that is valid for at least a few
// minutes. It refreshes an expiring token under a row lock. A rejected
// refresh token invalidates the grant.
func ResolveUserAccessToken(
	ctx context.Context,
	db *gorm.DB,
	encryptor crypto.Encryptor,
	httpCtx core.HTTPContext,
	cfg config.GitHubHostedAppConfig,
	accountID uuid.UUID,
) (*UserAccessToken, error) {
	if !cfg.UserAuthorizationEnabled() {
		return nil, ErrUserAuthorizationUnavailable
	}

	var resolved *UserAccessToken
	err := db.Transaction(func(tx *gorm.DB) error {
		authorization, err := models.LockAccountGitHubAuthorization(tx, accountID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if !authorization.UsableFor(cfg.ID) {
			return nil
		}

		scoped := encryption.ForDB(tx, encryptor)
		if accessTokenFresh(authorization) {
			token, err := scoped.Decrypt(ctx, authorization.EncryptedAccessToken, userTokenAssociatedData(accountID))
			if err != nil {
				return fmt.Errorf("decrypt GitHub user access token: %w", err)
			}
			resolved = &UserAccessToken{Token: string(token), Login: authorization.GitHubLogin}
			return nil
		}

		// An invalidated grant must commit, so this path returns no error.
		tokens, err := refreshAuthorization(ctx, tx, scoped, httpCtx, cfg, authorization)
		if err != nil || tokens == nil {
			return err
		}
		resolved = &UserAccessToken{Token: tokens.AccessToken, Login: authorization.GitHubLogin}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, ErrUserAuthorizationNotUsable
	}
	return resolved, nil
}

// DeleteUserAuthorization removes the grant, then revokes the token at GitHub.
// It returns the revoke error separately, so a failed revoke never keeps the
// grant.
func DeleteUserAuthorization(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	httpCtx core.HTTPContext,
	cfg config.GitHubHostedAppConfig,
	accountID uuid.UUID,
) (revokeErr error, err error) {
	authorization, err := models.FindAccountGitHubAuthorization(tx, accountID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := models.DeleteAccountGitHubAuthorization(tx, accountID); err != nil {
		return nil, err
	}
	if authorization.GitHubAppID != cfg.ID || authorization.InvalidatedAt != nil {
		return nil, nil
	}

	token, err := encryption.ForDB(tx, encryptor).Decrypt(ctx, authorization.EncryptedAccessToken, userTokenAssociatedData(accountID))
	if err != nil {
		return err, nil
	}
	return RevokeUserToken(httpCtx, cfg, string(token)), nil
}

// refreshAuthorization returns nil tokens and no error when it invalidates the
// grant.
func refreshAuthorization(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	httpCtx core.HTTPContext,
	cfg config.GitHubHostedAppConfig,
	authorization *models.AccountGitHubAuthorization,
) (*UserTokens, error) {
	if len(authorization.EncryptedRefreshToken) == 0 || refreshTokenExpired(authorization) {
		return nil, authorization.Invalidate(tx)
	}

	refreshToken, err := encryptor.Decrypt(ctx, authorization.EncryptedRefreshToken, userTokenAssociatedData(authorization.AccountID))
	if err != nil {
		return nil, fmt.Errorf("decrypt GitHub user refresh token: %w", err)
	}

	tokens, err := RefreshUserTokens(httpCtx, cfg, string(refreshToken))
	if errors.Is(err, ErrBadRefreshToken) {
		return nil, authorization.Invalidate(tx)
	}
	if err != nil {
		return nil, err
	}

	encryptedAccessToken, encryptedRefreshToken, err := encryptUserTokens(ctx, tx, encryptor, authorization.AccountID, tokens)
	if err != nil {
		return nil, err
	}
	err = authorization.UpdateTokens(
		tx,
		encryptedAccessToken,
		tokens.AccessTokenExpiresAt,
		encryptedRefreshToken,
		tokens.RefreshTokenExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func encryptUserTokens(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	accountID uuid.UUID,
	tokens *UserTokens,
) ([]byte, []byte, error) {
	scoped := encryption.ForDB(tx, encryptor)
	associatedData := userTokenAssociatedData(accountID)

	accessToken, err := scoped.Encrypt(ctx, []byte(tokens.AccessToken), associatedData)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt GitHub user access token: %w", err)
	}
	if tokens.RefreshToken == "" {
		return accessToken, []byte{}, nil
	}
	refreshToken, err := scoped.Encrypt(ctx, []byte(tokens.RefreshToken), associatedData)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt GitHub user refresh token: %w", err)
	}
	return accessToken, refreshToken, nil
}

func accessTokenFresh(authorization *models.AccountGitHubAuthorization) bool {
	if authorization.AccessTokenExpiresAt == nil {
		return true
	}
	return time.Until(*authorization.AccessTokenExpiresAt) > userAccessTokenRefreshMargin
}

func refreshTokenExpired(authorization *models.AccountGitHubAuthorization) bool {
	if authorization.RefreshTokenExpiresAt == nil {
		return false
	}
	return !time.Now().Before(*authorization.RefreshTokenExpiresAt)
}

func userTokenAssociatedData(accountID uuid.UUID) []byte {
	return []byte("account_github_authorization:" + accountID.String())
}
