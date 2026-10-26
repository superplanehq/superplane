package pricebooksync

import (
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/llm"
)

func catalogRateFromUSD(pricing llm.RawCatalogRate) (pricebook.Rate, bool) {
	input, ok := pricebook.CentsPerMillionFromUSDPerToken(pricing.Prompt)
	if !ok {
		return pricebook.Rate{}, false
	}
	output, ok := pricebook.CentsPerMillionFromUSDPerToken(pricing.Completion)
	if !ok {
		return pricebook.Rate{}, false
	}
	cacheRead, ok := pricebook.CentsPerMillionFromUSDPerToken(pricing.InputCacheRead)
	if !ok {
		return pricebook.Rate{}, false
	}
	cacheWrite, ok := pricebook.CentsPerMillionFromUSDPerToken(pricing.InputCacheWrite)
	if !ok {
		return pricebook.Rate{}, false
	}
	reasoning, ok := pricebook.CentsPerMillionFromUSDPerToken(pricing.InternalReasoning)
	if !ok {
		return pricebook.Rate{}, false
	}
	return pricebook.Rate{
		Input:      input,
		Output:     output,
		CacheRead:  cacheRead,
		CacheWrite: cacheWrite,
		Reasoning:  reasoning,
	}, true
}
