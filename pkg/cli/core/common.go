package core

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	APIVersion = "v1"
)

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

func ParseIntegrationScopedName(name string) (string, string, bool) {
	integrationName, resourceName, hasDot := strings.Cut(name, ".")
	if !hasDot || integrationName == "" || resourceName == "" {
		return "", "", false
	}

	return integrationName, resourceName, true
}

func FindIntegrationDefinition(ctx CommandContext, name string) (openapi_client.IntegrationsIntegrationDefinition, error) {
	response, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(ctx.Context).Execute()
	if err != nil {
		return openapi_client.IntegrationsIntegrationDefinition{}, err
	}

	for _, integration := range response.GetIntegrations() {
		if integration.GetName() == name {
			return integration, nil
		}
	}

	return openapi_client.IntegrationsIntegrationDefinition{}, fmt.Errorf("integration %q not found", name)
}

func ResolveCanvasID(ctx CommandContext, canvasID string) (string, error) {
	canvasID = strings.TrimSpace(canvasID)
	if canvasID != "" {
		return canvasID, nil
	}

	activeCanvas := strings.TrimSpace(ctx.Config.GetActiveCanvas())
	if activeCanvas == "" {
		return "", fmt.Errorf("canvas id is required; pass --canvas-id or set one with \"superplane canvases active\"")
	}

	return activeCanvas, nil
}
