package models

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/openapi_client"
	"gopkg.in/yaml.v3"
)

const (
	ExtensionVersionKind = "ExtensionVersion"
)

type ExtensionVersion struct {
	APIVersion string                                             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                                             `json:"kind" yaml:"kind"`
	Metadata   *openapi_client.ExtensionsExtensionVersionMetadata `json:"metadata" yaml:"metadata"`
	Status     *openapi_client.ExtensionsExtensionVersionStatus   `json:"status" yaml:"status"`
}

func ParseExtensionVersion(raw []byte) (*ExtensionVersion, error) {
	var yamlObject any
	if err := yaml.Unmarshal(raw, &yamlObject); err != nil {
		return nil, fmt.Errorf("failed to parse extension version resource: %w", err)
	}

	jsonData, err := json.Marshal(yamlObject)
	if err != nil {
		return nil, fmt.Errorf("failed to convert extension version resource to json: %w", err)
	}

	var resource ExtensionVersion
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse extension version json payload: %w", err)
	}

	if resource.Kind != ExtensionVersionKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("extension version apiVersion is required")
	}

	if resource.Metadata == nil {
		return nil, fmt.Errorf("extension version metadata is required")
	}

	if resource.Metadata.Id == nil {
		return nil, fmt.Errorf("extension version metadata.id is required")
	}

	return &resource, nil
}

func ExtensionVersionFromExtensionVersion(resource ExtensionVersion) openapi_client.ExtensionsExtensionVersion {
	version := openapi_client.ExtensionsExtensionVersion{}
	version.SetMetadata(*resource.Metadata)
	version.SetStatus(*resource.Status)
	return version
}
