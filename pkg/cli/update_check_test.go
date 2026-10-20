package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func TestConfigFileFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "--config with separate value", args: []string{"apps", "list", "--config", "/tmp/a.yaml"}, expected: "/tmp/a.yaml"},
		{name: "--config= inline value", args: []string{"apps", "--config=/tmp/b.yaml"}, expected: "/tmp/b.yaml"},
		{name: "--config as last arg has no value", args: []string{"apps", "--config"}, expected: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := configFileFromArgs(test.args)

			if test.expected == "" {
				// Falls through to the home directory default, which must at
				// least not return the flag itself.
				if result == "--config" {
					t.Fatalf("configFileFromArgs(%v) returned the flag name", test.args)
				}
				return
			}

			if result != test.expected {
				t.Fatalf("configFileFromArgs(%v) = %q, want %q", test.args, result, test.expected)
			}
		})
	}
}

func TestConfigFileFromArgsDefaultsToHomeConfig(t *testing.T) {
	result := configFileFromArgs([]string{"apps", "list"})

	if filepath.Base(result) != ".superplane.yaml" {
		t.Fatalf("configFileFromArgs default = %q, want a path ending in .superplane.yaml", result)
	}
}

func TestShouldCheckForUpdates(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		contents string
		expected bool
	}{
		{
			name:     "no timestamp recorded -> due",
			contents: "output: text\n",
			expected: true,
		},
		{
			name:     "checked just now -> not due",
			contents: "lastUpdateCheck: " + now.Add(-1*time.Minute).UTC().Format(time.RFC3339) + "\n",
			expected: false,
		},
		{
			name:     "checked 23 hours ago -> not due",
			contents: "lastUpdateCheck: " + now.Add(-23*time.Hour).UTC().Format(time.RFC3339) + "\n",
			expected: false,
		},
		{
			name:     "checked 25 hours ago -> due",
			contents: "lastUpdateCheck: " + now.Add(-25*time.Hour).UTC().Format(time.RFC3339) + "\n",
			expected: true,
		},
		{
			name:     "timestamp in the future -> due",
			contents: "lastUpdateCheck: " + now.Add(48*time.Hour).UTC().Format(time.RFC3339) + "\n",
			expected: true,
		},
		{
			name:     "malformed timestamp -> due",
			contents: "lastUpdateCheck: not-a-timestamp\n",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfigFile(t, test.contents)

			if result := shouldCheckForUpdates(path, now); result != test.expected {
				t.Fatalf("shouldCheckForUpdates() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestShouldCheckForUpdatesWhenConfigIsMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	if !shouldCheckForUpdates(missing, time.Now()) {
		t.Fatal("a missing config file should not block the update check")
	}

	if !shouldCheckForUpdates("", time.Now()) {
		t.Fatal("an unresolvable config path should not block the update check")
	}
}

func TestRecordUpdateCheckRoundTrips(t *testing.T) {
	path := writeConfigFile(t, "output: text\n")
	now := time.Now()

	if err := recordUpdateCheck(path, now); err != nil {
		t.Fatalf("recordUpdateCheck: %v", err)
	}

	// The freshly recorded timestamp must suppress the next check.
	if shouldCheckForUpdates(path, now) {
		t.Fatal("update check should not be due immediately after being recorded")
	}

	// And it must become due again once the interval has elapsed.
	if !shouldCheckForUpdates(path, now.Add(updateCheckInterval+time.Minute)) {
		t.Fatal("update check should be due again after the interval elapses")
	}
}

func TestRecordUpdateCheckPreservesExistingKeys(t *testing.T) {
	path := writeConfigFile(t, "output: json\ncurrentContext: my-org\n")

	if err := recordUpdateCheck(path, time.Now()); err != nil {
		t.Fatalf("recordUpdateCheck: %v", err)
	}

	config := viper.New()
	config.SetConfigFile(path)
	if err := config.ReadInConfig(); err != nil {
		t.Fatalf("read back config: %v", err)
	}

	if got := config.GetString(ConfigKeyOutput); got != "json" {
		t.Fatalf("output = %q, want %q; existing keys must survive the write", got, "json")
	}

	if got := config.GetString(ConfigKeyCurrentContext); got != "my-org" {
		t.Fatalf("currentContext = %q, want %q; existing keys must survive the write", got, "my-org")
	}

	if config.GetTime(ConfigKeyLastUpdateCheck).IsZero() {
		t.Fatal("lastUpdateCheck was not written")
	}
}

func TestRecordUpdateCheckCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".superplane.yaml")

	if err := recordUpdateCheck(path, time.Now()); err != nil {
		t.Fatalf("recordUpdateCheck on a missing file: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
}

func TestRecordUpdateCheckWithoutPath(t *testing.T) {
	if err := recordUpdateCheck("", time.Now()); err == nil {
		t.Fatal("expected an error when no configuration file can be resolved")
	}
}
