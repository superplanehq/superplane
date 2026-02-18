package jsruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/registry"
)

const jsComponentPrefix = "js."
const maxCodeSize = 256 * 1024

var validFilenamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.js$`)

// LoadComponents scans a directory for .js files and registers each valid one as a component
// in the registry. Returns the list of registry names that were successfully loaded.
func LoadComponents(dir string, rt *Runtime, reg *registry.Registry) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read components directory %s: %w", dir, err)
	}

	var loaded []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		registryName, err := loadComponentFile(dir, entry.Name(), rt, reg)
		if err != nil {
			log.WithError(err).Errorf("Failed to load JS component %s", entry.Name())
			continue
		}

		loaded = append(loaded, registryName)
	}

	return loaded, nil
}

// loadComponentFile reads, parses, and registers a single JS component file.
// Returns the registry name on success.
func loadComponentFile(dir, filename string, rt *Runtime, reg *registry.Registry) (string, error) {
	if !validFilenamePattern.MatchString(filename) {
		return "", fmt.Errorf(
			"invalid filename %q: must be lowercase alphanumeric with hyphens (e.g., my-component.js)",
			filename,
		)
	}

	path := filepath.Join(dir, filename)
	source, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if len(source) > maxCodeSize {
		return "", fmt.Errorf("file exceeds maximum code size of %d bytes", maxCodeSize)
	}

	def, err := rt.ParseDefinition(string(source))
	if err != nil {
		return "", fmt.Errorf("failed to parse: %w", err)
	}

	baseName := strings.TrimSuffix(filename, ".js")
	registryName := jsComponentPrefix + baseName

	if def.Label == "" {
		return "", fmt.Errorf("component must define a label")
	}

	adapter, err := NewJSComponentAdapter(rt, string(source), def, registryName)
	if err != nil {
		return "", fmt.Errorf("failed to create adapter: %w", err)
	}

	reg.RegisterJSComponent(registryName, adapter)

	return registryName, nil
}

// RegistryNameFromFilename converts a JS filename to its registry name (e.g., "transform.js"
// becomes "js.transform").
func RegistryNameFromFilename(filename string) string {
	return jsComponentPrefix + strings.TrimSuffix(filename, ".js")
}
