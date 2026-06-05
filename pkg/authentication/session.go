package authentication

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	defaultAccountSessionTTL    = 24 * time.Hour
	defaultAccountSessionMaxAge = 7 * 24 * time.Hour
	sessionStartClaim           = "ses"
)

var (
	accountSessionTTLOnce    sync.Once
	accountSessionTTL        time.Duration
	accountSessionMaxAgeOnce sync.Once
	accountSessionMaxAge     time.Duration
)

// AccountSessionTTL returns how long account_token cookies remain valid
// between activity refreshes. Override with ACCOUNT_SESSION_TTL.
func AccountSessionTTL() time.Duration {
	accountSessionTTLOnce.Do(func() {
		if v := os.Getenv("ACCOUNT_SESSION_TTL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				accountSessionTTL = d
				return
			}
		}
		accountSessionTTL = defaultAccountSessionTTL
	})
	return accountSessionTTL
}

// AccountSessionMaxAge returns the absolute lifetime of a login session,
// measured from the first sign-in, regardless of sliding refresh.
// Override with ACCOUNT_SESSION_MAX_AGE.
func AccountSessionMaxAge() time.Duration {
	accountSessionMaxAgeOnce.Do(func() {
		if v := os.Getenv("ACCOUNT_SESSION_MAX_AGE"); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				accountSessionMaxAge = d
				return
			}
		}
		accountSessionMaxAge = defaultAccountSessionMaxAge
	})
	return accountSessionMaxAge
}

// ResetAccountSessionTTLForTests clears cached session duration settings.
func ResetAccountSessionTTLForTests() {
	accountSessionTTLOnce = sync.Once{}
	accountSessionTTL = 0
	accountSessionMaxAgeOnce = sync.Once{}
	accountSessionMaxAge = 0
}

// SessionStartFromClaims returns when the login session began from the ses
// claim. Tokens without ses cannot be used for absolute max-age enforcement.
func SessionStartFromClaims(claims jwtLib.MapClaims) (time.Time, bool) {
	if ses, ok := claims[sessionStartClaim].(string); ok {
		if unix, err := strconv.ParseInt(ses, 10, 64); err == nil {
			return time.Unix(unix, 0), true
		}
	}

	return time.Time{}, false
}

// IsAccountSessionWithinMaxAge reports whether the session is still within
// its absolute lifetime cap. Tokens without ses are rejected.
func IsAccountSessionWithinMaxAge(claims jwtLib.MapClaims) bool {
	start, ok := SessionStartFromClaims(claims)
	if !ok {
		return false
	}

	return time.Since(start) <= AccountSessionMaxAge()
}

// GenerateAccountToken creates a signed account_token JWT with session tracking.
func GenerateAccountToken(jwtSigner *jwt.Signer, accountID string, sessionStart time.Time, ttl time.Duration) (string, error) {
	return jwtSigner.GenerateWithClaims(ttl, map[string]string{
		"sub":             accountID,
		sessionStartClaim: fmt.Sprintf("%d", sessionStart.Unix()),
	})
}

// IssueAccountSession mints a fresh account_token for a new login session.
func IssueAccountSession(w http.ResponseWriter, r *http.Request, jwtSigner *jwt.Signer, accountID string) error {
	return issueAccountSession(w, r, jwtSigner, accountID, time.Now())
}

func issueAccountSession(w http.ResponseWriter, r *http.Request, jwtSigner *jwt.Signer, accountID string, sessionStart time.Time) error {
	ttl := AccountSessionTTL()
	token, err := GenerateAccountToken(jwtSigner, accountID, sessionStart, ttl)
	if err != nil {
		return err
	}

	SetAccountCookie(w, r, token, ttl)
	return nil
}

// MaybeRefreshAccountSession extends active sessions using a sliding window.
// Any authenticated activity reissues the token for another full TTL, except
// when the token was just minted or the absolute session max age is reached.
func MaybeRefreshAccountSession(w http.ResponseWriter, r *http.Request, jwtSigner *jwt.Signer, account *models.Account) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return
	}

	claims, err := jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return
	}

	if !IsAccountSessionWithinMaxAge(claims) {
		return
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return
	}

	if time.Since(time.Unix(int64(iat), 0)) < time.Minute {
		return
	}

	sessionStart, ok := SessionStartFromClaims(claims)
	if !ok {
		return
	}

	_ = issueAccountSession(w, r, jwtSigner, account.ID.String(), sessionStart)
}
