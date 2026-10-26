package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrFactoryAgentResourceRefreshBusy   = models.ErrFactoryAgentResourceRefreshBusy
	ErrFactoryAgentResourceChanged       = errors.New("MCP OAuth connection changed")
	ErrFactoryAgentResourceTokenLifetime = errors.New("MCP access token does not remain valid for the runner execution timeout")
)

const factoryOAuthRefreshLease = 30 * time.Second

type FactoryAgentResourceRefreshStorageError struct{ cause error }

func (e *FactoryAgentResourceRefreshStorageError) Error() string {
	return "MCP OAuth storage is temporarily unavailable"
}
func (e *FactoryAgentResourceRefreshStorageError) Unwrap() error { return e.cause }

// MintFactoryAgentResourceAccessToken reuses the current access token and waits
// outside database transactions when another gateway is refreshing the grant.
// The DB handle must not be an enclosing transaction.
func MintFactoryAgentResourceAccessToken(ctx context.Context, encryptor crypto.Encryptor, httpClient HTTPDoer, db *gorm.DB, resource *models.FactoryAgentResource) (string, error) {
	return MintFactoryAgentResourceAccessTokenWithLifetime(ctx, encryptor, httpClient, db, resource, accessTokenRefreshSkew)
}

func MintFactoryAgentResourceAccessTokenWithLifetime(ctx context.Context, encryptor crypto.Encryptor, httpClient HTTPDoer, db *gorm.DB, resource *models.FactoryAgentResource, minRemaining time.Duration) (string, error) {
	minter := newFactoryTokenMinter(ctx, encryptor, httpClient, db, resource, minRemaining)
	for {
		token, err := minter.mint(resource)
		if !errors.Is(err, ErrFactoryAgentResourceRefreshBusy) {
			return token, err
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

// MintFactoryAgentResourceAccessTokenForRunner never waits for another job's
// rotating grant. The durable runner attempt can retry a busy lease later.
func MintFactoryAgentResourceAccessTokenForRunner(ctx context.Context, encryptor crypto.Encryptor, httpClient HTTPDoer, db *gorm.DB, resource *models.FactoryAgentResource, minRemaining time.Duration) (string, error) {
	minter := newFactoryTokenMinter(ctx, encryptor, httpClient, db, resource, RunnerAccessTokenMinimumLifetime(minRemaining))
	minter.fallbackLifetime = minter.minRemaining
	return minter.mint(resource)
}

// RunnerAccessTokenMinimumLifetime allows one minute for provider clock
// granularity and startup, while preserving the existing refresh skew. A fresh
// one-hour grant supports a one-hour job with a short delivery window.
func RunnerAccessTokenMinimumLifetime(executionTimeout time.Duration) time.Duration {
	return max(executionTimeout-time.Minute, accessTokenRefreshSkew)
}

type factoryTokenMinter struct {
	ctx              context.Context
	encryptor        crypto.Encryptor
	httpClient       HTTPDoer
	db               *gorm.DB
	pinned           models.FactoryAgentResource
	minRemaining     time.Duration
	fallbackLifetime time.Duration
}

type oauthRefreshSnapshot struct {
	resource *models.FactoryAgentResource
	secrets  map[string][]byte
}

func newFactoryTokenMinter(ctx context.Context, encryptor crypto.Encryptor, httpClient HTTPDoer, db *gorm.DB, resource *models.FactoryAgentResource, minRemaining time.Duration) *factoryTokenMinter {
	return &factoryTokenMinter{ctx: ctx, encryptor: encryption.ForDB(db, encryptor), httpClient: httpClient, db: db.WithContext(ctx), pinned: *resource, minRemaining: max(minRemaining, accessTokenRefreshSkew)}
}

func (m *factoryTokenMinter) mint(resource *models.FactoryAgentResource) (string, error) {
	if m.pinned.Config.Data().MCPAuth() != models.FactoryAgentResourceAuthOAuth {
		return "", nil
	}
	lease, snapshot, err := m.claim()
	if err != nil {
		return "", err
	}
	defer func() { _ = lease.Release(m.db.Session(&gorm.Session{NewDB: true})) }()
	currentAccess, err := m.decrypt(snapshot, models.FactoryAgentResourceSecretAccessToken)
	if err != nil {
		return "", err
	}
	if currentAccess != "" && accessTokenCurrent(accessTokenExpiresAt(snapshot.resource), time.Now(), m.minRemaining) {
		if err := m.finishCached(lease, snapshot); err != nil {
			return "", err
		}
		*resource = *snapshot.resource
		return currentAccess, nil
	}
	refresh, err := m.decrypt(snapshot, models.FactoryAgentResourceSecretRefreshToken)
	if err != nil {
		return "", err
	}
	if refresh == "" {
		cause := fmt.Errorf("MCP connection %s is not signed in", snapshot.resource.Name)
		return "", m.rejectMissingGrant(lease, snapshot, cause)
	}
	clientSecret, err := m.decrypt(snapshot, models.FactoryAgentResourceSecretClientSecret)
	if err != nil {
		return "", err
	}
	// Always use the caller's pinned recipient. A metadata replacement between
	// validation and this claim can never redirect the refresh credential.
	metadata := m.pinned.OAuthMetadata.Data()
	oauthCtx, cancel := TimeoutContext(m.ctx)
	tokens, refreshErr := RefreshAccessToken(oauthCtx, m.httpClient, metadata.TokenEndpoint, metadata.ClientID, clientSecret, refresh, metadata.Resource)
	cancel()
	if refreshErr != nil {
		token, fresh, err := m.finishRefreshError(lease, snapshot, currentAccess, refreshErr)
		if err == nil {
			*resource = *fresh.resource
		}
		return token, err
	}
	encrypted, err := encryptOAuthTokens(m.ctx, m.encryptor, snapshot.resource, tokens)
	if err != nil {
		return "", err
	}
	fresh, adopted, err := m.commitTokens(lease, snapshot, encrypted)
	if err != nil {
		return "", err
	}
	*resource = *fresh.resource
	if adopted {
		return m.adoptAccess(fresh, ErrFactoryAgentResourceRefreshBusy)
	}
	if m.fallbackLifetime > 0 && !accessTokenCurrent(accessTokenExpiresAt(fresh.resource), time.Now(), m.minRemaining) {
		// Preserve the rotated grant even when its token cannot support this
		// job. Repeated minting cannot extend a provider's fixed token TTL.
		return "", ErrFactoryAgentResourceTokenLifetime
	}
	return tokens.AccessToken, nil
}

func (m *factoryTokenMinter) claim() (*models.FactoryAgentResourceRefresh, *oauthRefreshSnapshot, error) {
	var lease *models.FactoryAgentResourceRefresh
	var snapshot *oauthRefreshSnapshot
	err := m.transaction(func(tx *gorm.DB) error {
		var err error
		snapshot, err = m.lockSnapshot(tx)
		if err != nil {
			return err
		}
		now := time.Now()
		expires := now.Add(factoryOAuthRefreshLease)
		if deadline, ok := m.ctx.Deadline(); ok && deadline.Before(expires) {
			expires = deadline
		}
		lease, err = snapshot.resource.ClaimOAuthRefresh(tx, now, expires)
		return err
	})
	return lease, snapshot, err
}

func (m *factoryTokenMinter) lockSnapshot(tx *gorm.DB) (*oauthRefreshSnapshot, error) {
	fresh, secrets, err := m.pinned.LockOAuthRefreshSnapshot(tx)
	if err != nil {
		return nil, err
	}
	if !sameOAuthRecipient(&m.pinned, fresh) {
		return nil, ErrFactoryAgentResourceChanged
	}
	snapshot := &oauthRefreshSnapshot{resource: fresh, secrets: make(map[string][]byte, len(secrets))}
	for _, secret := range secrets {
		snapshot.secrets[secret.Name] = secret.Value
	}
	return snapshot, nil
}

func (m *factoryTokenMinter) finishCached(lease *models.FactoryAgentResourceRefresh, saved *oauthRefreshSnapshot) error {
	return m.transaction(func(tx *gorm.DB) error {
		fresh, err := m.lockSnapshot(tx)
		if err != nil {
			return err
		}
		if err := lease.Lock(tx, time.Now()); err != nil {
			return err
		}
		if !sameOAuthSecrets(saved, fresh) {
			return ErrFactoryAgentResourceRefreshBusy
		}
		if err := markOAuthConnected(tx, fresh.resource); err != nil {
			return err
		}
		saved.resource = fresh.resource
		return lease.Release(tx)
	})
}

func (m *factoryTokenMinter) rejectMissingGrant(lease *models.FactoryAgentResourceRefresh, saved *oauthRefreshSnapshot, cause error) error {
	err := m.transaction(func(tx *gorm.DB) error {
		fresh, err := m.lockSnapshot(tx)
		if err != nil {
			return err
		}
		if err := lease.Lock(tx, time.Now()); err != nil {
			return err
		}
		if !sameOAuthSecrets(saved, fresh) {
			return ErrFactoryAgentResourceRefreshBusy
		}
		if err := fresh.resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthNeedsReconnect, "sign in again", nil); err != nil {
			return err
		}
		return lease.Release(tx)
	})
	if err != nil {
		return err
	}
	return cause
}

func (m *factoryTokenMinter) commitTokens(lease *models.FactoryAgentResourceRefresh, saved *oauthRefreshSnapshot, tokens *encryptedOAuthTokens) (*oauthRefreshSnapshot, bool, error) {
	var fresh *oauthRefreshSnapshot
	var adopted bool
	err := m.transaction(func(tx *gorm.DB) error {
		var err error
		fresh, err = m.lockSnapshot(tx)
		if err != nil {
			return err
		}
		if err := lease.Lock(tx, time.Now()); err != nil {
			return err
		}
		if !sameOAuthSecrets(saved, fresh) {
			adopted = true
			if err := markOAuthConnected(tx, fresh.resource); err != nil {
				return err
			}
			return lease.Release(tx)
		}
		if err := storeEncryptedOAuthTokens(tx, fresh.resource, tokens); err != nil {
			return err
		}
		if err := markOAuthConnected(tx, fresh.resource); err != nil {
			return err
		}
		return lease.Release(tx)
	})
	return fresh, adopted, err
}

func (m *factoryTokenMinter) finishRefreshError(lease *models.FactoryAgentResourceRefresh, saved *oauthRefreshSnapshot, currentAccess string, cause error) (string, *oauthRefreshSnapshot, error) {
	var endpointErr *TokenEndpointError
	if !errors.As(cause, &endpointErr) || !endpointErr.RejectsGrant() {
		if currentAccess != "" && accessTokenUnexpired(accessTokenExpiresAt(saved.resource).Add(-m.fallbackLifetime), time.Now()) {
			if err := m.finishCached(lease, saved); err != nil {
				return "", nil, err
			}
			return currentAccess, saved, nil
		}
		return "", nil, cause
	}
	var fresh *oauthRefreshSnapshot
	var rotated bool
	err := m.transaction(func(tx *gorm.DB) error {
		var err error
		fresh, err = m.lockSnapshot(tx)
		if err != nil {
			return err
		}
		if err := lease.Lock(tx, time.Now()); err != nil {
			return err
		}
		rotated = !bytes.Equal(saved.secrets[models.FactoryAgentResourceSecretRefreshToken], fresh.secrets[models.FactoryAgentResourceSecretRefreshToken])
		if !rotated && !sameOAuthSecrets(saved, fresh) {
			return ErrFactoryAgentResourceRefreshBusy
		}
		if !rotated {
			if err := fresh.resource.SetOAuthStatus(tx, models.FactoryAgentResourceOAuthNeedsReconnect, UserFacingOAuthError(cause), nil); err != nil {
				return err
			}
		} else if err := markOAuthConnected(tx, fresh.resource); err != nil {
			return err
		}
		return lease.Release(tx)
	})
	if err != nil {
		return "", nil, err
	}
	if !rotated {
		return "", nil, cause
	}
	token, err := m.adoptAccess(fresh, cause)
	return token, fresh, err
}

func (m *factoryTokenMinter) adoptAccess(snapshot *oauthRefreshSnapshot, cause error) (string, error) {
	access, err := m.decrypt(snapshot, models.FactoryAgentResourceSecretAccessToken)
	if err != nil {
		return "", err
	}
	if access == "" || (m.fallbackLifetime > 0 && !accessTokenCurrent(accessTokenExpiresAt(snapshot.resource), time.Now(), m.fallbackLifetime)) {
		return "", cause
	}
	return access, nil
}

func (m *factoryTokenMinter) decrypt(snapshot *oauthRefreshSnapshot, name string) (string, error) {
	value := snapshot.secrets[name]
	if len(value) == 0 {
		return "", nil
	}
	return DecryptResourceSecret(m.ctx, m.encryptor, snapshot.resource.ID, value)
}

func (m *factoryTokenMinter) transaction(run func(*gorm.DB) error) error {
	err := m.db.Transaction(run)
	if err == nil || errors.Is(err, ErrFactoryAgentResourceRefreshBusy) || errors.Is(err, ErrFactoryAgentResourceChanged) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, models.ErrFactoryAgentResourceURLTaken) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, models.ErrFactoryAgentResourceNotFound) {
		return ErrFactoryAgentResourceChanged
	}
	return &FactoryAgentResourceRefreshStorageError{cause: err}
}

