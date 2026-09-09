package pricesync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

const defaultOpenRouterModelsURL = "https://openrouter.ai/api/v1/models"

type openRouterModelsResponse struct {
	Data []openRouterModel `json:"data"`
}

type openRouterModel struct {
	ID      string            `json:"id"`
	Pricing openRouterPricing `json:"pricing"`
}

type openRouterPricing struct {
	Prompt            string `json:"prompt"`
	Completion        string `json:"completion"`
	InputCacheRead    string `json:"input_cache_read"`
	InputCacheWrite   string `json:"input_cache_write"`
	InternalReasoning string `json:"internal_reasoning"`
}

// OpenRouterSource loads exact model rates from the OpenRouter catalog.
type OpenRouterSource struct {
	HTTP core.HTTPContext
	URL  string
}

func (s OpenRouterSource) Name() string { return "openrouter" }

func (s OpenRouterSource) ScanModels(ctx context.Context) ([]ModelRate, error) {
	if s.HTTP == nil {
		return nil, fmt.Errorf("openrouter scan requires an HTTP client")
	}

	url := strings.TrimSpace(s.URL)
	if url == "" {
		url = defaultOpenRouterModelsURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build openrouter models request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("HTTP-Referer", "https://superplane.com")
	req.Header.Set("X-Title", "SuperPlane")

	res, err := s.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch openrouter models: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		msg, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("fetch openrouter models failed (%d): %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read openrouter models: %w", err)
	}

	rates, err := parseOpenRouterModels(body)
	if err != nil {
		return nil, err
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("openrouter catalog returned no model rates")
	}
	return rates, nil
}

func parseOpenRouterModels(body []byte) ([]ModelRate, error) {
	var response openRouterModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode openrouter models: %w", err)
	}

	rates := make([]ModelRate, 0, len(response.Data))
	seen := map[string]struct{}{}
	for _, item := range response.Data {
		id := strings.ToLower(strings.TrimSpace(item.ID))
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		rates = append(rates, ModelRate{
			MatchKey:  id,
			MatchMode: models.UsagePriceBookMatchExact,
			Rate: pricebook.Rate{
				Input:      parseUSDPerToken(item.Pricing.Prompt),
				Output:     parseUSDPerToken(item.Pricing.Completion),
				CacheRead:  parseUSDPerToken(item.Pricing.InputCacheRead),
				CacheWrite: parseUSDPerToken(item.Pricing.InputCacheWrite),
				Reasoning:  parseUSDPerToken(item.Pricing.InternalReasoning),
			},
		})
	}
	return rates, nil
}
