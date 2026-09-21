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
	// last check for a newer release was attempted.
	ConfigKeyLastUpdateCheck = "lastUpdateCheck"

	// updateCheckInterval is the minimum time between two update checks.
	updateCheckInterval = 24 * time.Hour
)

// MaybeStartUpdateCheck starts the background update check when the command
// warrants one and the previous check is older than updateCheckInterval.
//
// The timestamp is recorded before the check runs rather than after it
// succeeds, so a failing or slow endpoint cannot cause every subsequent
// command to retry it.
func MaybeStartUpdateCheck(args []string) {
	if !ShouldStartUpdateCheck(args) {
		return
	}

	configFile := configFileFromArgs(args)
	if !shouldCheckForUpdates(configFile, time.Now()) {
		return
	}

	// A failure to persist the timestamp only means the next command checks
	// again. It is not worth interrupting the user's command over.
	_ = recordUpdateCheck(configFile, time.Now())

	StartUpdateCheck()
}

// configFileFromArgs resolves the configuration file the same way initConfig
// does. It cannot read the parsed --config flag, because the update check runs
// before Cobra executes the command, so the raw arguments are scanned instead.
// An empty string means the path could not be determined.
func configFileFromArgs(args []string) string {
	for i, arg := range args {
		if value, found := strings.CutPrefix(arg, "--config="); found {
			return value
		}

		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
	}

	home, err := homedir.Dir()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/.superplane.yaml", home)
}

// shouldCheckForUpdates reports whether updateCheckInterval has elapsed since
// the last recorded check. It fails open: when the timestamp is missing,
// unreadable or malformed, a check is allowed.
func shouldCheckForUpdates(configFile string, now time.Time) bool {
	if configFile == "" {
		return true
	}

	config := viper.New()
	config.SetConfigFile(configFile)
	if err := config.ReadInConfig(); err != nil {
		return true
	}

	lastCheck := config.GetTime(ConfigKeyLastUpdateCheck)
	if lastCheck.IsZero() {
		return true
	}

	// A timestamp in the future means a clock change or a hand-edited file.
	// Treat it as due rather than blocking checks until it passes.
	if lastCheck.After(now) {
		return true
	}

	return now.Sub(lastCheck) >= updateCheckInterval
}

// recordUpdateCheck stores now as the moment the last update check ran.
//
// It uses an isolated viper instance so the global configuration the commands
// read later is left untouched, and reads the file first so existing keys are
// preserved on write.
func recordUpdateCheck(configFile string, now time.Time) error {
	if configFile == "" {
		return fmt.Errorf("no configuration file to record the update check in")
	}

	config := viper.New()
	config.SetConfigFile(configFile)

	// A missing or empty file is fine; the key is written into a new one.
	_ = config.ReadInConfig()

	config.Set(ConfigKeyLastUpdateCheck, now.UTC().Format(time.RFC3339))
	if err := config.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to record update check: %w", err)
	}

	return nil
}
