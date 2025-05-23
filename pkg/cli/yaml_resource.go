package cli

import (
	"fmt"

	"github.com/ghodss/yaml"
)

// returns tuple (apiVersion, kind, error)
func ParseYamlResourceHeaders(raw []byte) (string, string, error) {
	m := make(map[string]interface{})

	err := yaml.Unmarshal(raw, &m)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse resource; %s", err)
	}

	apiVersion, ok := m["apiVersion"].(string)

	if !ok {
		return "", "", fmt.Errorf("failed to parse resource's api version")
	}

	kind, ok := m["kind"].(string)

	if !ok {
		return "", "", fmt.Errorf("failed to parse resource's kind")
	}

	return apiVersion, kind, nil
}
