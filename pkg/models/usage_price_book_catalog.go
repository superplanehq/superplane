package models

import (
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

// CatalogModelPrice is one provider model id with cents-per-million rates.
type CatalogModelPrice struct {
	ModelID string
	Rate    pricebook.Rate
}

// ApplyCatalogPrices updates existing prefix/family rows from catalog prices
// and adds a prefix row only when no current key covers the model.
func ApplyCatalogPrices(rows []UsagePriceBookRate, prices []CatalogModelPrice) ([]UsagePriceBookRate, int, int) {
	next := CloneUsagePriceBookRates(rows)
	prefixes, families := catalogMatchLists(next)

	type assignment struct {
		index    int
		modelLen int
	}
	assigned := map[string]assignment{}
	updated := 0
	added := 0

	sorted := slices.Clone(prices)
	slices.SortFunc(sorted, func(a, b CatalogModelPrice) int {
		return len(strings.TrimSpace(b.ModelID)) - len(strings.TrimSpace(a.ModelID))
	})

	for _, price := range sorted {
		if strings.TrimSpace(price.ModelID) == "" {
			continue
		}

		match, ok := pricebook.MatchModel(price.ModelID, prefixes, families)
		if ok {
			id := match.Key + "\x00" + match.Mode
			if prev, exists := assigned[id]; exists && prev.modelLen >= len(price.ModelID) {
				continue
			}
			index := findModelRateIndex(next, match.Key, match.Mode)
			if index < 0 {
				continue
			}
			applyModelRate(&next[index], price.Rate)
			if _, exists := assigned[id]; !exists {
				updated++
			}
			assigned[id] = assignment{index: index, modelLen: len(price.ModelID)}
			continue
		}

		key := pricebook.NormalizeModelID(price.ModelID)
		if key == "" {
			continue
		}
		if findModelRateIndex(next, key, UsagePriceBookMatchPrefix) >= 0 {
			continue
		}
		row := UsagePriceBookRate{
			UsageKind: UsageKindModel,
			MatchKey:  key,
			MatchMode: UsagePriceBookMatchPrefix,
		}
		applyModelRate(&row, price.Rate)
		next = append(next, NormalizeUsagePriceBookRate(row))
		added++
	}

	return next, updated, added
}

func catalogMatchLists(rows []UsagePriceBookRate) ([]pricebook.PrefixRate, []pricebook.FamilyRate) {
	prefixes := make([]pricebook.PrefixRate, 0)
	families := make([]pricebook.FamilyRate, 0)
	for _, row := range rows {
		if row.UsageKind != UsageKindModel {
			continue
		}
		rate := pricebook.Rate{
			Input:      row.InputCentsPerMillion,
			Output:     row.OutputCentsPerMillion,
			CacheRead:  row.CacheReadCentsPerMillion,
			CacheWrite: row.CacheWriteCentsPerMillion,
			Reasoning:  row.ReasoningCentsPerMillion,
		}
		switch row.MatchMode {
		case UsagePriceBookMatchPrefix:
			prefixes = append(prefixes, pricebook.PrefixRate{Prefix: row.MatchKey, Rate: rate})
		case UsagePriceBookMatchFamily:
			families = append(families, pricebook.FamilyRate{Token: row.MatchKey, Rate: rate})
		}
	}
	return prefixes, families
}

func findModelRateIndex(rows []UsagePriceBookRate, matchKey, matchMode string) int {
	for i, row := range rows {
		if row.UsageKind == UsageKindModel && row.MatchKey == matchKey && row.MatchMode == matchMode {
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
