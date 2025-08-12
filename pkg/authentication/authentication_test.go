package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func TestHandler_Middleware(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-client-secret")
	handler := NewHandler(signer, r.Encryptor, r.AuthService, "")
	handler.RegisterRoutes(mux.NewRouter())

	//
	// Just a simple handler to verify that the middleware
	// will inject the user into the request's context.
	//
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Error("User not found in context")
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := handler.Middleware(testHandler)

	token, err := signer.GenerateWithClaims(r.User.String(), time.Hour, map[string]any{
		"org": r.Organization.ID.String(),
	})

	require.NoError(t, err)

	t.Run("with valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})

	t.Run("with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "invalid-token",
		})

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})
}
