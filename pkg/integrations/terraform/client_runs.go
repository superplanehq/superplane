package terraform

import (
	"context"
	"encoding/json"
	"fmt"
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
}

type WorkspacePayload struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		AutoApply bool   `json:"auto-apply"`
	} `json:"attributes"`
}

type PolicyChecksPayload struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Result struct {
				Result bool `json:"result"`
			} `json:"result"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) ReadRun(ctx context.Context, runID string) (*RunPayload, error) {
	path := fmt.Sprintf("/api/v2/runs/%s?include=workspace", runID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create run read request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read run: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to read run: bad status %d", resp.StatusCode)
	}

	var payload struct {
		Data     RunPayload         `json:"data"`
		Included []WorkspacePayload `json:"included"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode run: %w", err)
	}

	for _, inc := range payload.Included {
		if inc.Attributes.Name != "" {
			wk := inc
			payload.Data.Workspace = &wk
			break
		}
	}

	return &payload.Data, nil
}

func (c *Client) CreateRun(ctx context.Context, workspaceID, message string) (*RunPayload, error) {
	opts := map[string]any{
		"data": map[string]any{
			"type": "runs",
			"attributes": map[string]any{
				"message": message,
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
	req, err := c.newRequest(ctx, http.MethodPost, "/api/v2/runs", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create run create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	defer resp.Body.Close()

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

func (c *Client) ApplyRun(ctx context.Context, runID, comment string) error {
	opts := map[string]any{}
	if comment != "" {
		opts = map[string]any{"comment": comment}
	}
	path := fmt.Sprintf("/api/v2/runs/%s/actions/apply", runID)
	req, err := c.newRequest(ctx, http.MethodPost, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create apply request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to apply run: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to apply run: bad status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DiscardRun(ctx context.Context, runID, comment string) error {
	opts := map[string]any{}
	if comment != "" {
		opts = map[string]any{"comment": comment}
	}
	path := fmt.Sprintf("/api/v2/runs/%s/actions/discard", runID)
	req, err := c.newRequest(ctx, http.MethodPost, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create discard request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to discard run: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to discard run: bad status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) OverridePolicy(ctx context.Context, policyCheckID string) error {
	path := fmt.Sprintf("/api/v2/policy-checks/%s/actions/override", policyCheckID)
	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create override request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to override policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to override policy: bad status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ListPolicyChecks(ctx context.Context, runID string) (*PolicyChecksPayload, error) {
	path := fmt.Sprintf("/api/v2/runs/%s/policy-checks", runID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list policy checks request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list policy checks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to list policy checks: bad status %d", resp.StatusCode)
	}

	var payload PolicyChecksPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode policy checks: %w", err)
	}

	return &payload, nil
}
