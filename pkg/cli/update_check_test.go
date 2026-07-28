package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// writeConfigFile creates a CLI configuration file with the given contents and
// returns its path.
func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0600))

	return path
}

func TestUpdateCheckDue(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		contents string
		expected bool
	}{
		{
			name:     "no timestamp recorded yet",
			contents: "currentcontext: https://app.superplane.com/org\n",
			expected: true,
		},
		{
			name:     "empty timestamp",
			contents: "lastupdatecheck: \"\"\n",
			expected: true,
		},
		{
			name:     "checked one minute ago",
			contents: "lastupdatecheck: \"2026-07-28T11:59:00Z\"\n",
			expected: false,
		},
		{
			name:     "checked 23 hours ago",
			contents: "lastupdatecheck: \"2026-07-27T13:00:00Z\"\n",
			expected: false,
		},
		{
			name:     "checked exactly 24 hours ago",
			contents: "lastupdatecheck: \"2026-07-27T12:00:00Z\"\n",
			expected: true,
		},
		{
			name:     "checked 25 hours ago",
			contents: "lastupdatecheck: \"2026-07-27T11:00:00Z\"\n",
			expected: true,
		},
		{
			name:     "timestamp in another zone is normalized",
			contents: "lastupdatecheck: \"2026-07-28T13:30:00+02:00\"\n",
			expected: false,
		},
		{
			name:     "malformed timestamp does not suppress checks",
			contents: "lastupdatecheck: \"not-a-timestamp\"\n",
			expected: true,
		},
		{
			name:     "future timestamp does not suppress checks forever",
			contents: "lastupdatecheck: \"2027-01-01T00:00:00Z\"\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.contents)
			require.Equal(t, tt.expected, updateCheckDue(path, now))
		})
	}
}

func TestUpdateCheckDueWithUnusableConfigFile(t *testing.T) {
	now := time.Now()

	t.Run("missing file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")
		require.True(t, updateCheckDue(missing, now))
	})

	t.Run("unresolvable path", func(t *testing.T) {
		require.True(t, updateCheckDue("", now))
	})

	t.Run("corrupted yaml", func(t *testing.T) {
		path := writeConfigFile(t, "lastupdatecheck: [this is: not valid\n")
		require.True(t, updateCheckDue(path, now))
	})
}

func TestUpdateCheckConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "separate --config value",
			args:     []string{"apps", "list", "--config", "/tmp/custom.yaml"},
			expected: "/tmp/custom.yaml",
		},
		{
			name:     "inline --config value",
			args:     []string{"apps", "list", "--config=/tmp/inline.yaml"},
			expected: "/tmp/inline.yaml",
		},
		{
			name:     "trailing --config with no value falls back to the default",
			args:     []string{"apps", "list", "--config"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateCheckConfigFile(tt.args)

			if tt.expected != "" {
				require.Equal(t, tt.expected, got)
				return
			}

			// Falls back to $HOME/.superplane.yaml.
			require.True(t, filepath.IsAbs(got))
			require.Equal(t, ".superplane.yaml", filepath.Base(got))
		})
	}
}

func TestRecordUpdateCheckPersistsTimestamp(t *testing.T) {
	setupViper(t)

	now := time.Date(2026, 7, 28, 9, 30, 0, 0, time.UTC)
	recordUpdateCheck(now)

	// Read it back through a fresh instance, the way the next process would.
	last, ok := lastUpdateCheck(viper.ConfigFileUsed())
	require.True(t, ok)
	require.True(t, now.Equal(last), "expected %s, got %s", now, last)
}

func TestRecordUpdateCheckPreservesExistingConfig(t *testing.T) {
	setupViper(t)

	context := ConfigContext{
		URL:            "https://app.superplane.com",
		Organization:   "Nandor Lite Playground",
		OrganizationID: "eb3aca82-3864-4f76-b1d6-f670c297f136",
		APIToken:       "tok",
	}
	_, err := UpsertContext(context)
	require.NoError(t, err)

	recordUpdateCheck(time.Now())

	// The throttle timestamp must not clobber the contexts already on disk.
	reloaded := viper.New()
	reloaded.SetConfigFile(viper.ConfigFileUsed())
	require.NoError(t, reloaded.ReadInConfig())

	var contexts []ConfigContext
	require.NoError(t, reloaded.UnmarshalKey(ConfigKeyContexts, &contexts))
	require.Len(t, contexts, 1)
	require.Equal(t, "tok", contexts[0].APIToken)
	require.NotEmpty(t, reloaded.GetString(ConfigKeyLastUpdateCheck))
}

