package core

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SignIn links the identity, admits the member, and returns the session to issue.
func (p *Pipeline) SignIn(ctx context.Context, conn *models.SSOConnection, external ExternalIdentity, redirect string, existing *ExistingSession) (SignInResult, error) {
	if conn.JITEnabled {
		if err := billing.PrepareAdmission(ctx, conn.OrganizationID); err != nil && !seatError(err) {
			return SignInResult{}, err
		}
	}

	var result SignInResult
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		account, identity, err := p.LinkAccount(tx, conn, external, models.SSOProvisionedByJIT)
		if err != nil {
			return err
		}
		if err := p.EnsureMembership(ctx, tx, conn, account, identity, models.SSOProvisionedByJIT, "", true); err != nil {
			return err
		}

		now := p.now()
		identity.LastLoginAt = &now
		if external.IDPSubject != "" {
			idpSubject := external.IDPSubject
			identity.IDPSubject = &idpSubject
		}
		if external.SessionID != "" {
			sessionID := external.SessionID
			identity.IDPSessionID = &sessionID
		}
		if err := p.storeIDToken(ctx, tx, conn, identity, external.RawIDToken); err != nil {
			return err
		}
		identity.UpdatedAt = now
		if identity.CreatedAt.IsZero() {
			identity.CreatedAt = now
		}
		if err := identity.Save(tx); err != nil {
			return err
		}

		org, err := models.FindOrganizationByIDInTransaction(tx, conn.OrganizationID.String())
		if err != nil {
			return err
		}
		authUnix := now.Unix()
		claim, sessionStart := sessionClaim(existing, account.ID, conn.ID, authUnix)
		result = SignInResult{
			AccountID:    account.ID.String(),
			SessionStart: sessionStart,
			SSOClaim:     claim,
			Redirect:     destination(redirect, org.Slug),
		}
		return nil
	})
	return result, err
}

func (p *Pipeline) storeIDToken(ctx context.Context, tx *gorm.DB, conn *models.SSOConnection, identity *models.SSOIdentity, raw string) error {
	if !conn.IDPLogoutEnabled || raw == "" {
		identity.IDToken = nil
		return nil
	}
	encrypted, err := p.Encryptor.Encrypt(ctx, []byte(raw), []byte(models.SSOIDTokenAssociatedData))
	if err != nil {
		return err
	}
	identity.IDToken = encrypted
	return nil
}

func destination(redirect, slug string) string {
	if authentication.IsValidRedirectURL(redirect) {
		return redirect
	}
	return "/" + slug
}

// DecryptIDToken returns the stored ID token when the connection keeps one.
func (p *Pipeline) DecryptIDToken(ctx context.Context, identity *models.SSOIdentity) (string, error) {
	if identity == nil || len(identity.IDToken) == 0 || p.Encryptor == nil {
		return "", nil
	}
	plain, err := p.Encryptor.Decrypt(ctx, identity.IDToken, []byte(models.SSOIDTokenAssociatedData))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// AuthUnix is the sign-in time written into the session claim.
func AuthUnix(now time.Time) int64 {
	return now.Unix()
}
