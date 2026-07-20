package broker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/runnerbroker/livelogstoken"
)

func TestLiveLogsAuthAcceptsAdminToken(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/v1/tasks/{id}/live-logs", func(r chi.Router) {
		r.Use(liveLogsAuth("admin-token"))
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1/live-logs", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLiveLogsAuthAcceptsStreamJWT(t *testing.T) {
	authToken := "admin-token"
	now := time.Now()
	claims := livelogstoken.Claims{
		TaskID:  "task-1",
		Purpose: livelogstoken.Purpose,
		RegisteredClaims: gojwt.RegisteredClaims{
			Audience:  gojwt.ClaimStrings{livelogstoken.Audience},
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now.Add(-time.Minute)),
		},
	}
	tokenString, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString([]byte(authToken))
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Route("/v1/tasks/{id}/live-logs", func(r chi.Router) {
		r.Use(liveLogsAuth(authToken))
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1/live-logs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLiveLogsCORS(t *testing.T) {
	r := chi.NewRouter()
	r.Use(liveLogsCORS([]string{"http://localhost:8000"}))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:8000")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "http://localhost:8000", rec.Header().Get("Access-Control-Allow-Origin"))
}
