package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	// ConfigKeyLastUpdateCheck stores, in the CLI configuration file, when the
	// last check for a new CLI release happened.
	ConfigKeyLastUpdateCheck = "lastUpdateCheck"

	// updateCheckInterval is how long the CLI waits between two update checks.
	updateCheckInterval = 24 * time.Hour
)

// The update check runs before Cobra initializes the global viper instance, so
// the timestamp is read and written through a dedicated instance. Writing
// through the global one at this point would persist an empty configuration
// over the user's file.

func updateCheckConfigPath(args []string) string {
	if path := configPathFromArgs(args); path != "" {
		return path
	}

	home, err := homedir.Dir()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/.superplane.yaml", home)
}

func configPathFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}

		if value, found := strings.CutPrefix(arg, "--config="); found {
			return value
		}
	}

	return ""
}

func readLastUpdateCheck(path string) time.Time {
	if path == "" {
		return time.Time{}
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return time.Time{}
	}

	return v.GetTime(ConfigKeyLastUpdateCheck)
}

// recordLastUpdateCheck is best effort: a configuration file that cannot be
// written must not break the command the user actually ran.
func recordLastUpdateCheck(path string, now time.Time) {
	if path == "" {
		return
	}

	v := viper.New()
	v.SetConfigFile(path)

	// Existing settings are read back first so they survive the write.
	_ = v.ReadInConfig()
	v.Set(ConfigKeyLastUpdateCheck, now.UTC().Format(time.RFC3339))
	_ = v.WriteConfig()
}

func dueForUpdateCheck(lastCheck, now time.Time) bool {
	if lastCheck.IsZero() {
		return true
	}

	// A timestamp in the future means a clock change or a hand-edited config.
	// Checking now is better than staying silent until the clock catches up.
	if lastCheck.After(now) {
		return true
	}

	return now.Sub(lastCheck) >= updateCheckInterval
}
