package gitserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// APIClient calls the SuperPlane REST API to sync git changes to the canvas.
type APIClient struct {
	BaseURL string // e.g. "http://localhost:8000"
	Token   string // API token (Bearer)
}

// SlugToCanvasMapping maps a git repo slug to its canvas/org IDs.
type SlugToCanvasMapping struct {
	CanvasID string
	OrgID    string
}

// ResolveSlug looks up the canvas/org for a given slug.
// For POC: reads from the .superplane.yaml in the repo.
type SlugResolver func(slug string, repoPath string) (*SlugToCanvasMapping, error)

// APISyncHandler creates a SyncHandler that calls the REST API.
func APISyncHandler(baseURL string, tokenFunc func() string, resolver SlugResolver) *SyncHandler {
	return &SyncHandler{
		UpdateReadme: func(slug string, content string) error {
			mapping, err := resolver(slug, "")
			if err != nil {
				return fmt.Errorf("failed to resolve slug %s: %w", slug, err)
			}

			MarkSyncActive(mapping.CanvasID)
			defer MarkSyncDone(mapping.CanvasID)

			client := &APIClient{BaseURL: baseURL, Token: tokenFunc()}
			return client.UpdateReadme(mapping.CanvasID, content)
		},
		UpdateCanvas: func(slug string, yamlContent []byte) error {
			mapping, err := resolver(slug, "")
			if err != nil {
				return fmt.Errorf("failed to resolve slug %s: %w", slug, err)
			}

			MarkSyncActive(mapping.CanvasID)
			defer MarkSyncDone(mapping.CanvasID)

			client := &APIClient{BaseURL: baseURL, Token: tokenFunc()}
			return client.UpdateCanvasYAML(mapping.CanvasID, yamlContent)
		},
		UpdateApps: func(slug string, panels map[string]string, layout []byte) error {
			mapping, err := resolver(slug, "")
			if err != nil {
				return fmt.Errorf("failed to resolve slug %s: %w", slug, err)
			}

			MarkSyncActive(mapping.CanvasID)
			defer MarkSyncDone(mapping.CanvasID)

			client := &APIClient{BaseURL: baseURL, Token: tokenFunc()}
			return client.UpdateLaunchpad(mapping.CanvasID, panels, layout)
		},
	}
}

// UpdateReadme updates the canvas readme via PUT /api/v1/canvases/{id}/readme
func (c *APIClient) UpdateReadme(canvasID, content string) error {
	body := map[string]string{"content": content}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/api/v1/canvases/%s/readme", c.BaseURL, canvasID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("readme API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("readme API returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Auto-publish: get version ID from response and publish
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if versionID, ok := result["versionId"].(string); ok {
		return c.PublishVersion(canvasID, versionID)
	}

	log.Warnf("git-sync: readme updated but no versionId in response for canvas %s", canvasID)
	return nil
}

// UpdateCanvasYAML parses canvas YAML and syncs it to the canvas version API.
func (c *APIClient) UpdateCanvasYAML(canvasID string, yamlContent []byte) error {
	return c.SyncCanvasYAML(canvasID, yamlContent)
}

// UpdateLaunchpad updates the canvas launchpad/apps panels.
func (c *APIClient) UpdateLaunchpad(canvasID string, panels map[string]string, layout []byte) error {
	type Panel struct {
		ID      string            `json:"id"`
		Type    string            `json:"type"`
		Content map[string]string `json:"content"`
	}

	payload := struct {
		Panels []Panel         `json:"panels"`
		Layout json.RawMessage `json:"layout,omitempty"`
	}{
		Layout: layout,
	}

	for id, body := range panels {
		payload.Panels = append(payload.Panels, Panel{
			ID:      id,
			Type:    "markdown",
			Content: map[string]string{"body": body},
		})
	}

	jsonBody, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/v1/canvases/%s/launchpad", c.BaseURL, canvasID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("launchpad API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("launchpad API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PublishVersion publishes a draft version. Non-fatal if no changes.
func (c *APIClient) PublishVersion(canvasID, versionID string) error {
	url := fmt.Sprintf("%s/api/v1/canvases/%s/versions/%s/publish", c.BaseURL, canvasID, versionID)
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("publish API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		// "no changes" or "internal error" when content didn't actually change
		if resp.StatusCode == 500 {
			log.Infof("git-sync: publish skipped for %s/%s (likely no changes)", canvasID, versionID)
			return nil
		}
		return fmt.Errorf("publish API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
