package plugins

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/plugins"
)

type installCommand struct{}

func newInstallCommand(options core.BindOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <file.spx>",
		Short: "Install a plugin from a .spx archive",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(cmd, &installCommand{}, options)
	return cmd
}

func (c *installCommand) Execute(ctx core.CommandContext) error {
	spxPath := ctx.Args[0]

	pluginsDir := os.Getenv("SUPERPLANE_PLUGINS_DIR")
	if pluginsDir == "" {
		pluginsDir = "plugins"
	}

	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("creating plugins directory: %w", err)
	}

	// Extract to a temp directory first to read the manifest
	tmpDir, err := os.MkdirTemp("", "spx-install-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZip(spxPath, tmpDir); err != nil {
		return fmt.Errorf("extracting .spx archive: %w", err)
	}

	manifest, err := plugins.ParseManifest(tmpDir)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	if err := plugins.ValidateManifest(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Check that extension.js exists in the archive
	extensionPath := filepath.Join(tmpDir, "extension.js")
	if _, err := os.Stat(extensionPath); os.IsNotExist(err) {
		return fmt.Errorf("archive is missing extension.js")
	}

	// Move to final destination
	destDir := filepath.Join(pluginsDir, manifest.Name)

	// Remove old version if exists
	if _, err := os.Stat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("removing old version: %w", err)
		}
	}

	if err := os.Rename(tmpDir, destDir); err != nil {
		// os.Rename may fail across filesystems, fall back to copy
		if err := copyDir(tmpDir, destDir); err != nil {
			return fmt.Errorf("installing plugin: %w", err)
		}
	}

	// Update plugins.json
	pj, err := plugins.ReadPluginsJSON(pluginsDir)
	if err != nil {
		return fmt.Errorf("reading plugins.json: %w", err)
	}

	// Remove existing entry for this plugin name
	filtered := make([]plugins.PluginRecord, 0, len(pj.Plugins))
	for _, p := range pj.Plugins {
		if p.Name != manifest.Name {
			filtered = append(filtered, p)
		}
	}

	filtered = append(filtered, plugins.PluginRecord{
		Name:        manifest.Name,
		Version:     manifest.Version,
		InstalledAt: time.Now().UTC(),
	})

	pj.Plugins = filtered

	if err := plugins.WritePluginsJSON(pluginsDir, pj); err != nil {
		return fmt.Errorf("writing plugins.json: %w", err)
	}

	fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Installed %s v%s\n", manifest.Name, manifest.Version)

	for _, c := range manifest.SuperPlane.Contributes.Components {
		fmt.Fprintf(ctx.Cmd.OutOrStdout(), "  Registered component: %s\n", c.Name)
	}
	for _, t := range manifest.SuperPlane.Contributes.Triggers {
		fmt.Fprintf(ctx.Cmd.OutOrStdout(), "  Registered trigger: %s\n", t.Name)
	}

	if err := reloadPluginsViaAPI(ctx); err != nil {
		return err
	}

	return nil
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func reloadPluginsViaAPI(ctx core.CommandContext) error {
	config := ctx.API.GetConfig()
	if config == nil {
		return fmt.Errorf("api client config is required")
	}

	baseURL := ""
	if len(config.Servers) > 0 {
		baseURL = strings.TrimRight(config.Servers[0].URL, "/")
	}
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("api_url is required")
	}

	endpoint := baseURL + "/api/v1/plugins/reload"
	request, err := http.NewRequestWithContext(ctx.Context, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	if authorization := strings.TrimSpace(config.DefaultHeader["Authorization"]); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode >= http.StatusMultipleChoices {
		errorPayload := struct {
			Message string `json:"message"`
		}{}
		_ = json.Unmarshal(body, &errorPayload)
		if errorPayload.Message != "" {
			return errors.New(errorPayload.Message)
		}
		return fmt.Errorf("failed to reload plugins: %s", response.Status)
	}

	return nil
}
