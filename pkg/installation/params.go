package installation

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const paramsFileName = "params.json"

// Supported InstallParam.Type values. New types added here must be recognised
// by the install UI as well (see web_src/src/pages/install/types.ts).
const (
	ParamTypeString              = "string"
	ParamTypeIntegrationResource = "integration-resource"
	ParamTypeSecretPicker        = "secret_picker"
)

// InstallParam defines a single parameter for the install wizard.
type InstallParam struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // "string", "integration-resource", or "secret_picker"
	Placeholder string `json:"placeholder,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`

	// For type "integration-resource"
	Integration    string `json:"integration,omitempty"`    // integration type name (e.g. "digitalocean")
	ResourceType   string `json:"resourceType,omitempty"`   // resource type (e.g. "region", "size", "image")
	UseNameAsValue bool   `json:"useNameAsValue,omitempty"` // when true, substitute the resource name instead of the ID
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

// ValidateSecretPickerParams confirms that every "secret_picker" parameter
// value names an organization secret that actually exists, so installs cannot
// reference deleted/typoed credentials and surface a confusing failure later
// at node-execution time.
func ValidateSecretPickerParams(schema []InstallParam, values map[string]string, organizationID uuid.UUID) error {
	for _, p := range schema {
		if p.Type != ParamTypeSecretPicker {
			continue
		}

		// Only validate a real, intended secret reference: a user-provided
		// value, or an explicit default. The placeholder/param-name fallbacks
		// applied by ResolveInstallParams are not valid secret names, so an
		// optional picker left empty must not be treated as a missing secret.
		secretName := strings.TrimSpace(values[p.Name])
		if secretName == "" {
			secretName = strings.TrimSpace(p.Default)
		}
		if secretName == "" {
			continue
		}

		_, err := models.FindSecretByName(models.DomainTypeOrganization, organizationID, secretName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("install parameter %q: secret %q not found in organization", p.Name, secretName)
			}
			return fmt.Errorf("install parameter %q: failed to verify secret %q: %w", p.Name, secretName, err)
		}
	}
	return nil
}

// ResolveInstallParams merges user values with schema defaults.
// Every schema param is resolved — user value takes priority, then
// the same default → placeholder → name fallback as DefaultParamValues.
func ResolveInstallParams(schema []InstallParam, values map[string]string) map[string]string {
	resolved := DefaultParamValues(schema)
	for _, p := range schema {
		if val, ok := values[p.Name]; ok && strings.TrimSpace(val) != "" {
			resolved[p.Name] = val
		}
	}
	return resolved
}

// DefaultParamValues builds a fallback map from the schema using
// default → placeholder → param name, in that priority order.
//
// secret_picker params are special: they must resolve to a real organization
// secret name, so they only fall back to an explicit default. The placeholder
// and param-name fallbacks are not valid secret names, and substituting them
// into canvas.yaml would silently inject a bogus credential reference that
// passes install validation but fails later at node-execution time. An empty
// optional picker therefore resolves to an empty string (no substitution).
func DefaultParamValues(schema []InstallParam) map[string]string {
	defaults := make(map[string]string, len(schema))
	for _, p := range schema {
		if p.Default != "" {
			defaults[p.Name] = p.Default
		} else if p.Type == ParamTypeSecretPicker {
			defaults[p.Name] = ""
		} else if p.Placeholder != "" {
			defaults[p.Name] = p.Placeholder
		} else {
			defaults[p.Name] = p.Name
		}
	}
	return defaults
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