func TestStartUpdateCheckRespectsInterval(t *testing.T) {
	originalVersion := Version
	Version = "v0.13.0"
	t.Cleanup(func() { Version = originalVersion })
	t.Cleanup(func() { updateCheckResult = nil })

	// Point the release URL at a server that fails fast, so these cases only
	// assert on whether a check was started, not on its result.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := latestReleaseURL
	latestReleaseURL = server.URL
	t.Cleanup(func() { latestReleaseURL = originalURL })

	started := func(t *testing.T, contents string) bool {
		t.Helper()

		path := writeConfigFile(t, contents)
		updateCheckResult = nil
		StartUpdateCheck([]string{"apps", "list", "--config", path})

		return updateCheckResult != nil
	}

	t.Run("skips when checked recently", func(t *testing.T) {
		recent := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		require.False(t, started(t, fmt.Sprintf("lastupdatecheck: %q\n", recent)))
	})

	t.Run("runs when the last check is stale", func(t *testing.T) {
		stale := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
		require.True(t, started(t, fmt.Sprintf("lastupdatecheck: %q\n", stale)))
	})

	t.Run("runs when no timestamp has been recorded", func(t *testing.T) {
		require.True(t, started(t, "currentcontext: https://app.superplane.com/org\n"))
	})

	t.Run("dev builds never check", func(t *testing.T) {
		Version = "dev"
		defer func() { Version = "v0.13.0" }()

		stale := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
		require.False(t, started(t, fmt.Sprintf("lastupdatecheck: %q\n", stale)))
	})
}

// TestUpdateCheckHitsNetworkOncePerInterval is the regression test for the
// reported behaviour: the CLI used to call the GitHub releases API on every
// command. It drives the real main() sequence against a stub release server
// and counts the requests that actually reach it.
func TestUpdateCheckHitsNetworkOncePerInterval(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v99.0.0","assets":[]}`)
	}))
	defer server.Close()

	originalURL := latestReleaseURL
	latestReleaseURL = server.URL
	t.Cleanup(func() { latestReleaseURL = originalURL })

	originalVersion := Version
	Version = "v0.13.0"
	t.Cleanup(func() { Version = originalVersion })

	t.Cleanup(func() { updateCheckResult = nil })

	setupViper(t)
	configFile := viper.ConfigFileUsed()
	args := []string{"apps", "list", "--config", configFile}

	// runCommand mirrors cmd/cli/main.go: decide, start, execute, then print.
	runCommand := func() {
		updateCheckResult = nil
		if ShouldStartUpdateCheck(args) {
			StartUpdateCheck(args)
		}
		PrintUpdateNotice()
	}

	runCommand()
	require.Equal(t, int32(1), requests.Load(), "first command should check for updates")

	for i := 0; i < 5; i++ {
		runCommand()
	}
	require.Equal(t, int32(1), requests.Load(), "subsequent commands within the interval must not hit the network")

	// Age the recorded timestamp past the interval and confirm it checks again.
	recordUpdateCheck(time.Now().Add(-updateCheckInterval - time.Minute))
	runCommand()
	require.Equal(t, int32(2), requests.Load(), "a stale timestamp should trigger a new check")
}

// TestUpdateCheckRecordedWhenReleaseFetchFails guards the offline case: a
// failing check must still be recorded, otherwise every command retries it.
func TestUpdateCheckRecordedWhenReleaseFetchFails(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := latestReleaseURL
	latestReleaseURL = server.URL
	t.Cleanup(func() { latestReleaseURL = originalURL })

	originalVersion := Version
	Version = "v0.13.0"
	t.Cleanup(func() { Version = originalVersion })

	t.Cleanup(func() { updateCheckResult = nil })

	setupViper(t)
	args := []string{"apps", "list", "--config", viper.ConfigFileUsed()}

	for i := 0; i < 3; i++ {
		updateCheckResult = nil
		if ShouldStartUpdateCheck(args) {
			StartUpdateCheck(args)
		}
		PrintUpdateNotice()
	}

	require.Equal(t, int32(1), requests.Load(), "a failed check must not be retried on the next command")
}
