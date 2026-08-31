package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is set at build time via -ldflags.
// Defaults to "dev" for development builds.
var Version = "dev"

// updateCheckInterval controls how often StartUpdateCheck is allowed to hit
// the GitHub releases API. We only need to know about new releases once a
// day, so there's no reason to pay that network cost on every command.
const updateCheckInterval = 24 * time.Hour

var updateCheckResult <-chan *updateInfo

type updateInfo struct {
	release *releaseInfo
	err     error
}

func isDevBuild() bool {
	return Version == "" || Version == "dev"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)

	// Enable Cobra's built-in --version flag handling.
	// We pre-register the "version" flag without a shorthand to prevent
	// Cobra from adding -v, which conflicts with --verbose.
	RootCmd.Version = Version
	RootCmd.SetVersionTemplate("{{.Version}}\n")
	RootCmd.Flags().Bool("version", false, "Print the CLI version")
}

// StartUpdateCheck begins an async check for a newer CLI version.
// It is a no-op for dev builds.
func StartUpdateCheck() {
	if isDevBuild() {
		return
	}

	recordUpdateCheck()

	ch := make(chan *updateInfo, 1)
	updateCheckResult = ch
	go func() {
		release, err := fetchLatestRelease()
		ch <- &updateInfo{release: release, err: err}
	}()
}

// ShouldCheckForUpdateNow reports whether enough time has passed since the
// last update check for it to be worth hitting the GitHub releases API
// again. The timestamp of the last check is persisted in the CLI
// configuration file, so this holds across separate command invocations.
func ShouldCheckForUpdateNow() bool {
	last := viper.GetString(ConfigKeyLastUpdateCheck)
	if last == "" {
		return true
	}

	lastChecked, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return true
	}

	return time.Since(lastChecked) >= updateCheckInterval
}

// recordUpdateCheck persists the current time as the last update check
// timestamp. Failures to persist it are non-fatal: at worst, we check for
// updates again earlier than strictly necessary next time.
func recordUpdateCheck() {
	viper.Set(ConfigKeyLastUpdateCheck, time.Now().UTC().Format(time.RFC3339))
	_ = WriteConfig()
}

func ShouldStartUpdateCheck(args []string) bool {
	if isDevBuild() {
		return false
	}

	if len(args) == 0 {
		return false
	}

	skipValue := false
	for _, arg := range args {
		if skipValue {
			skipValue = false
			continue
		}

		switch arg {
		case "", "--":
			continue
		case "--config", "--output", "-o":
			skipValue = true
			continue
		case "-h", "--help", "help", "--version", "version", "completion", "upgrade", "self-update":
			return false
		}

		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "-o=") {
			continue
		}

		if strings.HasPrefix(arg, "-") {
			continue
		}

		return true
	}

	return false
}

// PrintUpdateNotice prints an upgrade notice to stderr if a newer version
// is available. It waits at most 1 second for the background check to finish.
func PrintUpdateNotice() {
	if updateCheckResult == nil {
		return
	}

	var info *updateInfo
	select {
	case info = <-updateCheckResult:
	case <-time.After(1 * time.Second):
		return
	}

	if info.err != nil || info.release == nil {
		return
	}

	notice := buildUpdateNotice(Version, info.release, runtime.GOOS, runtime.GOARCH)
	if notice != "" {
		fmt.Fprint(os.Stderr, notice)
	}
}

func buildUpdateNotice(currentVersion string, release *releaseInfo, goos, goarch string) string {
	if release == nil || !isNewerVersion(currentVersion, release.TagName) {
		return ""
	}

	downloadURL := cliDownloadURL(release.TagName, goos, goarch)
	asset, err := release.assetForPlatform(goos, goarch)
	if err == nil && asset.BrowserDownloadURL != "" {
		downloadURL = asset.BrowserDownloadURL
	}

	return fmt.Sprintf(
		"\nA new version of superplane CLI is available: %s -> %s\nRun: superplane upgrade\nDirect download: %s\n",
		currentVersion,
		release.TagName,
		downloadURL,
	)
}

// isNewerVersion returns true if latest is a newer semver than current.
// Both may optionally have a "v" prefix.
func isNewerVersion(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	if current == latest {
		return false
	}

	var cMajor, cMinor, cPatch int
	var lMajor, lMinor, lPatch int

	fmt.Sscanf(current, "%d.%d.%d", &cMajor, &cMinor, &cPatch)
	fmt.Sscanf(latest, "%d.%d.%d", &lMajor, &lMinor, &lPatch)

	if lMajor != cMajor {
		return lMajor > cMajor
	}

	if lMinor != cMinor {
		return lMinor > cMinor
	}

	return lPatch > cPatch
}
