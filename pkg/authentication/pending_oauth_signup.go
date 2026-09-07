package authentication

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	log "github.com/sirupsen/logrus"
)

const (
	pendingOAuthSignupCookie   = "pending_oauth_signup"
	pendingOAuthSignupTTL      = 15 * time.Minute
	pendingOAuthSignupType     = "pending_oauth_signup"
	pendingOAuthSignupClaimTyp = "typ"
)

func (a *Handler) setPendingOAuthSignupCookie(w http.ResponseWriter, r *http.Request, user goth.User) {
	if a.jwtSigner == nil {
		return
	}

	token, err := a.jwtSigner.GenerateWithClaims(pendingOAuthSignupTTL, map[string]string{
		pendingOAuthSignupClaimTyp: pendingOAuthSignupType,
		"provider":                 user.Provider,
		"provider_id":              user.UserID,
		"email":                    user.Email,
		"name":                     user.Name,
		"nick":                     user.NickName,
		"avatar":                   user.AvatarURL,
		"access_token":             user.AccessToken,
		"refresh_token":            user.RefreshToken,
		"expires_at":               formatPendingOAuthExpiry(user.ExpiresAt),
	})
	if err != nil {
		log.Errorf("Failed to sign pending OAuth signup cookie for %s: %v", user.Email, err)
		return
	}

	http.SetCookie(w, pendingOAuthSignupCookieValue(r, token, int(pendingOAuthSignupTTL.Seconds())))
}

func (a *Handler) gothUserFromPendingOAuthSignup(r *http.Request, provider string) (goth.User, bool) {
	cookie, err := r.Cookie(pendingOAuthSignupCookie)
	if err != nil || cookie.Value == "" || a.jwtSigner == nil {
		return goth.User{}, false
	}

	claims, err := a.jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return goth.User{}, false
	}

	if claimString(claims[pendingOAuthSignupClaimTyp]) != pendingOAuthSignupType {
		return goth.User{}, false
	}

	pendingProvider := claimString(claims["provider"])
	if pendingProvider == "" || pendingProvider != provider {
		return goth.User{}, false
	}

	email := claimString(claims["email"])
	userID := claimString(claims["provider_id"])
	if email == "" || userID == "" {
		return goth.User{}, false
	}

	return goth.User{
		Provider:     pendingProvider,
		UserID:       userID,
		Email:        email,
		Name:         claimString(claims["name"]),
		NickName:     claimString(claims["nick"]),
		AvatarURL:    claimString(claims["avatar"]),
		AccessToken:  claimString(claims["access_token"]),
		RefreshToken: claimString(claims["refresh_token"]),
		ExpiresAt:    parsePendingOAuthExpiry(claimString(claims["expires_at"])),
	}, true
}

func (a *Handler) completePendingOAuthSignup(w http.ResponseWriter, r *http.Request) bool {
	if !isSignupIntentFromRequest(r) {
		return false
	}

	user, ok := a.gothUserFromPendingOAuthSignup(r, muxProvider(r))
	if !ok {
		return false
	}

	a.finishProviderAuth(w, r, user)
	return true
}

func clearPendingOAuthSignupCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, pendingOAuthSignupCookieValue(r, "", -1))
}

func pendingOAuthSignupCookieValue(r *http.Request, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     pendingOAuthSignupCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
}

func formatPendingOAuthExpiry(expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return ""
	}

	return expiresAt.UTC().Format(time.RFC3339)
}

func parsePendingOAuthExpiry(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func claimString(value any) string {
	text, _ := value.(string)
	return text
}

func muxProvider(r *http.Request) string {
	return mux.Vars(r)["provider"]
}
