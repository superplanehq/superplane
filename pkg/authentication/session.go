package authentication

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const defaultAccountSessionTTL = 24 * time.Hour

var (
	accountSessionTTLOnce sync.Once
	accountSessionTTL     time.Duration
)

// AccountSessionTTL returns how long account_token cookies remain valid.
// Override with ACCOUNT_SESSION_TTL (Go duration syntax, e.g. "24h").
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

// ResetAccountSessionTTLForTests clears the cached session TTL.
func ResetAccountSessionTTLForTests() {
	accountSessionTTLOnce = sync.Once{}
	accountSessionTTL = 0
}

// IssueAccountSession mints a fresh account_token and writes it to the response.
func IssueAccountSession(w http.ResponseWriter, r *http.Request, jwtSigner *jwt.Signer, accountID string) error {
	ttl := AccountSessionTTL()
	token, err := jwtSigner.Generate(accountID, ttl)
	if err != nil {
		return err
	}

	SetAccountCookie(w, r, token, ttl)
	return nil
}

// MaybeRefreshAccountSession extends active sessions using a sliding window.
// Any authenticated activity reissues the token for another full TTL, except
// when the token was just minted (login redirect + first page load).
func MaybeRefreshAccountSession(w http.ResponseWriter, r *http.Request, jwtSigner *jwt.Signer, account *models.Account) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return
	}

	claims, err := jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return
	}

	if time.Since(time.Unix(int64(iat), 0)) < time.Minute {
		return
	}

	_ = IssueAccountSession(w, r, jwtSigner, account.ID.String())
}
