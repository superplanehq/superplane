package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
)

type PackageVersionCommand struct {
	Destination string
	EntryPoint  string
}

func (c *PackageVersionCommand) Execute(ctx core.CommandContext) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}

	entryPoint, err := resolveEntryPoint(projectDir, c.EntryPoint)
	if err != nil {
		return err
	}

	artifacts, err := buildExtensionVersionArtifacts(ctx.Context, projectDir, entryPoint)
	if err != nil {
		return err
	}

	destination, err := resolvePackageDestination(c.Destination)
	if err != nil {
		return err
	}

	if err := writeExtensionVersionArtifacts(destination, artifacts); err != nil {
		return err
	}

	if ctx.Renderer.IsText() {
		_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Wrote packaged extension version to %s\n", destination)
	}

	return nil
}

func resolvePackageDestination(destination string) (string, error) {
	target := strings.TrimSpace(destination)
	if target == "" {
		target = "./dist"
	}

	info, err := os.Stat(target)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat destination %q: %w", target, err)
	}

	if err == nil && !info.IsDir() {
		return "", fmt.Errorf("destination %q must be a directory", target)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("create destination directory %q: %w", target, err)
	}

	return filepath.Clean(target), nil
}

func formatManifestJSON(raw []byte) ([]byte, error) {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse manifest json: %w", err)
	}

	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("format manifest json: %w", err)
	}

	return append(formatted, '\n'), nil
}
