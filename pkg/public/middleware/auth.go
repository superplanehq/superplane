package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type contextKey string

const AccountContextKey contextKey = "account"
const UserContextKey contextKey = "user"

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

func OrganizationAuthMiddleware(jwtSigner *jwt.Signer) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accountID, err := getAccountFromCookie(r, jwtSigner)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			organizationID := findOrganizationID(r)
			if organizationID == "" {
				http.Error(w, "", http.StatusNotFound)
				return
			}

			account, err := models.FindAccountByID(accountID)
			if err != nil {
				http.Error(w, "", http.StatusNotFound)
				return
			}

			user, err := models.FindUserByEmail(organizationID, account.Email)
			if err != nil {
				http.Error(w, "", http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func findOrganizationID(r *http.Request) string {
	organizationID := r.Header.Get("x-organization-id")
	if organizationID != "" {
		return organizationID
	}

	organizationID = r.URL.Query().Get("organization_id")
	if organizationID != "" {
		return organizationID
	}

	return ""
}

func GetAccountFromContext(ctx context.Context) (*models.Account, bool) {
	account, ok := ctx.Value(AccountContextKey).(*models.Account)
	return account, ok
}

func getAccountFromCookie(r *http.Request, jwtSigner *jwt.Signer) (string, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return "", fmt.Errorf("account token cookie not found")
	}

	claims, err := jwtSigner.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return "", fmt.Errorf("invalid account token: %v", err)
	}

	accountClaim, exists := claims["sub"]
	if !exists {
		return "", fmt.Errorf("account ID missing from token")
	}

	accountID, ok := accountClaim.(string)
	if !ok {
		return "", fmt.Errorf("invalid account ID in token")
	}

	return accountID, nil
}

func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}
