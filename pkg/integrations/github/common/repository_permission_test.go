package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__Client__GetRepositoryPermissionLevel(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		permission string
		wantError  bool
	}{
		{
			name:       "returns the existing user's permission",
			status:     http.StatusOK,
			body:       `{"permission":"write","role_name":"write"}`,
			permission: "write",
		},
		{
			name:       "a nonexistent user has no repository access",
			status:     http.StatusNotFound,
			body:       `{"message":"former-author is not a user"}`,
			permission: "none",
		},
		{
			name:      "a missing repository remains an error",
			status:    http.StatusNotFound,
			body:      `{"message":"Not Found"}`,
			wantError: true,
		},
		{
			name:      "an authorization failure remains an error",
			status:    http.StatusForbidden,
			body:      `{"message":"Resource not accessible by integration"}`,
			wantError: true,
		},
		{
			name:      "a server failure remains an error",
			status:    http.StatusInternalServerError,
			body:      `{"message":"Internal Server Error"}`,
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/acme/payments/collaborators/former-author/permission", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			permission, _, err := clientForTestServer(t, server).GetRepositoryPermissionLevel(
				context.Background(), "acme/payments", "former-author",
			)
			if tc.wantError {
				require.Error(t, err)
				assert.Equal(t, tc.status, StatusCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, permission)
			assert.Equal(t, tc.permission, permission.GetPermission())
			assert.Equal(t, tc.permission, permission.GetRoleName())
		})
	}
}
