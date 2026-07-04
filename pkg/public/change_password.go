package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/impersonation"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

// hasActiveImpersonation reports whether the request is being served
// inside a fully validated impersonation session. We deliberately rely
// on middleware.GetImpersonationFromContext rather than re-parsing the
// cookie locally: the middleware has already verified that the cookie
// is a real impersonation token issued to *this* admin, that the admin
// is still an installation admin, and that the session is fresh
// relative to password rotation. A stale or unrelated cookie that
// happens to be structurally valid will not produce an Active info
// here, so we won't incorrectly block password changes for it.
func hasActiveImpersonation(r *http.Request) bool {
	info, ok := middleware.GetImpersonationFromContext(r.Context())
	return ok && info != nil && info.Active
}

const minPasswordLength = 8

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// changePassword lets the currently authenticated account rotate its
// password. On success, the password hash is updated, the account's
// password_changed_at is bumped (which invalidates every other cookie /
// scoped JWT for the account via middleware checks), every API token for
// the account is wiped, and the current device is reissued a fresh
// account_token cookie so the user stays signed in here.
//
// The endpoint is intentionally account-scoped (not org-scoped): password
// material lives on Account, and an account in many organizations should
// not need org context to manage its credentials.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	if !s.authHandler.PasswordLoginEnabled() {
		http.Error(w, "Password login is not enabled", http.StatusForbidden)
		return
	}

	// Always operate on the real account, never the impersonated one.
	// An installation admin must not be able to silently rewrite the
	// password of a user they are impersonating.
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if hasActiveImpersonation(r) {
		http.Error(w, "Password change is not allowed while impersonating", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var req changePasswordRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "Current password and new password are required", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < minPasswordLength {
		http.Error(w, "New password must be at least 8 characters long", http.StatusBadRequest)
		return
	}

	if req.NewPassword == req.CurrentPassword {
		http.Error(w, "New password must be different from the current password", http.StatusBadRequest)
		return
	}

	passwordAuth, err := models.FindAccountPasswordAuthByAccountID(account.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Password change is not available for this account", http.StatusForbidden)
			return
		}

		log.Errorf("Failed to load password auth for account %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !crypto.VerifyPassword(passwordAuth.PasswordHash, req.CurrentPassword) {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	newHash, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		log.Errorf("Failed to hash new password for account %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := passwordAuth.UpdatePasswordHashInTransaction(tx, newHash); err != nil {
			return err
		}

		if err := account.MarkPasswordChangedInTransaction(tx, now); err != nil {
			return err
		}

		return models.ClearTokenHashesForAccountInTransaction(tx, account.ID)
	})

	if err != nil {
		log.Errorf("Failed to rotate password for account %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Drop any impersonation cookie defensively (we already refused above
	// when one was present, but a stale cookie might still be on the wire).
	impersonation.ClearCookie(w, r)

	// Reissue the current device's cookie so the user stays signed in
	// here. Its iat is after PasswordChangedAt, so the freshness check
	// passes for this device only — every other device's cookie is now
	// stale and will be rejected.
	if err := authentication.IssueAccountSession(w, r, s.jwt, account.ID.String()); err != nil {
		log.Errorf("Failed to issue refreshed account_token after password change for %s: %v", account.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// accountHasPassword reports whether the account has an
// account_password_auth row, i.e. whether email/password is one of its
// sign-in methods. Used by the SPA to decide whether to render the
// change-password form on the Profile page.
func accountHasPassword(accountID uuid.UUID) (bool, error) {
	_, err := models.FindAccountPasswordAuthByAccountID(accountID)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return false, err
}
