package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestDueForUpdateCheck(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		lastCheck time.Time
		expected  bool
	}{
		{"never checked", time.Time{}, true},
		{"checked an hour ago", now.Add(-1 * time.Hour), false},
		{"checked just before the interval", now.Add(-23 * time.Hour), false},
		{"checked exactly one interval ago", now.Add(-updateCheckInterval), true},
		{"checked two days ago", now.Add(-48 * time.Hour), true},
		{"timestamp in the future", now.Add(1 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, dueForUpdateCheck(tt.lastCheck, now))
		})
	}
}

func TestConfigPathFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"no config flag", []string{"apps", "list"}, ""},
		{"separate value", []string{"--config", "/tmp/other.yaml", "apps"}, "/tmp/other.yaml"},
		{"inline value", []string{"--config=/tmp/other.yaml"}, "/tmp/other.yaml"},
		{"flag without value", []string{"apps", "--config"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, configPathFromArgs(tt.args))
		})
	}
}

func TestRecordLastUpdateCheckKeepsExistingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	require.NoError(t, os.WriteFile(path, []byte("output: json\n"), 0600))

	now := time.Now().UTC().Truncate(time.Second)
	recordLastUpdateCheck(path, now)

	require.Equal(t, now, readLastUpdateCheck(path).UTC())

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())
	require.Equal(t, "json", v.GetString(ConfigKeyOutput))
}

func TestReadLastUpdateCheckWithoutConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".superplane.yaml")

	require.True(t, readLastUpdateCheck(path).IsZero())
	require.True(t, readLastUpdateCheck("").IsZero())
}

func TestShouldStartUpdateCheckHonorsInterval(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()
	Version = "v0.13.0"

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	args := []string{"--config", path, "whoami"}

	require.True(t, ShouldStartUpdateCheck(args), "first run should check for updates")

	StartUpdateCheck(args)
	require.False(t, ShouldStartUpdateCheck(args), "a second run on the same day should not check again")

	recordLastUpdateCheck(path, time.Now().Add(-25*time.Hour))
	require.True(t, ShouldStartUpdateCheck(args), "a run on the next day should check again")
}
