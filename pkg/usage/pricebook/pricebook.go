package pricebook

import (
	"strings"
	"sync"
)

// FallbackVersion is stored on events when the process has not loaded a
// database price book yet.
const FallbackVersion = "2026-08-19.2"

const (
	tokensPerMillion = 1_000_000
	microsPerCent    = 10_000
	localFleetID     = "local"
)

// Catalog VM rates from https://superplane.com/pricing/.
// Tiny per-minute prices do not divide into integer micros per second.
// 3 micros/s bills ~10% under $0.0002/min; 2 micros/s bills ~20% over $0.0001/min.
const (
	MicrosPerSecondE1TinyAMD64  int64 = 3
	MicrosPerSecondE1TinyARM64  int64 = 2
	MicrosPerSecondE1LargeAMD64 int64 = 70
	MicrosPerSecondE1LargeARM64 int64 = 50
	MicrosPerSecondE1Tiny       int64 = MicrosPerSecondE1TinyAMD64
	MicrosPerSecondE1Large      int64 = MicrosPerSecondE1LargeAMD64
)

// Rate is USD cents per million tokens for one token class.
type Rate struct {
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
	Reasoning  int64
}

type PrefixRate struct {
	Provider string
	Prefix   string
	Rate     Rate
}

type FamilyRate struct {
	Provider string
	Token    string
	Rate     Rate
}

// ExactRate prices one provider catalog model id.
type ExactRate struct {
	Provider string
	ModelID  string
	Rate     Rate
}

// Book is one versioned catalog of model and compute rates.
type Book struct {
	Version      string
	ExactRates   []ExactRate
	PrefixRates  []PrefixRate
	FamilyRates  []FamilyRate
	ComputeRates map[string]int64
}

type exactKey struct {
	provider string
	model    string
}

type entry struct {
	provider string
	prefix   string
	rate     Rate
}

type familyEntry struct {
	provider string
	token    string
	rate     Rate
}

var (
	rateClaudeOpus   = Rate{Input: 1500, Output: 7500, CacheRead: 150, CacheWrite: 1875}
	rateClaudeSonnet = Rate{Input: 300, Output: 1500, CacheRead: 30, CacheWrite: 375}
	rateClaudeHaiku  = Rate{Input: 80, Output: 400, CacheRead: 8, CacheWrite: 100}
)

var mu sync.RWMutex
var current = defaultBook()

// Version is stored on every usage event priced by this book.
var Version = current.version

func defaultPrefixRates() []entry {
	anthropicOpenRouter := []string{"anthropic", "openrouter"}
	openaiOpenRouter := []string{"openai", "openrouter"}
	openrouterOnly := []string{"openrouter"}
	var rates []entry
	rates = append(rates, prefixRatesFor(anthropicOpenRouter, "claude-opus", rateClaudeOpus)...)
	rates = append(rates, prefixRatesFor(anthropicOpenRouter, "claude-sonnet", rateClaudeSonnet)...)
	rates = append(rates, prefixRatesFor(anthropicOpenRouter, "claude-haiku", rateClaudeHaiku)...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "gpt-4o-mini", openAIRate(15, 60))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "gpt-4o", openAIRate(250, 1000))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "gpt-5-mini", openAIRate(25, 200))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "gpt-5", openAIRate(125, 1000))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "o3-mini", openAIRate(110, 440))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "o3", openAIRate(2000, 8000))...)
	rates = append(rates, prefixRatesFor(openaiOpenRouter, "o4-mini", openAIRate(110, 440))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-2.5-pro", openAIRate(125, 1000))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-2.5-flash", openAIRate(15, 60))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-2.0-flash", openAIRate(10, 40))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-1.5-pro", openAIRate(125, 500))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-1.5-flash", openAIRate(8, 30))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-flash", openAIRate(15, 60))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini-pro", openAIRate(125, 1000))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "gemini", openAIRate(15, 60))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "grok-3-mini", openAIRate(30, 50))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "grok-3", openAIRate(300, 1500))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "grok-2", openAIRate(200, 1000))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "grok", openAIRate(300, 1500))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "deepseek-reasoner", openAIRate(55, 219))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "deepseek-r1", openAIRate(55, 219))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "deepseek-chat", openAIRate(27, 110))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "deepseek-v3", openAIRate(27, 110))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "deepseek", openAIRate(27, 110))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "qwen-max", openAIRate(160, 640))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "qwen-plus", openAIRate(40, 120))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "qwen-turbo", openAIRate(5, 20))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "qwen3", openAIRate(30, 90))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "qwen", openAIRate(40, 120))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "kimi-k2", openAIRate(60, 250))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "moonshot", openAIRate(120, 120))...)
	rates = append(rates, prefixRatesFor(openrouterOnly, "kimi", openAIRate(60, 250))...)
	return rates
}

