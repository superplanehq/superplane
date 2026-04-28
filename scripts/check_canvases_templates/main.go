package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"

	// Import server to auto-register all integrations, components, and triggers via init().
	_ "github.com/superplanehq/superplane/pkg/server"
)

func main() {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		exitWithError(err)
	}

	dir := filepath.Join("templates", "canvases")
	if err := validateCanvasTemplates(reg, dir); err != nil {
		exitWithError(err)
	}
}

func validateCanvasTemplates(registry *registry.Registry, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		if err := validateCanvasTemplate(registry, filepath.Join(dir, entry.Name()), entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func validateCanvasTemplate(registry *registry.Registry, path string, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return fmt.Errorf("%s: yaml to json: %w", name, err)
	}

	var canvas pb.Canvas
	if err := protojson.Unmarshal(jsonData, &canvas); err != nil {
		return fmt.Errorf("%s: protojson: %w", name, err)
	}

	if canvas.Metadata == nil {
		return fmt.Errorf("%s: missing metadata", name)
	}

	if canvas.Metadata.Name == "" {
		return fmt.Errorf("%s: missing name", name)
	}

	canvas.Metadata.IsTemplate = true
	if _, _, err := canvases.ParseCanvas(registry, models.TemplateOrganizationID.String(), &canvas); err != nil {
		return fmt.Errorf("%s: ParseCanvas: %w", name, err)
	}

	return nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
