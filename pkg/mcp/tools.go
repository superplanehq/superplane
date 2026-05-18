package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gopkg.in/yaml.v3"
)

// handleCanvasGet retrieves a canvas and returns it in YAML or JSON format
func handleCanvasGet(ctx context.Context, reg *registry.Registry, args map[string]interface{}) (interface{}, error) {
	canvasID, ok := args["canvas_id"].(string)
	if !ok || canvasID == "" {
		return nil, fmt.Errorf("canvas_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	format := "yaml"
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}

	// Parse UUIDs
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas_id: %w", err)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Fetch canvas
	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// Fetch live version
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch live canvas version: %w", err)
	}

	// Build canvas data structure
	canvasData := map[string]interface{}{
		"id":             canvas.ID.String(),
		"organization_id": canvas.OrganizationID.String(),
		"name":           canvas.Name,
		"description":    canvas.Description,
		"created_at":     canvas.CreatedAt,
		"updated_at":     canvas.UpdatedAt,
		"version": map[string]interface{}{
			"id":                        liveVersion.ID.String(),
			"state":                     liveVersion.State,
			"name":                      liveVersion.Name,
			"description":               liveVersion.Description,
			"change_management_enabled": liveVersion.ChangeManagementEnabled,
			"nodes":                     liveVersion.Nodes,
			"edges":                     liveVersion.Edges,
			"created_at":                liveVersion.CreatedAt,
			"updated_at":                liveVersion.UpdatedAt,
		},
	}

	// Serialize to requested format
	var outputBytes []byte
	var contentType string

	switch format {
	case "json":
		outputBytes, err = json.MarshalIndent(canvasData, "", "  ")
		contentType = "application/json"
	case "yaml":
		outputBytes, err = yaml.Marshal(canvasData)
		contentType = "application/x-yaml"
	default:
		return nil, fmt.Errorf("unsupported format: %s (use 'yaml' or 'json')", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to serialize canvas: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type":     "text",
				"text":     string(outputBytes),
				"mimeType": contentType,
			},
		},
	}, nil
}

// handleCanvasListVersions lists all versions of a canvas
func handleCanvasListVersions(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	canvasID, ok := args["canvas_id"].(string)
	if !ok || canvasID == "" {
		return nil, fmt.Errorf("canvas_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	// Parse UUIDs
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas_id: %w", err)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Verify canvas exists and belongs to org
	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// List versions
	versions, err := models.ListCanvasVersions(canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list canvas versions: %w", err)
	}

	// Build response
	versionList := make([]map[string]interface{}, len(versions))
	for i, v := range versions {
		versionList[i] = map[string]interface{}{
			"id":         v.ID.String(),
			"state":      v.State,
			"name":       v.Name,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		}
		if v.PublishedAt != nil {
			versionList[i]["published_at"] = v.PublishedAt
		}
	}

	output, err := json.MarshalIndent(map[string]interface{}{
		"canvas_id": canvasID,
		"org_id":    orgID,
		"versions":  versionList,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize versions: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(output),
			},
		},
	}, nil
}

// handleIntegrationsList lists all integrations for an organization
func handleIntegrationsList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	// Parse UUID
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// List integrations
	integrations, err := models.ListIntegrations(orgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}

	// Build response
	integrationList := make([]map[string]interface{}, len(integrations))
	for i, integration := range integrations {
		integrationList[i] = map[string]interface{}{
			"id":                integration.ID.String(),
			"app_name":          integration.AppName,
			"installation_name": integration.InstallationName,
			"state":             integration.State,
			"state_description": integration.StateDescription,
			"created_at":        integration.CreatedAt,
			"updated_at":        integration.UpdatedAt,
		}
	}

	output, err := json.MarshalIndent(map[string]interface{}{
		"org_id":       orgID,
		"integrations": integrationList,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize integrations: %w", err)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(output),
			},
		},
	}, nil
}
