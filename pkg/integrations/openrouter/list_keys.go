package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ListedKey struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

// ListKeys includes disabled keys so an uncertain create can always be found.
// The API paginates by offset: callers continue until the returned page is empty.
func (c *Client) ListKeys(ctx context.Context, offset int) ([]ListedKey, error) {
	if strings.TrimSpace(c.ManagementKey) == "" {
		return nil, fmt.Errorf("provisioning API key is not configured")
	}
	body, err := c.execRequestWithKey(ctx, http.MethodGet, baseURL+"/keys?include_disabled=true&offset="+strconv.Itoa(offset), nil, c.ManagementKey)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []ListedKey `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode key list: %w", err)
	}
	return response.Data, nil
}
