package gitserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
)

// CanvasSync handles parsing canvas YAML and syncing it to the API.
type CanvasSync struct {
	client *APIClient
}

// SyncCanvasYAML parses canvas YAML and pushes it through the version update API.
func (c *APIClient) SyncCanvasYAML(canvasID string, yamlContent []byte) error {
	// Parse the YAML using the same models as the CLI
	canvas, err := models.ParseCanvas(yamlContent)
	if err != nil {
		return fmt.Errorf("failed to parse canvas YAML: %w", err)
	}

	canvasObj := models.CanvasFromCanvas(*canvas)

	// First, get or create a draft version
	versionID, err := c.ensureDraftVersion(canvasID)
	if err != nil {
		return fmt.Errorf("failed to ensure draft version: %w", err)
	}

	// Build the update body matching CanvasesUpdateCanvasVersionBody
	body := map[string]interface{}{
		"canvas":    canvasObj,
		"versionId": versionID,
		"autoLayout": map[string]interface{}{
			"algorithm": "ALGORITHM_HORIZONTAL",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal canvas update body: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/canvases/%s/versions", c.BaseURL, canvasID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("canvas version update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("canvas version update returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Extract version ID from response and publish
	var result struct {
		Version struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
		} `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Version.Metadata.ID != "" {
		log.Infof("git-sync: canvas version %s updated, publishing...", result.Version.Metadata.ID)
		return c.PublishVersion(canvasID, result.Version.Metadata.ID)
	}

	log.Warnf("git-sync: canvas updated but could not extract version ID for publish")
	return nil
}

// ensureDraftVersion gets the current user's draft or creates one.
func (c *APIClient) ensureDraftVersion(canvasID string) (string, error) {
	// List versions and find a draft
	listURL := fmt.Sprintf("%s/api/v1/canvases/%s/versions", c.BaseURL, canvasID)
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Versions []struct {
			Metadata struct {
				ID    string `json:"id"`
				State string `json:"state"`
			} `json:"metadata"`
		} `json:"versions"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// Find existing draft (any non-published version)
	for _, v := range result.Versions {
		if v.Metadata.State != "STATE_PUBLISHED" && v.Metadata.ID != "" {
			return v.Metadata.ID, nil
		}
	}

	// Create a new draft: POST /api/v1/canvases/{id}/versions
	createURL := fmt.Sprintf("%s/api/v1/canvases/%s/versions", c.BaseURL, canvasID)
	createReq, err := http.NewRequest("POST", createURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+c.Token)

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return "", err
	}
	defer createResp.Body.Close()

	if createResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(createResp.Body)
		return "", fmt.Errorf("create draft returned %d: %s", createResp.StatusCode, string(respBody))
	}

	var createResult struct {
		Version struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
		} `json:"version"`
	}
	json.NewDecoder(createResp.Body).Decode(&createResult)

	if createResult.Version.Metadata.ID != "" {
		return createResult.Version.Metadata.ID, nil
	}

	return "", fmt.Errorf("failed to create or find draft version")
}
