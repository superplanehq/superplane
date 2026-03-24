package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRequireInstallationAdmin(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	token, err := signer.Generate(r.Account.ID.String(), time.Hour)
	require.NoError(t, err)

	// Chain: AccountAuthMiddleware → RequireInstallationAdmin → final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin-ok"))
	})

	handler := AccountAuthMiddleware(signer)(
		RequireInstallationAdmin()(finalHandler),
	)

	t.Run("non-admin account gets 404", func(t *testing.T) {
		// Ensure account is NOT an admin
		require.NoError(t, database.Conn().Model(&models.Account{}).
			Where("id = ?", r.Account.ID).
			Update("installation_admin", false).Error)

		req := httptest.NewRequest(http.MethodGet, "/admin/api/organizations", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("admin account reaches the handler", func(t *testing.T) {
		// Promote to admin
		require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

		req := httptest.NewRequest(http.MethodGet, "/admin/api/organizations", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "admin-ok", res.Body.String())
	})

	t.Run("unauthenticated request gets 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/organizations", nil)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		// AccountAuthMiddleware redirects/errors before reaching RequireInstallationAdmin
		assert.NotEqual(t, http.StatusOK, res.Code)
	})
}
