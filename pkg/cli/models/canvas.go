package models

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	CanvasKind = "Canvas"
)

type Canvas struct {
	APIVersion string                                    `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                                    `json:"kind" yaml:"kind"`
	Metadata   *openapi_client.WorkflowsWorkflowMetadata `json:"metadata" yaml:"metadata"`
	Spec       *openapi_client.WorkflowsWorkflowSpec     `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func ParseCanvas(raw []byte) (*Canvas, error) {
	var resource Canvas
	if err := yaml.Unmarshal(raw, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse canvas resource: %w", err)
	}

	if resource.Kind != CanvasKind {
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

func WorkflowFromCanvas(resource Canvas) openapi_client.WorkflowsWorkflow {
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

func CanvasResourceFromWorkflow(workflow openapi_client.WorkflowsWorkflow) Canvas {
	return Canvas{
		APIVersion: "v1",
		Kind:       CanvasKind,
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
