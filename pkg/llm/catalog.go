package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// ErrNoCatalogPrices means the provider list-models API has no rate fields.
var ErrNoCatalogPrices = errors.New("provider does not publish catalog prices")

// CatalogPrice is one model id and the raw prices returned by its provider.
type CatalogPrice struct {
	ID      string
	Pricing RawCatalogRate
}

// ListCatalogPrices fetches priced models for providers that publish catalog rates.
func ListCatalogPrices(ctx context.Context, httpClient core.HTTPContext, provider string, creds Credentials) ([]CatalogPrice, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if normalized != ProviderOpenRouter {
		return nil, fmt.Errorf("%w: %s", ErrNoCatalogPrices, provider)
	}

	client, err := New(httpClient, ProviderOpenRouter, creds)
	if err != nil {
		return nil, err
	}
	compat, ok := client.(*openAICompatClient)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoCatalogPrices, provider)
	}
	return compat.listCatalogPrices(ctx)
}

// RawCatalogRate contains provider prices in US dollars per token.
// Billing owns conversion into ledger rates.
type RawCatalogRate struct {
	Prompt            string `json:"prompt"`
	Completion        string `json:"completion"`
	InputCacheRead    string `json:"input_cache_read"`
	InputCacheWrite   string `json:"input_cache_write"`
	InternalReasoning string `json:"internal_reasoning"`
}
