package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/superplanehq/superplane/pkg/crypto"
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

			//
			// If the authorization header is used,
			// we expect a user API token.
			//
			if r.Header.Get("Authorization") != "" {
				user, err := authenticateUserByToken(r)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), UserContextKey, user)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			//
			// Otherwise, we authenticate the account with the cookie,
			// and expect an organization ID in the header or query parameters.
			//
			user, err := authenticateUserByCookie(jwtSigner, r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func authenticateUserByToken(r *http.Request) (*models.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header not found")
	}

	headerParts := strings.Split(authHeader, "Bearer ")
	if len(headerParts) != 2 {
		return nil, fmt.Errorf("invalid authorization header")
	}

	hashedToken := crypto.HashToken(headerParts[1])
	return models.FindActiveUserByTokenHash(hashedToken)
}

func authenticateUserByCookie(jwtSigner *jwt.Signer, r *http.Request) (*models.User, error) {
	accountID, err := getAccountFromCookie(r, jwtSigner)
	if err != nil {
		return nil, err
	}

	organizationID := findOrganizationID(r)
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID not found")
	}

	account, err := models.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	return models.FindActiveUserByEmail(organizationID, account.Email)
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
