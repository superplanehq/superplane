package cli

import (
	"context"
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

const (
	releasesDownloadBaseURL      = "https://github.com/superplanehq/superplane/releases/download"
	releaseDownloadHeaderTimeout = 30 * time.Second
	releaseDownloadTimeout       = 15 * time.Minute
)

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

		if err := downloadAndReplaceBinary(cmd.Context(), asset.BrowserDownloadURL, executablePath); err != nil {
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

func downloadAndReplaceBinary(ctx context.Context, downloadURL, executablePath string) error {
	if err := validateSelfUpdateTarget(downloadURL); err != nil {
		return err
	}

	fileMode, err := currentExecutableMode(executablePath)
	if err != nil {
		return err
	}

	tempPath, err := downloadBinaryToTempFile(ctx, downloadURL, filepath.Dir(executablePath), fileMode)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()

	return replaceBinary(tempPath, executablePath)
}

func validateSelfUpdateTarget(downloadURL string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	return fmt.Errorf("self-update is not yet supported on Windows; download the binary directly from %s", downloadURL)
}

func currentExecutableMode(executablePath string) (os.FileMode, error) {
	currentInfo, err := os.Stat(executablePath)
	if err != nil {
		return 0, fmt.Errorf("stat current executable: %w", err)
	}

	return currentInfo.Mode().Perm(), nil
}

func downloadBinaryToTempFile(ctx context.Context, downloadURL, tempDir string, fileMode os.FileMode) (string, error) {
	binaryBody, err := downloadBinary(ctx, downloadURL)
	if err != nil {
		return "", err
	}
	defer binaryBody.Close()

	return writeBinaryToTempFile(binaryBody, tempDir, fileMode)
}

func downloadBinary(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	client := newReleaseDownloadClient()
	ctx, cancel := context.WithTimeout(ctx, releaseDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download updated binary: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, fmt.Errorf("download updated binary: unexpected status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func writeBinaryToTempFile(binaryBody io.Reader, tempDir string, fileMode os.FileMode) (string, error) {
	tempFile, err := os.CreateTemp(tempDir, ".superplane-update-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}

	tempPath := tempFile.Name()
	written, err := io.Copy(tempFile, binaryBody)
	if err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("write downloaded binary: %w", err)
	}

	if written == 0 {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("downloaded binary is empty")
	}

	if err := tempFile.Chmod(fileMode); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("set binary permissions: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("sync downloaded binary: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("close downloaded binary: %w", err)
	}

	return tempPath, nil
}

func replaceBinary(tempPath, executablePath string) error {
	if err := os.Rename(tempPath, executablePath); err != nil {
		return fmt.Errorf("replace existing binary: %w", err)
	}

	return nil
}

func newReleaseDownloadClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{}
	}

	cloned := transport.Clone()
	cloned.ResponseHeaderTimeout = releaseDownloadHeaderTimeout

	return &http.Client{Transport: cloned}
}
