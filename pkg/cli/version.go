package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
// Defaults to "dev" for development builds.
var Version = "dev"

var updateCheckResult <-chan *updateInfo

type updateInfo struct {
	latest string
	err    error
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

	ch := make(chan *updateInfo, 1)
	updateCheckResult = ch
	go func() {
		latest, err := fetchLatestRelease()
		ch <- &updateInfo{latest: latest, err: err}
	}()
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
		case "-h", "--help", "help", "--version", "version", "completion":
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

	if info.err != nil || info.latest == "" {
		return
	}

	if isNewerVersion(Version, info.latest) {
		fmt.Fprintf(os.Stderr, "\nA new version of superplane CLI is available: %s -> %s\n", Version, info.latest)
		fmt.Fprintf(os.Stderr, "Download from: https://github.com/superplanehq/superplane/releases/tag/%s\n", info.latest)
	}
}

const latestReleaseURL = "https://api.github.com/repos/superplanehq/superplane/releases/latest"

func fetchLatestRelease() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", latestReleaseURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
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
