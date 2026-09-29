package models

import "slices"

// UsagePriceBookRatesEqual reports whether two rate slices carry the same
// match rules and cents values. It ignores ids and version stamps.
func UsagePriceBookRatesEqual(a, b []UsagePriceBookRate) bool {
	if len(a) != len(b) {
		return false
	}

	left := CloneUsagePriceBookRates(a)
	right := CloneUsagePriceBookRates(b)
	slices.SortFunc(left, usagePriceBookRateLess)
	slices.SortFunc(right, usagePriceBookRateLess)
	return slices.EqualFunc(left, right, usagePriceBookRateValuesEqual)
}

func usagePriceBookRateLess(a, b UsagePriceBookRate) int {
	if c := stringsCompare(a.UsageKind, b.UsageKind); c != 0 {
		return c
	}
	if c := stringsCompare(a.MatchKey, b.MatchKey); c != 0 {
		return c
	}
	return stringsCompare(a.MatchMode, b.MatchMode)
}

func stringsCompare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func usagePriceBookRateValuesEqual(a, b UsagePriceBookRate) bool {
	return a.UsageKind == b.UsageKind &&
		a.MatchKey == b.MatchKey &&
		a.MatchMode == b.MatchMode &&
		a.InputCentsPerMillion == b.InputCentsPerMillion &&
		a.OutputCentsPerMillion == b.OutputCentsPerMillion &&
		a.CacheReadCentsPerMillion == b.CacheReadCentsPerMillion &&
		a.CacheWriteCentsPerMillion == b.CacheWriteCentsPerMillion &&
		a.ReasoningCentsPerMillion == b.ReasoningCentsPerMillion &&
		a.MicrosPerSecond == b.MicrosPerSecond
}
