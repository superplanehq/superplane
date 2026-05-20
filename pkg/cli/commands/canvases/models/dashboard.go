package models

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	DashboardKind       = "Dashboard"
	DashboardAPIVersion = "v1"
)

type DashboardMetadata struct {
	CanvasID string `json:"canvasId,omitempty" yaml:"canvasId,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
}

type DashboardSpec struct {
	Panels []openapi_client.CanvasesDashboardPanel      `json:"panels" yaml:"panels"`
	Layout []openapi_client.CanvasesDashboardLayoutItem `json:"layout" yaml:"layout"`
}

type Dashboard struct {
	APIVersion string            `json:"apiVersion" yaml:"apiVersion"`
	Kind       string            `json:"kind" yaml:"kind"`
	Metadata   DashboardMetadata `json:"metadata" yaml:"metadata"`
	Spec       DashboardSpec     `json:"spec" yaml:"spec"`
}

func ParseDashboard(raw []byte) (*Dashboard, error) {
	var resource Dashboard
	if err := core.NewDecoder(raw).DecodeYAML(&resource); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard yaml: %w", err)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("dashboard apiVersion is required")
	}
	if resource.APIVersion != DashboardAPIVersion {
		return nil, fmt.Errorf("unsupported dashboard apiVersion %q", resource.APIVersion)
	}
	if resource.Kind != DashboardKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.Spec.Panels == nil {
		resource.Spec.Panels = []openapi_client.CanvasesDashboardPanel{}
	}
	if resource.Spec.Layout == nil {
		resource.Spec.Layout = []openapi_client.CanvasesDashboardLayoutItem{}
	}

	return &resource, nil
}

func DashboardResourceFromDashboard(dashboard openapi_client.CanvasesCanvasDashboard, canvasName string) Dashboard {
	return Dashboard{
		APIVersion: DashboardAPIVersion,
		Kind:       DashboardKind,
		Metadata: DashboardMetadata{
			CanvasID: dashboard.GetCanvasId(),
			Name:     canvasName,
		},
		Spec: DashboardSpec{
			Panels: dashboard.GetPanels(),
			Layout: dashboard.GetLayout(),
		},
	}
}

func UpdateDashboardRequestFromDashboard(resource Dashboard) openapi_client.CanvasesUpdateCanvasDashboardBody {
	body := openapi_client.CanvasesUpdateCanvasDashboardBody{}
	body.SetPanels(resource.Spec.Panels)
	body.SetLayout(resource.Spec.Layout)
	return body
}
