package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

const releasesDownloadBaseURL = "https://github.com/superplanehq/superplane/releases/download"

var latestReleaseURL = "https://api.github.com/repos/superplanehq/superplane/releases/latest"

type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"self-update"},
	Short:   "Update the SuperPlane CLI to the latest release",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isDevBuild() {
			return fmt.Errorf("self-update is unavailable for dev builds")
		}

		release, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("check latest release: %w", err)
		}

		if !isNewerVersion(Version, release.TagName) {
			fmt.Fprintf(cmd.OutOrStdout(), "superplane CLI is already up to date (%s)\n", Version)
			return nil
		}

		asset, err := release.currentPlatformAsset()
		if err != nil {
			return err
		}

		executablePath, err := currentExecutablePath()
		if err != nil {
			return fmt.Errorf("resolve current executable: %w", err)
		}

		if err := downloadAndReplaceBinary(asset.BrowserDownloadURL, executablePath); err != nil {
			return fmt.Errorf("upgrade CLI: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated superplane CLI: %s -> %s\n", Version, release.TagName)
		fmt.Fprintf(cmd.OutOrStdout(), "Installed binary: %s\n", executablePath)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(upgradeCmd)
}

func fetchLatestRelease() (*releaseInfo, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("latest release response is missing tag_name")
	}

	return &release, nil
}

func currentCLIAssetName() string {
	return cliAssetName(runtime.GOOS, runtime.GOARCH)
}

func cliAssetName(goos, goarch string) string {
	return fmt.Sprintf("superplane-cli-%s-%s", goos, goarch)
}

func cliDownloadURL(tagName, goos, goarch string) string {
	return fmt.Sprintf("%s/%s/%s", releasesDownloadBaseURL, tagName, cliAssetName(goos, goarch))
}

func (r *releaseInfo) currentPlatformAsset() (*releaseAsset, error) {
	return r.assetForPlatform(runtime.GOOS, runtime.GOARCH)
}

func (r *releaseInfo) assetForPlatform(goos, goarch string) (*releaseAsset, error) {
	expectedName := cliAssetName(goos, goarch)

	for _, asset := range r.Assets {
		if asset.Name != expectedName {
			continue
		}

		if asset.BrowserDownloadURL == "" {
			asset.BrowserDownloadURL = cliDownloadURL(r.TagName, goos, goarch)
		}

		return &asset, nil
	}

	return nil, fmt.Errorf("release %s does not contain a CLI asset for %s/%s (expected %s)", r.TagName, goos, goarch, expectedName)
}

func currentExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolvedPath, nil
	}

	return path, nil
}

func downloadAndReplaceBinary(downloadURL, executablePath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("self-update is not yet supported on Windows; download the binary directly from %s", downloadURL)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download updated binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download updated binary: unexpected status %d", resp.StatusCode)
	}

	currentInfo, err := os.Stat(executablePath)
	if err != nil {
		return fmt.Errorf("stat current executable: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(executablePath), ".superplane-update-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}

	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write downloaded binary: %w", err)
	}

	if err := tempFile.Chmod(currentInfo.Mode().Perm()); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set binary permissions: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync downloaded binary: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close downloaded binary: %w", err)
	}

	if err := os.Rename(tempPath, executablePath); err != nil {
		return fmt.Errorf("replace existing binary: %w", err)
	}

	return nil
}
