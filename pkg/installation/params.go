package installation

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const paramsFileName = "params.json"

// InstallParam defines a single parameter for the install wizard.
type InstallParam struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "string" or "integration-resource"
	Placeholder string `json:"placeholder,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`

	// For type "integration-resource"
	Integration  string `json:"integration,omitempty"`  // integration type name (e.g. "digitalocean")
	ResourceType string `json:"resourceType,omitempty"` // resource type (e.g. "region", "size", "image")
}

// ParamsFile is the structure of params.json in the app repo.
type ParamsFile struct {
	InstallParams []InstallParam `json:"install_params"`
}

// FetchParams loads and parses the optional params.json from the app repo.
// Returns nil if the file doesn't exist (params are optional).
func FetchParams(repo *Repository, ref string) (*ParamsFile, error) {
	if ref == "" {
		return nil, fmt.Errorf("params fetch requires a resolved ref")
	}

	body, err := fetchURL(rawFileURL(repo, ref, paramsFileName))
	if err != nil {
		if errors.Is(err, errFileNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var params ParamsFile
	if err := json.Unmarshal(body, &params); err != nil {
		return nil, fmt.Errorf("parse params.json: %w", err)
	}

	return &params, nil
}

// ValidateInstallParams checks that all required params have values.
func ValidateInstallParams(schema []InstallParam, values map[string]string) error {
	for _, p := range schema {
		val, ok := values[p.Name]
		if !ok || strings.TrimSpace(val) == "" {
			if p.Required && p.Default == "" {
				return fmt.Errorf("install parameter %q is required", p.Name)
			}
		}
	}
	return nil
}

// ResolveInstallParams merges user values with defaults from the schema.
func ResolveInstallParams(schema []InstallParam, values map[string]string) map[string]string {
	resolved := make(map[string]string, len(schema))
	for _, p := range schema {
		if val, ok := values[p.Name]; ok && strings.TrimSpace(val) != "" {
			resolved[p.Name] = val
		} else if p.Default != "" {
			resolved[p.Name] = p.Default
		} else if p.Placeholder != "" {
			resolved[p.Name] = p.Placeholder
		} else {
			// Always resolve every param so no {{ install_params.xxx }} tokens remain.
			resolved[p.Name] = p.Name
		}
	}
	return resolved
}

var installParamPattern = regexp.MustCompile(`\{\{\s*install_params\.(\w+)\s*\}\}`)

// SubstituteInstallParams replaces {{ install_params.xxx }} placeholders
// in the given YAML content with the resolved parameter values.
func SubstituteInstallParams(content []byte, params map[string]string) []byte {
	return installParamPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		groups := installParamPattern.FindSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		name := string(groups[1])
		if val, ok := params[name]; ok {
			return []byte(val)
		}
		return match // leave unresolved placeholders as-is
	})
}