func defaultFamilyRates() []familyEntry {
	return familyRatesFor([]string{"anthropic", "openrouter"}, []struct {
		token string
		rate  Rate
	}{
		{token: "opus", rate: rateClaudeOpus},
		{token: "sonnet", rate: rateClaudeSonnet},
		{token: "haiku", rate: rateClaudeHaiku},
	}...)
}

func prefixRatesFor(providers []string, prefix string, rate Rate) []entry {
	rates := make([]entry, 0, len(providers))
	for _, provider := range providers {
		rates = append(rates, entry{provider: provider, prefix: prefix, rate: rate})
	}
	return rates
}

func familyRatesFor(providers []string, families ...struct {
	token string
	rate  Rate
}) []familyEntry {
	rates := make([]familyEntry, 0, len(providers)*len(families))
	for _, provider := range providers {
		for _, family := range families {
			rates = append(rates, familyEntry{provider: provider, token: family.token, rate: family.rate})
		}
	}
	return rates
}

func defaultComputeRates() map[string]int64 {
	return map[string]int64{
		"e1-large-amd64": MicrosPerSecondE1LargeAMD64,
		"e1-large-arm64": MicrosPerSecondE1LargeARM64,
		"e1-tiny-amd64":  MicrosPerSecondE1TinyAMD64,
		"e1-tiny-arm64":  MicrosPerSecondE1TinyARM64,
		"local":          0,
	}
}

type bookState struct {
	version      string
	exact        map[exactKey]Rate
	rates        []entry
	familyRates  []familyEntry
	computeRates map[string]int64
}

func defaultBook() bookState {
	return bookState{
		version:      FallbackVersion,
		exact:        map[exactKey]Rate{},
		rates:        defaultPrefixRates(),
		familyRates:  defaultFamilyRates(),
		computeRates: defaultComputeRates(),
	}
}

// Replace installs a database-backed book as the in-memory catalog.
func Replace(book Book) {
	next := bookState{
		version:      strings.TrimSpace(book.Version),
		exact:        map[exactKey]Rate{},
		computeRates: map[string]int64{},
	}
	if next.version == "" {
		next.version = FallbackVersion
	}
	for _, item := range book.ExactRates {
		provider := strings.ToLower(strings.TrimSpace(item.Provider))
		model := strings.ToLower(strings.TrimSpace(CatalogModelID(item.ModelID)))
		if provider == "" || model == "" {
			continue
		}
		next.exact[exactKey{provider: provider, model: model}] = item.Rate
	}
	for _, item := range book.PrefixRates {
		provider := strings.ToLower(strings.TrimSpace(item.Provider))
		prefix := strings.ToLower(strings.TrimSpace(item.Prefix))
		if provider == "" || prefix == "" {
			continue
		}
		next.rates = append(next.rates, entry{provider: provider, prefix: prefix, rate: item.Rate})
	}
	for _, item := range book.FamilyRates {
		provider := strings.ToLower(strings.TrimSpace(item.Provider))
		token := strings.ToLower(strings.TrimSpace(item.Token))
		if provider == "" || token == "" {
			continue
		}
		next.familyRates = append(next.familyRates, familyEntry{provider: provider, token: token, rate: item.Rate})
	}
	for key, rate := range book.ComputeRates {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		next.computeRates[normalized] = rate
	}
	if len(next.exact) == 0 && len(next.rates) == 0 {
		next.rates = defaultPrefixRates()
	}
	if len(next.exact) == 0 && len(next.familyRates) == 0 {
		next.familyRates = defaultFamilyRates()
	}

	mu.Lock()
	current = next
	Version = next.version
	mu.Unlock()
}

// Reset restores the compiled-in catalog. Tests use this after Replace or SetComputeRates.
func Reset() {
	mu.Lock()
	current = defaultBook()
	Version = current.version
	mu.Unlock()
}

// EstimateMicros prices a call in millionths of a US dollar.
// Unknown models return 0 so token counts still record.
func EstimateMicros(provider, model string, input, output, cacheRead, cacheWrite, reasoning int64) int64 {
	rate, ok := lookup(provider, model)
	if !ok {
		return 0
	}
	return micros(input, rate.Input) +
		micros(output, rate.Output) +
		micros(cacheRead, rate.CacheRead) +
		micros(cacheWrite, rate.CacheWrite) +
		micros(reasoning, rate.Reasoning)
}

