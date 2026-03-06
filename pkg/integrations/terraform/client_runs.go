package terraform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RunPayload struct {
	ID         string `json:"id"`
	Attributes struct {
		Status    string `json:"status"`
		Message   string `json:"message"`
		CreatedAt string `json:"created-at"`
	} `json:"attributes"`
	Workspace *WorkspacePayload
	Plan      *PlanReference
}

type PlanReference struct {
	ID string `json:"id"`
}

type PlanPayload struct {
	ID         string `json:"id"`
	Attributes struct {
		ResourceAdditions    int    `json:"resource-additions"`
		ResourceChanges      int    `json:"resource-changes"`
		ResourceDestructions int    `json:"resource-destructions"`
		LogReadURL           string `json:"log-read-url"`
	} `json:"attributes"`
	Links struct {
		JSONOutput string `json:"json-output"`
	} `json:"links"`
}

type OrganizationPayload struct {
	ID         string `json:"id"`
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

type WorkspacePayload struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		AutoApply bool   `json:"auto-apply"`
	} `json:"attributes"`
	Relationships struct {
		Organization struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"organization"`
	} `json:"relationships"`
	Organization *OrganizationPayload
}

func (c *Client) ReadRun(runID string) (*RunPayload, error) {
	path := fmt.Sprintf("/api/v2/runs/%s?include=workspace,workspace.organization", runID)
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create run read request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to read run: bad status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read run response: %w", err)
	}

	var payload struct {
		Data     RunPayload        `json:"data"`
		Included []json.RawMessage `json:"included"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode run: %w", err)
	}

	var runDetails struct {
		Data struct {
			Relationships struct {
				Plan struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"plan"`
			} `json:"relationships"`
		} `json:"data"`
	}

	_ = json.Unmarshal(bodyBytes, &runDetails)
	if runDetails.Data.Relationships.Plan.Data.ID != "" {
		payload.Data.Plan = &PlanReference{ID: runDetails.Data.Relationships.Plan.Data.ID}
	}

	// Parse included resources (workspaces and organizations)
	var workspace *WorkspacePayload
	orgs := make(map[string]*OrganizationPayload)

	for _, raw := range payload.Included {
		var typeCheck struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeCheck); err != nil {
			continue
		}

		switch typeCheck.Type {
		case "workspaces":
			var ws WorkspacePayload
			if err := json.Unmarshal(raw, &ws); err == nil && ws.Attributes.Name != "" {
				workspace = &ws
			}
		case "organizations":
			var org OrganizationPayload
			if err := json.Unmarshal(raw, &org); err == nil {
				orgs[org.ID] = &org
			}
		}
	}

	if workspace != nil {
		// Attach organization to workspace if available
		orgID := workspace.Relationships.Organization.Data.ID
		if org, ok := orgs[orgID]; ok {
			workspace.Organization = org
		}
		payload.Data.Workspace = workspace
	}

	return &payload.Data, nil
}

func (c *Client) CreateRun(workspaceID, message string, isPlanOnly bool) (*RunPayload, error) {
	opts := map[string]any{
		"data": map[string]any{
			"type": "runs",
			"attributes": map[string]any{
				"message":   message,
				"plan-only": isPlanOnly,
			},
			"relationships": map[string]any{
				"workspace": map[string]any{
					"data": map[string]any{
						"type": "workspaces",
						"id":   workspaceID,
					},
				},
			},
		},
	}
	req, err := c.newRequest(http.MethodPost, "/api/v2/runs", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create run create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to create run: bad status %d", resp.StatusCode)
	}

	var payload struct {
		Data RunPayload `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode created run: %w", err)
	}

	return &payload.Data, nil
}

func (c *Client) CancelRun(runID, comment string) error {
	opts := map[string]any{}
	if comment != "" {
		opts = map[string]any{"comment": comment}
	}
	path := fmt.Sprintf("/api/v2/runs/%s/actions/cancel", runID)
	req, err := c.newRequest(http.MethodPost, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to cancel run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to cancel run: bad status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ReadPlan(planID string) (*PlanPayload, error) {
	path := fmt.Sprintf("/api/v2/plans/%s", planID)
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan read request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to read plan: bad status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan body: %w", err)
	}
	var payload struct {
		Data PlanPayload `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode plan: %w", err)
	}

	return &payload.Data, nil
}

func (c *Client) DownloadLog(logURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, logURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create log download request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download log: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to download log: bad status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read log body: %w", err)
	}

	return string(bodyBytes), nil
}

func (c *Client) DownloadJSONOutput(apiPath string) (string, error) {
	req, err := c.newRequest(http.MethodGet, apiPath, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create json output request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download json output: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to download json output: bad status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read json output body: %w", err)
	}

	return string(bodyBytes), nil
}
