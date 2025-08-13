package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type contextKey string

const AccountContextKey contextKey = "account"

func AccountAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accountID, err := getAccountFromCookie(r, jwtSigner)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			account, err := models.FindAccountByID(accountID)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			ctx := context.WithValue(r.Context(), AccountContextKey, account)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func GetAccountFromContext(ctx context.Context) (*models.Account, bool) {
	account, ok := ctx.Value(AccountContextKey).(*models.Account)
	return account, ok
}
