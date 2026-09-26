package models

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

// CatalogModelPrice is one provider model id with cents-per-million rates.
type CatalogModelPrice struct {
	Provider string
	ModelID  string
	Rate     pricebook.Rate
}

// ApplyCatalogPrices updates or adds exact rows for catalog prices.
// It copies other providers and compute rates unchanged.
func ApplyCatalogPrices(rows []UsagePriceBookRate, prices []CatalogModelPrice) ([]UsagePriceBookRate, int, int) {
	next := CloneUsagePriceBookRates(rows)
	updated := 0
	added := 0

	for _, price := range prices {
		provider := strings.ToLower(strings.TrimSpace(price.Provider))
		key := strings.ToLower(strings.TrimSpace(pricebook.CatalogModelID(price.ModelID)))
		if provider == "" || key == "" {
			continue
		}

		index := findModelRateIndex(next, provider, key, UsagePriceBookMatchExact)
		if index >= 0 {
			applyModelRate(&next[index], price.Rate)
			updated++
			continue
		}

		row := UsagePriceBookRate{
			UsageKind: UsageKindModel,
			Provider:  provider,
			MatchKey:  key,
			MatchMode: UsagePriceBookMatchExact,
		}
		applyModelRate(&row, price.Rate)
		next = append(next, NormalizeUsagePriceBookRate(row))
		added++
	}

	return next, updated, added
}

func findModelRateIndex(rows []UsagePriceBookRate, provider, matchKey, matchMode string) int {
	for i, row := range rows {
		if row.UsageKind == UsageKindModel &&
			row.Provider == provider &&
			row.MatchKey == matchKey &&
			row.MatchMode == matchMode {
			return i
		}
	}
	return -1
}

func applyModelRate(row *UsagePriceBookRate, rate pricebook.Rate) {
	row.InputCentsPerMillion = rate.Input
	row.OutputCentsPerMillion = rate.Output
	row.CacheReadCentsPerMillion = rate.CacheRead
	row.CacheWriteCentsPerMillion = rate.CacheWrite
	row.ReasoningCentsPerMillion = rate.Reasoning
}
