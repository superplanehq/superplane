package pricebook

import "strings"

// Version is stored on every usage event priced by this book.
const Version = "2026-08-19"

const tokensPerMillion = 1_000_000
const microsPerCent = 10_000

// Rate is USD cents per million tokens for one token class.
type Rate struct {
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
	Reasoning  int64
}

type entry struct {
	prefix string
	rate   Rate
}

var (
	rateClaudeOpus   = Rate{Input: 1500, Output: 7500, CacheRead: 150, CacheWrite: 1875}
	rateClaudeSonnet = Rate{Input: 300, Output: 1500, CacheRead: 30, CacheWrite: 375}
	rateClaudeHaiku  = Rate{Input: 80, Output: 400, CacheRead: 8, CacheWrite: 100}
)

// Published approximate provider list prices. Longest prefix wins.
// List cheaper siblings (mini/nano) as longer prefixes than the flagship id.
var rates = []entry{
	{prefix: "claude-opus", rate: rateClaudeOpus},
	{prefix: "claude-sonnet", rate: rateClaudeSonnet},
	{prefix: "claude-haiku", rate: rateClaudeHaiku},
	{prefix: "gpt-4o-mini", rate: Rate{Input: 15, Output: 60}},
	{prefix: "gpt-4o", rate: Rate{Input: 250, Output: 1000}},
	{prefix: "gpt-5-mini", rate: Rate{Input: 25, Output: 200}},
	{prefix: "gpt-5", rate: Rate{Input: 125, Output: 1000}},
	{prefix: "o3-mini", rate: Rate{Input: 110, Output: 440}},
	{prefix: "o3", rate: Rate{Input: 2000, Output: 8000}},
	{prefix: "o4-mini", rate: Rate{Input: 110, Output: 440}},
}

var familyRates = []struct {
	token string
	rate  Rate
}{
	{token: "opus", rate: rateClaudeOpus},
	{token: "sonnet", rate: rateClaudeSonnet},
	{token: "haiku", rate: rateClaudeHaiku},
}

// EstimateMicros prices a call in millionths of a US dollar.
// Unknown models return 0 so token counts still record.
func EstimateMicros(provider, model string, input, output, cacheRead, cacheWrite, reasoning int64) int64 {
	rate, ok := lookup(model)
	if !ok {
		return 0
	}
	_ = provider
	return micros(input, rate.Input) +
		micros(output, rate.Output) +
		micros(cacheRead, rate.CacheRead) +
		micros(cacheWrite, rate.CacheWrite) +
		micros(reasoning, rate.Reasoning)
}

// MicrosToCents converts millionths of a dollar to whole cents.
func MicrosToCents(micros int64) int64 {
	if micros < 0 {
		return 0
	}
	return micros / microsPerCent
}

func lookup(model string) (Rate, bool) {
	normalized := normalizeModelID(model)
	if rate, ok := lookupPrefix(normalized); ok {
		return rate, true
	}
	return lookupFamilyToken(normalized)
}

func normalizeModelID(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	provider, rest, found := strings.Cut(normalized, "/")
	if found && provider != "" && rest != "" && !strings.Contains(rest, "/") {
		return rest
	}
	return normalized
}

func lookupPrefix(normalized string) (Rate, bool) {
	bestPrefix := ""
	var best Rate
	found := false
	for _, item := range rates {
		if !strings.HasPrefix(normalized, item.prefix) {
			continue
		}
		if !found || len(item.prefix) > len(bestPrefix) {
			bestPrefix = item.prefix
			best = item.rate
			found = true
		}
	}
	return best, found
}

func lookupFamilyToken(normalized string) (Rate, bool) {
	parts := strings.Split(normalized, "-")
	for _, family := range familyRates {
		for _, part := range parts {
			if part == family.token {
				return family.rate, true
			}
		}
	}
	return Rate{}, false
}

func micros(tokens, centsPerMillion int64) int64 {
	if tokens <= 0 || centsPerMillion <= 0 {
		return 0
	}
	return tokens * centsPerMillion * microsPerCent / tokensPerMillion
}
