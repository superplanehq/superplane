package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"
)

func ValidateCanvasTemplates(registry *registry.Registry, dir string) error {
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
