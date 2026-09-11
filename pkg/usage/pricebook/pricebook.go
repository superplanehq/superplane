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
	Prefix string
	Rate   Rate
}

type FamilyRate struct {
	Token string
	Rate  Rate
}

// Book is one versioned catalog of model and compute rates.
type Book struct {
	Version      string
	PrefixRates  []PrefixRate
	FamilyRates  []FamilyRate
	ComputeRates map[string]int64
}

type entry struct {
	prefix string
	rate   Rate
}

type familyEntry struct {
	token string
	rate  Rate
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
	return []entry{
		{prefix: "claude-opus", rate: rateClaudeOpus},
		{prefix: "claude-sonnet", rate: rateClaudeSonnet},
		{prefix: "claude-haiku", rate: rateClaudeHaiku},
		{prefix: "gpt-4o-mini", rate: openAIRate(15, 60)},
		{prefix: "gpt-4o", rate: openAIRate(250, 1000)},
		{prefix: "gpt-5-mini", rate: openAIRate(25, 200)},
		{prefix: "gpt-5", rate: openAIRate(125, 1000)},
		{prefix: "o3-mini", rate: openAIRate(110, 440)},
		{prefix: "o3", rate: openAIRate(2000, 8000)},
		{prefix: "o4-mini", rate: openAIRate(110, 440)},
		{prefix: "gemini-2.5-pro", rate: openAIRate(125, 1000)},
		{prefix: "gemini-2.5-flash", rate: openAIRate(15, 60)},
		{prefix: "gemini-2.0-flash", rate: openAIRate(10, 40)},
		{prefix: "gemini-1.5-pro", rate: openAIRate(125, 500)},
		{prefix: "gemini-1.5-flash", rate: openAIRate(8, 30)},
		{prefix: "gemini-flash", rate: openAIRate(15, 60)},
		{prefix: "gemini-pro", rate: openAIRate(125, 1000)},
		{prefix: "gemini", rate: openAIRate(15, 60)},
		{prefix: "grok-3-mini", rate: openAIRate(30, 50)},
		{prefix: "grok-3", rate: openAIRate(300, 1500)},
		{prefix: "grok-2", rate: openAIRate(200, 1000)},
		{prefix: "grok", rate: openAIRate(300, 1500)},
		{prefix: "deepseek-reasoner", rate: openAIRate(55, 219)},
		{prefix: "deepseek-r1", rate: openAIRate(55, 219)},
		{prefix: "deepseek-chat", rate: openAIRate(27, 110)},
		{prefix: "deepseek-v3", rate: openAIRate(27, 110)},
		{prefix: "deepseek", rate: openAIRate(27, 110)},
		{prefix: "qwen-max", rate: openAIRate(160, 640)},
		{prefix: "qwen-plus", rate: openAIRate(40, 120)},
		{prefix: "qwen-turbo", rate: openAIRate(5, 20)},
		{prefix: "qwen3", rate: openAIRate(30, 90)},
		{prefix: "qwen", rate: openAIRate(40, 120)},
		{prefix: "kimi-k2", rate: openAIRate(60, 250)},
		{prefix: "moonshot", rate: openAIRate(120, 120)},
		{prefix: "kimi", rate: openAIRate(60, 250)},
	}
}

func defaultFamilyRates() []familyEntry {
	return []familyEntry{
		{token: "opus", rate: rateClaudeOpus},
		{token: "sonnet", rate: rateClaudeSonnet},
		{token: "haiku", rate: rateClaudeHaiku},
	}
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
	rates        []entry
	familyRates  []familyEntry
	computeRates map[string]int64
}

func defaultBook() bookState {
	return bookState{
		version:      FallbackVersion,
		rates:        defaultPrefixRates(),
		familyRates:  defaultFamilyRates(),
		computeRates: defaultComputeRates(),
	}
}

// Replace installs a database-backed book as the in-memory catalog.
func Replace(book Book) {
	next := bookState{
		version:      strings.TrimSpace(book.Version),
		computeRates: map[string]int64{},
	}
	if next.version == "" {
		next.version = FallbackVersion
	}
	for _, item := range book.PrefixRates {
		prefix := strings.ToLower(strings.TrimSpace(item.Prefix))
		if prefix == "" {
			continue
		}
		next.rates = append(next.rates, entry{prefix: prefix, rate: item.Rate})
	}
	for _, item := range book.FamilyRates {
		token := strings.ToLower(strings.TrimSpace(item.Token))
		if token == "" {
			continue
		}
		next.familyRates = append(next.familyRates, familyEntry{token: token, rate: item.Rate})
	}
	for key, rate := range book.ComputeRates {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		next.computeRates[normalized] = rate
	}
	if len(next.rates) == 0 {
		next.rates = defaultPrefixRates()
	}
	if len(next.familyRates) == 0 {
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

// IsPriced is true when the price book has a rate for the model id.
func IsPriced(model string) bool {
	_, ok := lookup(model)
	return ok
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
	mu.RLock()
	defer mu.RUnlock()
	bestPrefix := ""
	var best Rate
	found := false
	for _, item := range current.rates {
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
	mu.RLock()
	defer mu.RUnlock()
	parts := strings.Split(normalized, "-")
	for _, family := range current.familyRates {
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
