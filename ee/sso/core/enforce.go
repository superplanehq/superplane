package core

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// AccessDecision is the enforcement result for one cookie request.
type AccessDecision struct {
	Required   bool
	StartURL   string
	Connection *models.SSOConnection
}

// CheckOrganizationAccess applies the session rules for an active connection.
// A nil decision means the request may continue.
func (p *Pipeline) CheckOrganizationAccess(ctx context.Context, user *models.User, session authentication.AccountSession) (*AccessDecision, error) {
	if user == nil || user.OrganizationID == uuid.Nil {
		return nil, nil
	}
	conn, err := models.FindActiveSSOConnection(database.DB(ctx), user.OrganizationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	now := p.now()
	entry, ok := entryFor(session.SSO, conn.ID.String())
	if ok {
		var identity *models.SSOIdentity
		if user.AccountID != nil {
			identity, err = models.FindSSOIdentityByAccount(database.DB(ctx), conn.ID, *user.AccountID)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				identity = nil
			}
		}
		if identity == nil || !identity.Active || revoked(identity, entry.AuthUnix) {
			return required(conn), nil
		}
		age := time.Duration(conn.MaxSessionAgeSeconds) * time.Second
		if now.Sub(time.Unix(entry.AuthUnix, 0)) <= age {
			return nil, nil
		}
	}

	if conn.Enforcement == models.SSOEnforcementOptional {
		return nil, nil
	}
	if user.IsOwner && conn.AllowOwnerBypass {
		logging.AddRequestLogFields(ctx, map[string]any{"sso_owner_bypass": true})
		return nil, nil
	}
	return required(conn), nil
}

func required(conn *models.SSOConnection) *AccessDecision {
	return &AccessDecision{
		Required:   true,
		StartURL:   "/auth/sso/" + conn.ID.String() + "/start",
		Connection: conn,
	}
}

func entryFor(raw, connectionID string) (authentication.SSOSession, bool) {
	for _, session := range authentication.ParseSSOSessions(raw) {
		if session.ConnectionID.String() == connectionID {
			return session, true
		}
	}
	return authentication.SSOSession{}, false
}

func revoked(identity *models.SSOIdentity, authUnix int64) bool {
	if identity.SessionsRevokedAt == nil {
		return false
	}
	return !identity.SessionsRevokedAt.Before(time.Unix(authUnix, 0))
}
