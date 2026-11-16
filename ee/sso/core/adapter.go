package core

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ExternalIdentity is the person the protocol adapter authenticated.
type ExternalIdentity struct {
	Subject       string
	IDPSubject    string
	Email         string
	EmailVerified bool
	Name          string
	SessionID     string
	AuthTime      time.Time
	RawIDToken    string
	Claims        map[string]any
}

// LoginRequest is the start of an SP-initiated or IdP-initiated sign-in.
type LoginRequest struct {
	LoginHint string
	Redirect  string
	Test      bool
}

// LoginState is the short-lived binding between the start request and the callback.
type LoginState struct {
	ConnectionID uuid.UUID
	State        string
	Nonce        string
	Verifier     string
	Redirect     string
	Test         bool
}

// Adapter is one SSO protocol. The pipeline after HandleCallback does not
// depend on the protocol.
type Adapter interface {
	Protocol() string
	ValidateConfig(ctx context.Context, conn *models.SSOConnection) error
	BeginLogin(ctx context.Context, conn *models.SSOConnection, req LoginRequest) (redirectURL string, state LoginState, err error)
	HandleCallback(ctx context.Context, conn *models.SSOConnection, r *http.Request, state LoginState) (*ExternalIdentity, error)
	BeginLogout(ctx context.Context, conn *models.SSOConnection, idTokenHint string) (redirectURL string, ok bool, err error)
	VerifyLogoutToken(ctx context.Context, conn *models.SSOConnection, raw string) (subject, sessionID string, err error)
}

// SignInResult is the session the HTTP layer must issue.
type SignInResult struct {
	AccountID    string
	SessionStart time.Time
	SSOClaim     string
	Redirect     string
}

// Pipeline links an external identity to an account and a membership.
type Pipeline struct {
	Encryptor          crypto.Encryptor
	Entitlements       licensing.Entitlements
	Auth               authorization.Authorization
	Now                func() time.Time
	BeforeIdentityLock func(tx *gorm.DB, id uuid.UUID)
}

func (p *Pipeline) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// ExistingSession is a valid account cookie, when the browser already has one.
type ExistingSession struct {
	AccountID    uuid.UUID
	SessionStart time.Time
	SSO          string
}

func sessionClaim(existing *ExistingSession, accountID, connectionID uuid.UUID, authUnix int64) (string, time.Time) {
	if existing != nil && existing.AccountID == accountID {
		return authentication.MergeSSOSession(existing.SSO, connectionID, authUnix), existing.SessionStart
	}
	return authentication.FormatSSOSessions([]authentication.SSOSession{{
		ConnectionID: connectionID,
		AuthUnix:     authUnix,
	}}), time.Unix(authUnix, 0)
}
