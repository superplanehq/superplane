package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
)

func readAndCloseFile(path string) ([]byte, error) {
	// #nosec G304 - path comes from CLI flags supplied by the user.
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

// renderMapAsYAMLBlock prints a labeled, indented YAML block to stdout. It
// is used by `console panels get` and similar text renderers to show the
// panel content without overwhelming the user with raw JSON. Keys are
// rendered in sorted order so successive runs are stable.
func renderMapAsYAMLBlock(stdout io.Writer, label string, value map[string]any) error {
	if _, err := fmt.Fprintf(stdout, "%s:\n", label); err != nil {
		return err
	}
	if len(value) == 0 {
		_, err := fmt.Fprintln(stdout, "  (empty)")
		return err
	}

	ordered := orderedMap(value)
	jsonBytes, err := json.Marshal(ordered)
	if err != nil {
		return err
	}
	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(strings.TrimRight(string(yamlBytes), "\n"), "\n") {
		if _, err := fmt.Fprintf(stdout, "  %s\n", line); err != nil {
			return err
		}
	}
	return nil
}

// orderedMap recursively produces a JSON-serializable structure with keys
// sorted alphabetically so the output is stable.
func orderedMap(value any) any {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(v))
		for _, k := range keys {
			ordered[k] = orderedMap(v[k])
		}
		return ordered
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = orderedMap(item)
		}
		return out
	default:
		return v
	}
}
