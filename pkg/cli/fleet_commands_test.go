package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/commands/fleets"
	"github.com/superplanehq/superplane/pkg/cli/commands/runners"
)

func fleetRoot(t *testing.T) *cobra.Command {
	t.Helper()
	previousConfigFile := cfgFile
	t.Cleanup(func() { cfgFile = previousConfigFile })
	cfgFile = viper.ConfigFileUsed()
	root := &cobra.Command{Use: "superplane", PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return ValidateEnvironmentContext() }}
	root.PersistentFlags().String("output", "json", "")
	root.AddCommand(fleets.NewCommand(defaultBindOptions()), runners.NewCommand(defaultBindOptions()))
	return root
}

func TestFleetCommandsRouteExplicitScopes(t *testing.T) {
	cases := []struct {
		args                                                 []string
		method, organizationPath, installationPath, response string
	}{
		{[]string{"fleets", "list"}, "GET", "/fleets", "/fleets", `{"fleets":[{"id":"direct","enabled":false}]}`},
		{[]string{"fleets", "create", "--file", "-"}, "POST", "/fleets", "/fleets", `{"fleet":{"id":"direct"}}`},
		{[]string{"fleets", "describe", "direct"}, "GET", "/fleets/direct", "/fleets/direct", `{"fleet":{"id":"direct"}}`},
		{[]string{"fleets", "update", "direct", "--file", "-"}, "PATCH", "/fleets/direct", "/fleets/direct", `{"fleet":{"id":"direct"}}`},
		{[]string{"fleets", "list-tasks", "--fleet", "direct", "--state", "queued", "--limit", "2", "--after-id", "cursor"}, "GET", "/fleets/direct/tasks", "/fleets/direct/tasks", `{"tasks":[{"id":"task","queuedAt":"2026-11-01T00:00:00Z"}],"totalCount":"3","hasNextPage":true}`},
		{[]string{"runners", "list", "--fleet", "direct", "--state", "idle", "--limit", "2", "--after-id", "cursor"}, "GET", "/runners", "/fleets/direct/runners", `{"runners":[{"id":"runner"}],"totalCount":"3","hasNextPage":true}`},
		{[]string{"runners", "create", "--fleet", "direct", "--task", "task", "--idempotency-key", "request"}, "POST", "/runners", "/fleets/direct/runners", `{"runner":{"id":"runner"},"registrationToken":"short-lived","registrationExpiresAt":"2026-11-01T00:00:00Z","runnerApiUrl":"https://runner.example.com"}`},
		{[]string{"runners", "describe", "runner", "--fleet", "direct"}, "GET", "/runners/runner", "/fleets/direct/runners/runner", `{"runner":{"id":"runner"}}`},
		{[]string{"runners", "delete", "runner", "--fleet", "direct", "--state", "idle"}, "DELETE", "/runners/runner", "/fleets/direct/runners/runner", `{"runner":{"id":"runner","state":"terminated"}}`},
	}
	for _, installation := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%t/%s", installation, strings.Join(tc.args, " ")), func(t *testing.T) {
				setupViper(t)
				requests := 0
				path := "/api/v1/organizations/selected" + tc.organizationPath
				if installation {
					path = "/admin/api/installation" + tc.installationPath
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					require.Equal(t, "Bearer credential", r.Header.Get("Authorization"))
					w.Header().Set("Content-Type", "application/json")
					if strings.HasSuffix(r.URL.Path, "/capacity") {
						require.Equal(t, path+"/capacity", r.URL.Path)
						_, _ = fmt.Fprint(w, `{"idleRunners":"2"}`)
						return
					}
					require.Equal(t, path, r.URL.Path)
					require.Equal(t, tc.method, r.Method)
					if !installation && strings.HasPrefix(tc.organizationPath, "/runners") && tc.method != "POST" {
						require.Equal(t, "direct", r.URL.Query().Get("fleetId"))
					}
					if strings.Contains(strings.Join(tc.args, " "), "--after-id") {
						require.Equal(t, "cursor", r.URL.Query().Get("afterId"))
						require.Equal(t, "2", r.URL.Query().Get("limit"))
						require.NotEmpty(t, r.URL.Query()["states"])
					}
					if tc.args[0] == "runners" && tc.args[1] == "create" {
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, true, body["ephemeral"])
						require.Equal(t, "task", body["taskId"])
						require.Equal(t, "request", body["idempotencyKey"])
						if !installation {
							require.Equal(t, "direct", body["fleetId"])
						}
					}
					if tc.args[0] == "fleets" && tc.args[1] == "update" {
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, false, body["enabled"])
						require.NotContains(t, body, "runnerVersion")
					}
					_, _ = fmt.Fprint(w, tc.response)
				}))
				defer server.Close()
				if installation {
					t.Setenv(EnvURL, server.URL)
					t.Setenv(EnvToken, "credential")
				} else {
					_, err := UpsertContext(ConfigContext{URL: server.URL, APIToken: "credential", OrganizationID: "selected"})
					require.NoError(t, err)
					// Environment credentials on the same server preserve the selected organization.
					if tc.args[0] == "fleets" && tc.args[1] == "list" {
						t.Setenv(EnvURL, server.URL+"/")
						t.Setenv(EnvToken, "credential")
					}
				}
				before, err := os.ReadFile(viper.ConfigFileUsed())
				require.NoError(t, err)
				root := fleetRoot(t)
				var out bytes.Buffer
				root.SetOut(&out)
				root.SetIn(strings.NewReader("fleetId: direct\nrunnerVersion: dev\nspec:\n  architecture: amd64\n"))
				if tc.args[1] == "update" {
					root.SetIn(strings.NewReader("enabled: false\n"))
				}
				args := append([]string{}, tc.args...)
				if installation {
					args = append(args, "--installation")
				}
				root.SetArgs(args)
				require.NoError(t, root.Execute())
				var result map[string]any
				require.NoError(t, json.Unmarshal(out.Bytes(), &result))
				if strings.Contains(tc.response, "totalCount") {
					require.Equal(t, "3", result["totalCount"])
					require.Equal(t, true, result["hasNextPage"])
				}
				if tc.args[1] == "create" && tc.args[0] == "runners" {
					require.Equal(t, "short-lived", result["registrationToken"])
				}
				after, err := os.ReadFile(viper.ConfigFileUsed())
				require.NoError(t, err)
				require.Equal(t, before, after)
				expected := 1
				if tc.args[0] == "fleets" && tc.args[1] == "describe" {
					expected = 2
				}
				require.Equal(t, expected, requests)
			})
		}
	}
}

