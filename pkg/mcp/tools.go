package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gopkg.in/yaml.v3"
)

// safeTime returns a formatted time string or empty string for nil pointers
func safeTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// handleCanvasGet retrieves a canvas using the CLI for consistent YAML format.
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

	// Validate UUIDs
	if _, err := uuid.Parse(canvasID); err != nil {
		return nil, fmt.Errorf("invalid canvas_id: %w", err)
	}
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Mint JWT for CLI auth
	authToken := ""
	if bt, ok := ctx.Value("bearer_token").(string); ok {
		authToken = bt
	}
	if signer, ok := ctx.Value("jwt_signer").(*jwt.Signer); ok && signer != nil {
		if _, verr := signer.ValidateAndGetClaims(authToken); verr != nil {
			orgUUID, _ := uuid.Parse(orgID)
			canvasUUID, _ := uuid.Parse(canvasID)
			if canvas, err := models.FindCanvas(orgUUID, canvasUUID); err == nil && canvas.CreatedBy != nil {
				claims := jwt.ScopedTokenClaims{
					Subject: canvas.CreatedBy.String(),
					OrgID:   orgID,
					Purpose: "mcp-canvas-get",
					Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
						{ResourceType: "org", Action: "read"},
						{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
					}),
				}
				if minted, merr := signer.GenerateScopedToken(claims, 5*time.Minute); merr == nil {
					authToken = minted
				}
			}
		}
	}

	// Use CLI for consistent output format
	cliBin := "/tmp/sp-cli"
	if _, err := os.Stat(cliBin); os.IsNotExist(err) {
		cliBin = "superplane"
	}

	outputFlag := "-o"
	outputFormat := format
	if format == "" {
		outputFormat = "yaml"
	}

	// Try draft first, fall back to live
	cmd := exec.CommandContext(ctx, cliBin, "canvases", "get", canvasID, "--draft", outputFlag, outputFormat)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SUPERPLANE_URL=http://localhost:8000"),
		fmt.Sprintf("SUPERPLANE_TOKEN=%s", authToken),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without --draft
		cmd = exec.CommandContext(ctx, cliBin, "canvases", "get", canvasID, outputFlag, outputFormat)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("SUPERPLANE_URL=http://localhost:8000"),
			fmt.Sprintf("SUPERPLANE_TOKEN=%s", authToken),
		)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("canvas get failed: %s", string(output))
		}
	}

	contentType := "text/yaml"
	if format == "json" {
		contentType = "application/json"
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type":     "text",
				"text":     string(output),
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

// handleCanvasUpdate updates a canvas draft version by calling the internal API.
// It uses the openapi_client to properly serialize the canvas YAML to protobuf format.
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

	// Parse UUIDs to validate
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas_id: %w", err)
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}

	// Verify canvas exists and get creator for JWT
	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	// Write YAML to temp file (the CLI parser expects a file)
	tmpFile, err := os.CreateTemp("", "mcp-canvas-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Ensure YAML has proper apiVersion/kind/metadata structure
	if !strings.Contains(yamlContent, "apiVersion:") {
		// Parse the raw YAML and re-wrap it properly
		var rawData map[string]interface{}
		if err := yaml.Unmarshal([]byte(yamlContent), &rawData); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}

		name := canvas.Name
		if n, ok := rawData["name"].(string); ok && n != "" {
			name = n
		}

		wrapped := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Canvas",
			"metadata": map[string]interface{}{
				"id":   canvasID,
				"name": name,
			},
			"spec": map[string]interface{}{
				"nodes": rawData["nodes"],
				"edges": rawData["edges"],
			},
		}

		yamlBytes, err := yaml.Marshal(wrapped)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap YAML: %w", err)
		}
		yamlContent = string(yamlBytes)
	}

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Mint a JWT for the API call
	authToken := ""
	if bt, ok := ctx.Value("bearer_token").(string); ok {
		authToken = bt
	}
	if signer, ok := ctx.Value("jwt_signer").(*jwt.Signer); ok && signer != nil {
		if _, verr := signer.ValidateAndGetClaims(authToken); verr != nil {
			userID := ""
			if canvas.CreatedBy != nil {
				userID = canvas.CreatedBy.String()
			}
			claims := jwt.ScopedTokenClaims{
				Subject: userID,
				OrgID:   orgID,
				Purpose: "mcp-canvas-update",
				Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
					{ResourceType: "org", Action: "read"},
					{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
					{ResourceType: "canvases", Action: "update_version", Resources: []string{canvasID}},
				}),
			}
			if minted, merr := signer.GenerateScopedToken(claims, 5*time.Minute); merr == nil {
				authToken = minted
			}
		}
	}

	// Call superplane CLI to do the update (it handles all the YAML→protobuf conversion)
	// Use the CLI binary (not the server binary)
	cliBin := "/tmp/sp-cli"
	if _, err := os.Stat(cliBin); os.IsNotExist(err) {
		cliBin = "superplane" // fallback to PATH
	}
	cmd := exec.CommandContext(ctx, cliBin, "canvases", "update", "-f", tmpFile.Name(), "--draft")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SUPERPLANE_URL=http://localhost:8000"),
		fmt.Sprintf("SUPERPLANE_TOKEN=%s", authToken),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("canvas update failed: %s", string(output))
	}

	result := map[string]interface{}{
		"success": true,
		"canvas_id": canvasID,
		"output": string(output),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
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
