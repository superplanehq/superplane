package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// CFDTunnel is a Cloudflare Tunnel (cloudflared) returned by the account-scoped cfd_tunnel API.
type CFDTunnel struct {
	ID         string `json:"id,omitempty"`
	AccountTag string `json:"account_tag,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status,omitempty"`
	ConfigSrc  string `json:"config_src,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	DeletedAt  string `json:"deleted_at,omitempty"`
	// Metadata is an object in list/get responses; use RawMessage so unmarshaling accepts any JSON shape.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Token is returned only when creating a tunnel; treat as sensitive.
	Token string `json:"token,omitempty"`
}

// CreateCFDTunnelRequest is the payload for POST /accounts/{account_id}/cfd_tunnel
type CreateCFDTunnelRequest struct {
	Name      string `json:"name"`
	ConfigSrc string `json:"config_src,omitempty"`
}

// ListCFDTunnels lists Cloudflare Tunnels for an account (non-deleted by default).
func (c *Client) ListCFDTunnels(accountID string) ([]CFDTunnel, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/cfd_tunnel", c.BaseURL, accountID)
	responseBody, err := c.execRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Result  []CFDTunnel `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return response.Result, nil
}

// GetCFDTunnel retrieves a tunnel by ID.
func (c *Client) GetCFDTunnel(accountID, tunnelID string) (*CFDTunnel, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s", c.BaseURL, accountID, tunnelID)
	responseBody, err := c.execRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  CFDTunnel `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return &response.Result, nil
}

// CreateCFDTunnel creates a new Cloudflare Tunnel.
func (c *Client) CreateCFDTunnel(accountID string, req CreateCFDTunnelRequest) (*CFDTunnel, error) {
	rawURL := fmt.Sprintf("%s/accounts/%s/cfd_tunnel", c.BaseURL, accountID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool      `json:"success"`
		Result  CFDTunnel `json:"result"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return nil, newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return &response.Result, nil
}

// DeleteCFDTunnel permanently deletes a tunnel.
func (c *Client) DeleteCFDTunnel(accountID, tunnelID string) error {
	rawURL := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s", c.BaseURL, accountID, tunnelID)
	responseBody, err := c.execRequest(http.MethodDelete, rawURL, nil)
	if err != nil {
		return err
	}

	var response struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !response.Success {
		return newCloudflareAPIError(http.StatusOK, responseBody)
	}

	return nil
}