// EstimateComputeMicros prices runner-fleet seconds. Local and empty fleets are 0.
func EstimateComputeMicros(machineType, fleetID string, seconds int64) int64 {
	if seconds <= 0 {
		return 0
	}
	if isUnbilledFleet(fleetID) {
		return 0
	}
	rate := computeRate(machineType)
	if rate <= 0 {
		return 0
	}
	return seconds * rate
}

func isUnbilledFleet(fleetID string) bool {
	id := strings.TrimSpace(fleetID)
	return id == "" || strings.EqualFold(id, localFleetID)
}

func computeRate(machineType string) int64 {
	key := strings.ToLower(strings.TrimSpace(machineType))
	mu.RLock()
	defer mu.RUnlock()
	if rate, ok := current.computeRates[key]; ok {
		return rate
	}
	return 0
}

// SetComputeRates replaces in-memory VM rates. Tests and the DB loader use this.
func SetComputeRates(rates map[string]int64) {
	next := make(map[string]int64, len(rates))
	for key, rate := range rates {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		next[normalized] = rate
	}
	mu.Lock()
	current.computeRates = next
	mu.Unlock()
}

// Lookup returns the rate for a provider and model id.
func Lookup(provider, model string) (Rate, bool) {
	return lookup(provider, model)
}

// IsPriced is true when the price book has a rate for the provider and model id.
func IsPriced(provider, model string) bool {
	_, ok := lookup(provider, model)
	return ok
}

// MicrosToCents converts millionths of a dollar to whole cents.
func MicrosToCents(micros int64) int64 {
	if micros < 0 {
		return 0
	}
	return micros / microsPerCent
}

func lookup(provider, model string) (Rate, bool) {
	if rate, ok := lookupExact(provider, model); ok {
		return rate, true
	}
	normalized := normalizeModelID(model)
	if rate, ok := lookupPrefix(provider, normalized); ok {
		return rate, true
	}
	if rate, ok := lookupFamilyToken(provider, normalized); ok {
		return rate, true
	}
	return lookupCompiledIn(provider, normalized)
}

func lookupCompiledIn(provider, normalized string) (Rate, bool) {
	if rate, ok := matchPrefixRate(defaultPrefixRates(), provider, normalized); ok {
		return rate, true
	}
	return matchFamilyRate(defaultFamilyRates(), provider, normalized)
}

func lookupExact(provider, model string) (Rate, bool) {
	key := exactKey{
		provider: strings.ToLower(strings.TrimSpace(provider)),
		model:    strings.ToLower(strings.TrimSpace(CatalogModelID(model))),
	}
	if key.provider == "" || key.model == "" {
		return Rate{}, false
	}
	mu.RLock()
	defer mu.RUnlock()
	rate, ok := current.exact[key]
	return rate, ok
}

func normalizeModelID(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(CatalogModelID(model)))
	provider, rest, found := strings.Cut(normalized, "/")
	if found && provider != "" && rest != "" && !strings.Contains(rest, "/") {
		return rest
	}
	return normalized
}

func lookupPrefix(provider, normalized string) (Rate, bool) {
	mu.RLock()
	defer mu.RUnlock()
	return matchPrefixRate(current.rates, provider, normalized)
}

func lookupFamilyToken(provider, normalized string) (Rate, bool) {
	mu.RLock()
	defer mu.RUnlock()
	return matchFamilyRate(current.familyRates, provider, normalized)
}

func matchPrefixRate(rates []entry, provider, normalized string) (Rate, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return Rate{}, false
	}
	bestPrefix := ""
	var best Rate
	found := false
	for _, item := range rates {
		if item.provider != provider || !strings.HasPrefix(normalized, item.prefix) {
			continue
		}
		if !found || len(item.prefix) > len(bestPrefix) {
			bestPrefix = item.prefix
			best = item.rate
			found = true
		}
	}
	if found {
		return best, true
	}
	return Rate{}, false
}

func matchFamilyRate(rates []familyEntry, provider, normalized string) (Rate, bool) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return Rate{}, false
	}
	parts := strings.Split(normalized, "-")
	for _, family := range rates {
		if family.provider != provider {
			continue
		}
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

func openAIRate(input, output int64) Rate {
	return Rate{
		Input:     input,
		Output:    output,
		CacheRead: input / 10,
	}
}
