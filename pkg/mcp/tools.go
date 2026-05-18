package mcp

import (
	"context"
	"bytes"
	"encoding/json"
	"net/http"
	"io"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gopkg.in/yaml.v3"
	"time"
)

// safeTime returns a formatted time string or empty string for nil pointers
func safeTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

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
		"id":              canvas.ID.String(),
		"organization_id": canvas.OrganizationID.String(),
		"name":            canvas.Name,
		"description":     canvas.Description,
		"created_at":      safeTime(canvas.CreatedAt),
		"updated_at":      safeTime(canvas.UpdatedAt),
		"version": map[string]interface{}{
			"id":                        liveVersion.ID.String(),
			"state":                     liveVersion.State,
			"name":                      liveVersion.Name,
			"description":               liveVersion.Description,
			"change_management_enabled": liveVersion.ChangeManagementEnabled,
			"nodes":                     liveVersion.Nodes,
			"edges":                     liveVersion.Edges,
			"created_at":                safeTime(liveVersion.CreatedAt),
			"updated_at":                safeTime(liveVersion.UpdatedAt),
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
			"created_at": safeTime(v.CreatedAt),
			"updated_at": safeTime(v.UpdatedAt),
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
			"created_at":        safeTime(integration.CreatedAt),
			"updated_at":        safeTime(integration.UpdatedAt),
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

// handleCanvasUpdate updates a canvas draft version with new YAML content
func handleCanvasUpdate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	canvasID, ok := args["canvas_id"].(string)
	if !ok || canvasID == "" {
		return nil, fmt.Errorf("canvas_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	yamlContent, ok := args["yaml_content"].(string)
	if !ok || yamlContent == "" {
		return nil, fmt.Errorf("yaml_content is required")
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

	// Verify canvas exists
	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// Parse YAML content
	var canvasData map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &canvasData); err != nil {
		return nil, fmt.Errorf("invalid YAML content: %w", err)
	}

	// Call our own API to create/update the draft version.
	// The API expects the YAML to have apiVersion/kind/metadata/spec structure.
	// If the agent sends raw nodes/edges, wrap them.
	apiURL := fmt.Sprintf("http://localhost:8000/api/v1/canvases/%s/versions", canvasID)

	// Ensure YAML has proper structure
	if _, hasApiVersion := canvasData["apiVersion"]; !hasApiVersion {
		// Wrap raw canvas data in proper structure
		wrapped := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Canvas",
			"metadata": map[string]interface{}{
				"id":   canvasID,
				"name": canvasData["name"],
			},
			"spec": map[string]interface{}{
				"nodes": canvasData["nodes"],
				"edges": canvasData["edges"],
			},
		}
		yamlBytes, err := yaml.Marshal(wrapped)
		if err != nil {
			return nil, fmt.Errorf("failed to re-serialize YAML: %w", err)
		}
		yamlContent = string(yamlBytes)
	}

	// Build the update request
	updateBody := map[string]interface{}{
		"yaml":  yamlContent,
		"draft": true,
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return nil, fmt.Errorf("marshal update body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-organization-id", orgID)
	if bearerToken, ok := ctx.Value("bearer_token").(string); ok && bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("canvas update failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var updateResp map[string]interface{}
	json.Unmarshal(respBody, &updateResp)

	result := map[string]interface{}{
		"success":   true,
		"canvas_id": canvas.ID.String(),
		"message":   "Draft version created/updated",
		"response":  updateResp,
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize result: %w", err)
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

// handleIndexSearch searches the component registry
func handleIndexSearch(ctx context.Context, reg *registry.Registry, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	typeFilter := ""
	if t, ok := args["type"].(string); ok {
		typeFilter = t
	}

	queryLower := ""
	for _, r := range query {
		if r >= 'A' && r <= 'Z' {
			queryLower += string(r + 32)
		} else {
			queryLower += string(r)
		}
	}

	results := []map[string]interface{}{}

	// Search triggers
	if typeFilter == "" || typeFilter == "trigger" {
		for _, trigger := range reg.ListTriggers() {
			nameLower := ""
			for _, r := range trigger.Name() {
				if r >= 'A' && r <= 'Z' {
					nameLower += string(r + 32)
				} else {
					nameLower += string(r)
				}
			}
			if containsSubstring(nameLower, queryLower) {
				results = append(results, map[string]interface{}{
					"name":        trigger.Name(),
					"type":        "trigger",
					"description": trigger.Description(),
				})
			}
		}
	}

	// Search actions
	if typeFilter == "" || typeFilter == "action" {
		for _, action := range reg.ListActions() {
			nameLower := ""
			for _, r := range action.Name() {
				if r >= 'A' && r <= 'Z' {
					nameLower += string(r + 32)
				} else {
					nameLower += string(r)
				}
			}
			if containsSubstring(nameLower, queryLower) {
				results = append(results, map[string]interface{}{
					"name":        action.Name(),
					"type":        "action",
					"description": action.Description(),
				})
			}
		}
	}

	output, err := json.MarshalIndent(map[string]interface{}{
		"query":   query,
		"type":    typeFilter,
		"results": results,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize results: %w", err)
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

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// handleIndexGetSchema retrieves the full schema for a specific component
func handleIndexGetSchema(ctx context.Context, reg *registry.Registry, args map[string]interface{}) (interface{}, error) {
	name, ok := args["component_name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("component_name is required")
	}

	// Try to find as action first
	action, err := reg.GetAction(name)
	if err == nil {
		schema := map[string]interface{}{
			"name":        action.Name(),
			"type":        "action",
			"description": action.Description(),
			"config":      action.Configuration(),
		}

		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize schema: %w", err)
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

	// Try to find as trigger
	trigger, err := reg.GetTrigger(name)
	if err == nil {
		schema := map[string]interface{}{
			"name":        trigger.Name(),
			"type":        "trigger",
			"description": trigger.Description(),
			"config":      trigger.Configuration(),
		}

		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize schema: %w", err)
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

	return nil, fmt.Errorf("component not found: %s", name)
}

// handleIntegrationsGet retrieves details for a specific integration
func handleIntegrationsGet(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	integrationID, ok := args["integration_id"].(string)
	if !ok || integrationID == "" {
		return nil, fmt.Errorf("integration_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	// Parse UUIDs
	integrationUUID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, fmt.Errorf("invalid integration_id: %w", err)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Find integration
	integration, err := models.FindIntegration(orgUUID, integrationUUID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	// Build response (sanitize secrets)
	result := map[string]interface{}{
		"id":                integration.ID.String(),
		"organization_id":   integration.OrganizationID.String(),
		"app_name":          integration.AppName,
		"installation_name": integration.InstallationName,
		"state":             integration.State,
		"state_description": integration.StateDescription,
		"created_at":        safeTime(integration.CreatedAt),
		"updated_at":        safeTime(integration.UpdatedAt),
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize integration: %w", err)
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

// handleRunsList lists recent runs for a canvas
func handleRunsList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	canvasID, ok := args["canvas_id"].(string)
	if !ok || canvasID == "" {
		return nil, fmt.Errorf("canvas_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
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

	// Verify canvas exists
	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// List runs
	runs, err := models.ListCanvasRuns(canvasUUID, limit, nil, models.CanvasRunFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}

	// Build response
	runList := make([]map[string]interface{}, len(runs))
	for i, run := range runs {
		runList[i] = map[string]interface{}{
			"run_id":     run.ID.String(),
			"state":      run.State,
			"result":     run.Result,
			"created_at": safeTime(run.CreatedAt),
			"updated_at": safeTime(run.UpdatedAt),
		}
		if run.FinishedAt != nil {
			runList[i]["finished_at"] = safeTime(run.FinishedAt)
		}
	}

	output, err := json.MarshalIndent(map[string]interface{}{
		"canvas_id": canvasID,
		"org_id":    orgID,
		"runs":      runList,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize runs: %w", err)
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

// handleRunGet retrieves details for a specific run
func handleRunGet(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	runID, ok := args["run_id"].(string)
	if !ok || runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}

	canvasID, ok := args["canvas_id"].(string)
	if !ok || canvasID == "" {
		return nil, fmt.Errorf("canvas_id is required")
	}

	orgID, ok := args["org_id"].(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("org_id is required")
	}

	// Parse UUIDs
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("invalid run_id: %w", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas_id: %w", err)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Verify canvas exists
	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// Find run
	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvasUUID, runUUID)
	if err != nil {
		return nil, fmt.Errorf("run not found: %w", err)
	}

	// Build response
	result := map[string]interface{}{
		"run_id":      run.ID.String(),
		"workflow_id": run.WorkflowID.String(),
		"state":       run.State,
		"result":      run.Result,
		"created_at":  safeTime(run.CreatedAt),
		"updated_at":  safeTime(run.UpdatedAt),
	}
	if run.FinishedAt != nil {
		result["finished_at"] = safeTime(run.FinishedAt)
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize run: %w", err)
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
