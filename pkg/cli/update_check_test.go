package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
	return path
}

func freezeClock(t *testing.T, at time.Time) {
	t.Helper()

	original := timeNow
	timeNow = func() time.Time { return at }
	t.Cleanup(func() { timeNow = original })
}

func TestUpdateCheckDue(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		contents string
		expected bool
	}{
		{
			name:     "no record of a previous check",
			contents: "currentContext: http://localhost:8000/acme\n",
			expected: true,
		},
		{
			name:     "empty config file",
			contents: "",
			expected: true,
		},
		{
			name:     "last check is older than the interval",
			contents: "lastupdatecheck: " + now.Add(-25*time.Hour).Format(time.RFC3339) + "\n",
			expected: true,
		},
		{
			name:     "last check is exactly one interval ago",
			contents: "lastupdatecheck: " + now.Add(-updateCheckInterval).Format(time.RFC3339) + "\n",
			expected: true,
		},
		{
			name:     "last check is inside the interval",
			contents: "lastupdatecheck: " + now.Add(-time.Hour).Format(time.RFC3339) + "\n",
			expected: false,
		},
		{
			name:     "timestamp without quotes, which YAML reads as a time",
			contents: "lastupdatecheck: " + now.Add(-time.Hour).Format(time.RFC3339) + "\n",
			expected: false,
		},
		{
			name:     "timestamp in quotes",
			contents: "lastupdatecheck: \"" + now.Add(-time.Hour).Format(time.RFC3339) + "\"\n",
			expected: false,
		},
		{
			name:     "timestamp is not readable",
			contents: "lastupdatecheck: not-a-timestamp\n",
			expected: true,
		},
		{
			name:     "config file is not valid YAML",
			contents: "\tthis is not yaml\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freezeClock(t, now)

			require.Equal(t, tt.expected, updateCheckDue(writeConfigFile(t, tt.contents)))
		})
	}
}

func TestUpdateCheckDueWhenConfigFileIsMissing(t *testing.T) {
	freezeClock(t, time.Now())

	require.True(t, updateCheckDue(filepath.Join(t.TempDir(), "absent.yaml")))
	require.True(t, updateCheckDue(""))
}

// A check that just ran must not run again until the interval passes.
func TestRecordUpdateCheckStopsTheNextCheck(t *testing.T) {
	now := time.Now()
	freezeClock(t, now)

	path := writeConfigFile(t, "")
	require.NoError(t, recordUpdateCheck(path))
	require.False(t, updateCheckDue(path))

	freezeClock(t, now.Add(updateCheckInterval))
	require.True(t, updateCheckDue(path))
}

func TestRecordUpdateCheckKeepsOtherSettings(t *testing.T) {
	freezeClock(t, time.Now())

	path := writeConfigFile(t, "currentContext: http://localhost:8000/acme\noutput: json\n")
	require.NoError(t, recordUpdateCheck(path))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &config))
	require.Equal(t, "http://localhost:8000/acme", config["currentContext"])
	require.Equal(t, "json", config["output"])
	require.Contains(t, config, ConfigKeyLastUpdateCheck)
}

// A user can write the key by hand in camel case. The CLI must not rename it.
func TestRecordUpdateCheckKeepsTheKeyNameInTheFile(t *testing.T) {
	freezeClock(t, time.Now())

	path := writeConfigFile(t, "lastUpdateCheck: \"2020-01-01T00:00:00Z\"\n")
	require.NoError(t, recordUpdateCheck(path))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &config))
	require.Contains(t, config, "lastUpdateCheck")
	require.NotContains(t, config, ConfigKeyLastUpdateCheck)
}

func TestRecordUpdateCheckCreatesMissingConfigFile(t *testing.T) {
	freezeClock(t, time.Now())

	path := filepath.Join(t.TempDir(), ".superplane.yaml")
	require.NoError(t, recordUpdateCheck(path))
	require.False(t, updateCheckDue(path))
}

func TestRecordUpdateCheckWithoutAPath(t *testing.T) {
	freezeClock(t, time.Now())

	require.Error(t, recordUpdateCheck(""))
}

// Viper writes configuration keys in lower case. The timestamp must survive a
// command that saves the configuration after an update check.
func TestUpdateCheckDueIsNotCaseSensitive(t *testing.T) {
	now := time.Now()
	freezeClock(t, now)

	path := writeConfigFile(t, "lastUpdateCheck: "+now.Add(-time.Hour).Format(time.RFC3339)+"\n")
	require.False(t, updateCheckDue(path))
}

func TestConfigFilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	defaultPath := filepath.Join(home, ".superplane.yaml")

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "no arguments", args: nil, expected: defaultPath},
		{name: "unrelated flags", args: []string{"apps", "list", "-o", "json"}, expected: defaultPath},
		{name: "config flag", args: []string{"--config", "/tmp/custom.yaml", "whoami"}, expected: "/tmp/custom.yaml"},
		{name: "config flag with equals", args: []string{"--config=/tmp/custom.yaml"}, expected: "/tmp/custom.yaml"},
		{name: "config flag without a value", args: []string{"whoami", "--config"}, expected: defaultPath},
		{name: "empty config flag", args: []string{"--config="}, expected: defaultPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, configFilePath(tt.args))
		})
	}
}

// The CLI writes this file without the user asking for it, once a day. It must
// not remove the comments or change the order of the keys.
func TestRecordUpdateCheckKeepsTheShapeOfTheFile(t *testing.T) {
	freezeClock(t, time.Now())

	original := `# SuperPlane CLI configuration
currentContext: http://localhost:8000/acme
contexts:
    - url: http://localhost:8000
      organization: acme
      apiToken: secret-token # do not share
output: json
`
	path := writeConfigFile(t, original)
	require.NoError(t, recordUpdateCheck(path))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	updated := string(contents)
	require.Contains(t, updated, "# SuperPlane CLI configuration")
	require.Contains(t, updated, "apiToken: secret-token # do not share")
	require.Contains(t, updated, "output: json")

	// The new key is appended, so every original line keeps its position.
	require.Equal(t, strings.Split(original, "\n")[:6], strings.Split(updated, "\n")[:6])
	require.Contains(t, updated, ConfigKeyLastUpdateCheck)
}

// A second run must replace the timestamp, not add a second one.
func TestRecordUpdateCheckReplacesTheTimestamp(t *testing.T) {
	now := time.Now()
	freezeClock(t, now)

	path := writeConfigFile(t, "output: json\n")
	require.NoError(t, recordUpdateCheck(path))

	freezeClock(t, now.Add(48*time.Hour))
	require.NoError(t, recordUpdateCheck(path))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(contents), ConfigKeyLastUpdateCheck))
	require.False(t, updateCheckDue(path))
}

func TestReadConfigDocumentRejectsAFileThatIsNotAMapping(t *testing.T) {
	_, err := readConfigDocument(writeConfigFile(t, "- one\n- two\n"))
	require.Error(t, err)
}
