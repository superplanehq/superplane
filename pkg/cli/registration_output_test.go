package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func TestRegistrationOutputPreservesStructuredCredentials(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			setupViper(t)
			expires := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Second).Format(time.RFC3339)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/admin/api/installation/fleets/direct/runners", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"runner":{"id":"runner"},"registrationToken":"token'quoted","registrationExpiresAt":%q,"runnerApiUrl":"https://runner.example.com","displayName":"direct"}`, expires)
			}))
			defer server.Close()
			t.Setenv(EnvURL, server.URL)
			t.Setenv(EnvToken, "management-token")
			root := fleetRoot(t)
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs([]string{"runners", "create", "--fleet", "direct", "--installation", "--output", format})
			require.NoError(t, root.Execute())
			var result map[string]any
			require.NoError(t, yaml.Unmarshal(out.Bytes(), &result))
			require.Equal(t, "token'quoted", result["registrationToken"])
			require.Equal(t, expires, result["registrationExpiresAt"])
			require.Equal(t, "https://runner.example.com", result["runnerApiUrl"])
			require.Equal(t, "direct", result["displayName"])
			require.Equal(t, "runner", result["runner"].(map[string]any)["id"])
			require.NotContains(t, out.String(), "management-token")
		})
	}
}

func TestInstallationPermissionErrorDoesNotFallback(t *testing.T) {
	setupViper(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/admin/api/installation/fleets", r.URL.Path)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	t.Setenv(EnvURL, server.URL)
	t.Setenv(EnvToken, "credential")
	root := fleetRoot(t)
	root.SetErr(&bytes.Buffer{})
	root.SetOut(&bytes.Buffer{})
	root.SetArgs([]string{"fleets", "list", "--installation"})
	require.Error(t, root.Execute())
	require.Equal(t, 1, requests)
}