func sameOAuthSecrets(saved, fresh *oauthRefreshSnapshot) bool {
	for _, name := range []string{models.FactoryAgentResourceSecretRefreshToken, models.FactoryAgentResourceSecretAccessToken, models.FactoryAgentResourceSecretClientSecret} {
		if !bytes.Equal(saved.secrets[name], fresh.secrets[name]) {
			return false
		}
	}
	return true
}

func sameOAuthRecipient(saved, fresh *models.FactoryAgentResource) bool {
	previous, current := saved.OAuthMetadata.Data(), fresh.OAuthMetadata.Data()
	previous.AccessTokenExpiresAt, current.AccessTokenExpiresAt = "", ""
	return saved.ID == fresh.ID && saved.OrganizationID == fresh.OrganizationID && saved.FactoryID == fresh.FactoryID &&
		saved.Kind == fresh.Kind && saved.Enabled == fresh.Enabled && saved.Config.Data().URL == fresh.Config.Data().URL &&
		saved.Config.Data().MCPAuth() == fresh.Config.Data().MCPAuth() && previous == current &&
		saved.OAuthStatus == fresh.OAuthStatus && sameOAuthConnectedAt(saved, fresh) && sameOAuthConnector(saved, fresh)
}

func sameOAuthConnectedAt(saved, fresh *models.FactoryAgentResource) bool {
	if saved.OAuthConnectedAt == nil || fresh.OAuthConnectedAt == nil {
		return saved.OAuthConnectedAt == nil && fresh.OAuthConnectedAt == nil
	}
	return saved.OAuthConnectedAt.Truncate(time.Microsecond).Equal(fresh.OAuthConnectedAt.Truncate(time.Microsecond))
}

func sameOAuthConnector(saved, fresh *models.FactoryAgentResource) bool {
	if saved.OAuthConnectedBy == nil || fresh.OAuthConnectedBy == nil {
		return saved.OAuthConnectedBy == nil && fresh.OAuthConnectedBy == nil
	}
	return *saved.OAuthConnectedBy == *fresh.OAuthConnectedBy
}
