package plugins

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/plugins"
)

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install <file.spx>",
		Short: "Install a plugin from a .spx archive",
		Args:  cobra.ExactArgs(1),
		RunE:  runInstall,
	}
}

func runInstall(cmd *cobra.Command, args []string) error {
	spxPath := args[0]

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

	fmt.Fprintf(cmd.OutOrStdout(), "Installed %s v%s\n", manifest.Name, manifest.Version)

	for _, c := range manifest.SuperPlane.Contributes.Components {
		fmt.Fprintf(cmd.OutOrStdout(), "  Registered component: %s\n", c.Name)
	}
	for _, t := range manifest.SuperPlane.Contributes.Triggers {
		fmt.Fprintf(cmd.OutOrStdout(), "  Registered trigger: %s\n", t.Name)
	}

	// Signal the server to reload
	signalServer(cmd)

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

func signalServer(cmd *cobra.Command) {
	pidFile := os.Getenv("SUPERPLANE_PID_FILE")
	if pidFile == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Server signaled to reload plugins (SIGHUP)")
		return
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Could not read PID file, restart the server to load plugins\n")
		return
	}

	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Invalid PID file, restart the server to load plugins\n")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Could not find server process, restart the server to load plugins\n")
		return
	}

	if err := proc.Signal(syscall.SIGHUP); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Could not signal server, restart the server to load plugins\n")
		return
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Server signaled to reload plugins")
}
