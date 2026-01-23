package cli

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/openapi_client"
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

const (
	canvasAPIVersion = "v1"
	canvasKind       = "Canvas"
)

type CanvasResource struct {
	APIVersion string                                    `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                                    `json:"kind" yaml:"kind"`
	Metadata   *openapi_client.WorkflowsWorkflowMetadata `json:"metadata" yaml:"metadata"`
	Spec       *openapi_client.WorkflowsWorkflowSpec     `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func ParseCanvasResource(raw []byte) (*CanvasResource, error) {
	var resource CanvasResource
	if err := yaml.Unmarshal(raw, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse canvas resource: %w", err)
	}

	if resource.Kind != canvasKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("canvas apiVersion is required")
	}

	if resource.Metadata.Name == nil {
		return nil, fmt.Errorf("canvas metadata.name is required")
	}

	return &resource, nil
}

func WorkflowFromCanvasResource(resource CanvasResource) openapi_client.WorkflowsWorkflow {
	workflow := openapi_client.WorkflowsWorkflow{}
	metadata := openapi_client.WorkflowsWorkflowMetadata{}
	metadata.SetName(*resource.Metadata.Name)
	if resource.Metadata.Description != nil {
		metadata.SetDescription(*resource.Metadata.Description)
	}
	if resource.Metadata.Id != nil {
		metadata.SetId(*resource.Metadata.Id)
	}

	workflow.SetMetadata(metadata)
	if resource.Spec != nil {
		workflow.SetSpec(*resource.Spec)
	} else {
		workflow.SetSpec(*EmptyWorkflowSpec())
	}

	return workflow
}

func CanvasResourceFromWorkflow(workflow openapi_client.WorkflowsWorkflow) CanvasResource {
	return CanvasResource{
		APIVersion: canvasAPIVersion,
		Kind:       canvasKind,
		Metadata:   workflow.Metadata,
		Spec:       workflow.Spec,
	}
}

func EmptyWorkflowSpec() *openapi_client.WorkflowsWorkflowSpec {
	return &openapi_client.WorkflowsWorkflowSpec{
		Nodes: []openapi_client.ComponentsNode{},
		Edges: []openapi_client.ComponentsEdge{},
	}
}
