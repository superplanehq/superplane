package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

// ErrNoCatalogPrices means the provider list-models API has no rate fields.
var ErrNoCatalogPrices = errors.New("provider does not publish catalog prices")

// CatalogPrice is one model id with cents-per-million rates from a provider catalog.
type CatalogPrice struct {
	ID   string
	Rate pricebook.Rate
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
