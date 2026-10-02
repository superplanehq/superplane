package customllm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const anthropicVersion = "2023-06-01"

type Client struct {
	APIKey  string
	BaseURL string
	APIType string
	http    core.HTTPContext
}

type Model struct {
	ID   string
	Name string
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}
	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}
	baseURL, err := ctx.GetConfig("baseURL")
	if err != nil {
		return nil, err
	}
	apiType, err := ctx.GetConfig("apiType")
	if err != nil {
		return nil, err
	}
	normalizedType, err := NormalizeAPIType(string(apiType))
	if err != nil {
		return nil, err
	}
	trimmedURL := strings.TrimRight(strings.TrimSpace(string(baseURL)), "/")
	if trimmedURL == "" {
		return nil, fmt.Errorf("baseURL is required")
	}
	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	return &Client{
		APIKey:  key,
		BaseURL: trimmedURL,
		APIType: normalizedType,
		http:    httpClient,
	}, nil
}

func (c *Client) ListModels() ([]Model, error) {
	body, err := c.get(c.BaseURL + "/models")
	if err != nil {
		return nil, err
	}
	return decodeModels(body)
}

func (c *Client) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.APIType == APITypeAnthropic {
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("anthropic-version", anthropicVersion)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	res, err := c.http.Do(req)
	if err != nil {
		message := fmt.Sprintf("request failed: %v", err)
		return nil, core.NewProviderTransportError(message, errors.New(message))
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("request got %d code: %s", res.StatusCode, string(responseBody))
		return nil, core.NewProviderAPIError(res.StatusCode, message, errors.New(message))
	}
	return responseBody, nil
}

func decodeModels(body []byte) ([]Model, error) {
	var envelope struct {
		Data   []rawModel `json:"data"`
		Models []rawModel `json:"models"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Data != nil {
			return namedModels(envelope.Data), nil
		}
		if envelope.Models != nil {
			return namedModels(envelope.Models), nil
		}
	}

	var list []rawModel
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
		return namedModels(list), nil
	}
	return nil, fmt.Errorf("model list response did not include models")
}

type rawModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func namedModels(raw []rawModel) []Model {
	out := make([]Model, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = strings.TrimSpace(item.Name)
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(item.DisplayName)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name == "" {
			name = id
		}
		out = append(out, Model{ID: id, Name: name})
	}
	return out
}
