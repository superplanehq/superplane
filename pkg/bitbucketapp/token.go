package bitbucketapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	// StaleAfter is how long SuperPlane waits for another Forge delivery
	// before it treats the cached system token as unusable.
	StaleAfter = 3 * time.Hour
	reconnect  = "Reconnect Bitbucket"
)

// ErrReconnect means the cached Forge token is missing, expired, or stale.
var ErrReconnect = errors.New(reconnect)

type SystemTokenSource func(installationID string) (token string, expiresAt time.Time, err error)

var (
	tokenSourceMu sync.RWMutex
	tokenSource   SystemTokenSource
)

// SetSystemTokenSource installs the process-wide reader for Forge system
// tokens. The Bitbucket integration calls it when a runner needs credentials.
func SetSystemTokenSource(source SystemTokenSource) {
	tokenSourceMu.Lock()
	tokenSource = source
	tokenSourceMu.Unlock()
}

// CurrentSystemToken returns the cached bot token for one Forge installation.
func CurrentSystemToken(installationID string) (string, time.Time, error) {
	tokenSourceMu.RLock()
	source := tokenSource
	tokenSourceMu.RUnlock()
	if source == nil {
		return "", time.Time{}, fmt.Errorf("bitbucket forge token source is not configured")
	}
	return source(strings.TrimSpace(installationID))
}

// LoadSystemToken reads and decrypts the token stored for an installation.
func LoadSystemToken(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, installationID string, now time.Time) (string, time.Time, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return "", time.Time{}, fmt.Errorf("forge installation id is required")
	}
	row, err := models.FindBitbucketForgeInstallation(tx, installationID)
	if err != nil {
		return "", time.Time{}, err
	}
	if row.Stale(now) {
		return "", time.Time{}, ErrReconnect
	}
	plain, err := encryptor.Decrypt(ctx, row.SystemToken, []byte(installationID))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("decrypt bitbucket forge token: %w", err)
	}
	token := strings.TrimSpace(string(plain))
	if token == "" {
		return "", time.Time{}, ErrReconnect
	}
	return token, row.TokenExpiresAt, nil
}

// ShouldReplaceToken reports whether a newly delivered token expires later
// than the one already cached. A near-expiry token must not overwrite a
// longer-lived one.
func ShouldReplaceToken(currentExpiry, deliveredExpiry time.Time) bool {
	if deliveredExpiry.IsZero() {
		return false
	}
	return deliveredExpiry.After(currentExpiry)
}
