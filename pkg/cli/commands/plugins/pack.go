package plugins

import (
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	pluginspkg "github.com/superplanehq/superplane/pkg/plugins"
)

func newPackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pack [directory]",
		Short: "Package a plugin into a .spx archive",
		Long:  "Bundle and package a plugin project into a .spx archive for distribution. Defaults to the current directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPack,
	}
}

func runPack(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	manifest, err := pluginspkg.ParseManifest(absDir)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	if err := pluginspkg.ValidateManifest(manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Bundle with esbuild
	mainFile := manifest.Main
	if mainFile == "" {
		mainFile = "dist/index.js"
	}

	mainPath := filepath.Join(absDir, mainFile)
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		// Try building first
		fmt.Fprintln(cmd.OutOrStdout(), "Building plugin...")
		buildCmd := exec.Command("npm", "run", "build")
		buildCmd.Dir = absDir
		buildCmd.Stdout = cmd.OutOrStdout()
		buildCmd.Stderr = cmd.OutOrStderr()
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		if _, err := os.Stat(mainPath); os.IsNotExist(err) {
			return fmt.Errorf("build did not produce %s", mainFile)
		}
	}

	bundlePath := filepath.Join(absDir, "extension.js")

	fmt.Fprintln(cmd.OutOrStdout(), "Bundling with esbuild...")
	esbuildCmd := exec.Command("npx", "esbuild",
		mainPath,
		"--bundle",
		"--platform=node",
		"--target=node18",
		"--format=cjs",
		"--external:@superplane/sdk",
		fmt.Sprintf("--outfile=%s", bundlePath),
	)
	esbuildCmd.Dir = absDir
	esbuildCmd.Stdout = cmd.OutOrStdout()
	esbuildCmd.Stderr = cmd.OutOrStderr()

	if err := esbuildCmd.Run(); err != nil {
		return fmt.Errorf("esbuild bundling failed: %w", err)
	}
	defer os.Remove(bundlePath)

	// Create .spx archive
	archiveName := fmt.Sprintf("%s-%s.spx", manifest.Name, manifest.Version)
	archivePath := filepath.Join(absDir, archiveName)

	if err := createSpxArchive(archivePath, absDir, bundlePath); err != nil {
		return fmt.Errorf("creating .spx archive: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", archiveName)
	return nil
}

func createSpxArchive(archivePath, projectDir, bundlePath string) error {
	outFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	// Add package.json
	packageJSON, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}

	pjWriter, err := w.Create("package.json")
	if err != nil {
		return err
	}
	if _, err := pjWriter.Write(packageJSON); err != nil {
		return err
	}

	// Add extension.js
	extensionJS, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("reading extension.js: %w", err)
	}

	ejWriter, err := w.Create("extension.js")
	if err != nil {
		return err
	}
	if _, err := ejWriter.Write(extensionJS); err != nil {
		return err
	}

	return nil
}
