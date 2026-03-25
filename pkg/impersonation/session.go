package impersonation

import (
	"fmt"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	CookieName = "impersonation_token"
	TokenType  = "impersonation"
	TTL        = 1 * time.Hour

	claimType    = "type"
	claimAdmin   = "admin_account_id"
	claimAccount = "impersonated_account_id"
	claimSub     = "sub"
)

type Claims struct {
	AdminAccountID        string
	ImpersonatedAccountID string
}

// GenerateToken creates a signed JWT for an impersonation session.
func GenerateToken(signer *jwt.Signer, adminAccountID, targetAccountID string) (string, error) {
	return signer.GenerateWithClaims(TTL, map[string]string{
		claimType:    TokenType,
		claimAdmin:   adminAccountID,
		claimAccount: targetAccountID,
		claimSub:     adminAccountID,
	})
}

// ValidateToken parses and validates an impersonation token, returning its claims.
func ValidateToken(signer *jwt.Signer, tokenString string) (*Claims, error) {
	jwtClaims, err := signer.ValidateAndGetClaims(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid impersonation token: %w", err)
	}

	tokenType, _ := jwtClaims[claimType].(string)
	if tokenType != TokenType {
		return nil, fmt.Errorf("not an impersonation token")
	}

	adminID, _ := jwtClaims[claimAdmin].(string)
	accountID, _ := jwtClaims[claimAccount].(string)

	if adminID == "" || accountID == "" {
		return nil, fmt.Errorf("incomplete impersonation claims")
	}

	return &Claims{
		AdminAccountID:        adminID,
		ImpersonatedAccountID: accountID,
	}, nil
}

// SetCookie writes the impersonation cookie to the response.
func SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(TTL.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie deletes the impersonation cookie.
func ClearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadCookie extracts the impersonation token from the request, if present.
func ReadCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}
