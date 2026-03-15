package models

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/openapi_client"
	"gopkg.in/yaml.v3"
)

const (
	ExtensionKind = "Extension"
)

type Extension struct {
	APIVersion string                                      `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                                      `json:"kind" yaml:"kind"`
	Metadata   *openapi_client.ExtensionsExtensionMetadata `json:"metadata" yaml:"metadata"`
}

func ParseExtension(raw []byte) (*Extension, error) {
	var yamlObject any
	if err := yaml.Unmarshal(raw, &yamlObject); err != nil {
		return nil, fmt.Errorf("failed to parse canvas resource: %w", err)
	}

	jsonData, err := json.Marshal(yamlObject)
	if err != nil {
		return nil, fmt.Errorf("failed to convert canvas resource to json: %w", err)
	}

	var resource Extension
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse canvas json payload: %w", err)
	}

	if resource.Kind != ExtensionKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("extension apiVersion is required")
	}

	if resource.Metadata == nil {
		return nil, fmt.Errorf("extension metadata is required")
	}

	if resource.Metadata.Name == nil {
		return nil, fmt.Errorf("extension metadata.name is required")
	}

	return &resource, nil
}

func ExtensionFromExtension(resource Extension) openapi_client.ExtensionsExtension {
	extension := openapi_client.ExtensionsExtension{}
	extension.SetMetadata(*resource.Metadata)
	return extension
}

func ExtensionResourceFromExtension(extension openapi_client.ExtensionsExtension) Extension {
	return Extension{
		APIVersion: "v1",
		Kind:       ExtensionKind,
		Metadata:   extension.Metadata,
	}
}