func TestRunnerReusableScopeAndPermissionErrors(t *testing.T) {
	for _, installation := range []bool{false, true} {
		t.Run(fmt.Sprint(installation), func(t *testing.T) {
			setupViper(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/v1/me" {
					_, _ = fmt.Fprint(w, `{"user":{"organizationId":"identity"}}`)
					return
				}
				require.Equal(t, "/api/v1/organizations/identity/runners", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, false, body["ephemeral"])
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			t.Setenv(EnvURL, server.URL)
			t.Setenv(EnvToken, "credential")
			root := fleetRoot(t)
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			args := []string{"runners", "create", "--fleet", "direct", "--ephemeral=false"}
			if installation {
				args = append(args, "--installation")
			}
			root.SetArgs(args)
			err := root.Execute()
			require.Error(t, err)
			if installation {
				require.ErrorContains(t, err, "must be ephemeral")
				require.Zero(t, requests)
				return
			}
			require.Equal(t, 2, requests)
		})
	}
}

func TestFleetCommandsResolveOrganizationOnEnvironmentServer(t *testing.T) {
	setupViper(t)
	_, err := UpsertContext(ConfigContext{
		URL: "https://production.example.com", APIToken: "production-token", OrganizationID: "production-organization",
	})
	require.NoError(t, err)
	before, err := os.ReadFile(viper.ConfigFileUsed())
	require.NoError(t, err)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		require.Equal(t, "Bearer local-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/me":
			_, _ = fmt.Fprint(w, `{"user":{"organizationId":"local-organization"}}`)
		case "/api/v1/organizations/local-organization/fleets":
			_, _ = fmt.Fprint(w, `{"fleets":[{"id":"local-fleet"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv(EnvURL, server.URL+"/")
	t.Setenv(EnvToken, "local-token")
	root := fleetRoot(t)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"fleets", "list"})
	require.NoError(t, root.Execute())
	require.JSONEq(t, `{"fleets":[{"id":"local-fleet"}]}`, out.String())
	require.Equal(t, []string{"/api/v1/me", "/api/v1/organizations/local-organization/fleets"}, paths)
	after, err := os.ReadFile(viper.ConfigFileUsed())
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestFleetUpdateRejectsConflictingFileID(t *testing.T) {
	setupViper(t)
	root := fleetRoot(t)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader("fleetId: other\nenabled: false\n"))
	root.SetArgs([]string{"fleets", "update", "direct", "--file", "-", "--installation"})
	require.ErrorContains(t, root.Execute(), "conflicts")
}
