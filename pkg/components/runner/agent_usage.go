package runner

import (
	"encoding/json"
	"math"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

func RecordRunnerLLMUsage(usage core.UsageRecorder, logger *log.Entry, finishedEventType string, configuration any, result json.RawMessage) {
	if usage == nil {
		return
	}
	provider, ok := providerForFinishedEvent(finishedEventType)
	if !ok {
		return
	}
	record, ok := ParseRunnerLLMUsage(provider, configuration, result)
	if !ok {
		return
	}
	record.IdempotencyKey = models.UsageIdempotencyKeyRunner
	if err := usage.Record(record); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record runner LLM usage")
	}
}

func ParseRunnerLLMUsage(provider string, configuration any, result json.RawMessage) (core.UsageRecord, bool) {
	parsed := parseRunnerResultUsage(result)
	if parsed.InputTokens == 0 && parsed.OutputTokens == 0 && parsed.CacheReadTokens == 0 && parsed.CacheWriteTokens == 0 && parsed.ReasoningTokens == 0 && parsed.CostMicros == nil {
		return core.UsageRecord{}, false
	}

	model := strings.TrimSpace(parsed.Model)
	if model == "" {
		model = configurationString(configuration, "model")
	}
	if model == "" {
		model = "unknown"
	}

	source := configurationCredentialsSource(configuration)
	total := parsed.InputTokens + parsed.OutputTokens + parsed.CacheReadTokens + parsed.CacheWriteTokens + parsed.ReasoningTokens
	return core.UsageRecord{
		Provider:         provider,
		Model:            model,
		InputTokens:      parsed.InputTokens,
		OutputTokens:     parsed.OutputTokens,
		CacheReadTokens:  parsed.CacheReadTokens,
		CacheWriteTokens: parsed.CacheWriteTokens,
		ReasoningTokens:  parsed.ReasoningTokens,
		TotalTokens:      total,
		CostMicros:       parsed.CostMicros,
		FundingSource:    FundingSourceForCredentials(source),
	}, true
}

func providerForFinishedEvent(finishedEventType string) (string, bool) {
	switch finishedEventType {
	case "runnerClaudeCode.finished":
		return models.UsageProviderAnthropic, true
	case "runnerCodex.finished":
		return models.UsageProviderOpenAI, true
	case "runnerOpenRouter.finished":
		return models.UsageProviderOpenRouter, true
	default:
		return "", false
	}
}

type parsedRunnerUsage struct {
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	CostMicros       *int64
}

func parseRunnerResultUsage(raw json.RawMessage) parsedRunnerUsage {
	if len(raw) == 0 {
		return parsedRunnerUsage{}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return parsedRunnerUsage{}
	}
	return parsedRunnerUsage{
		Model:            firstString(payload, "model", "model_id"),
		InputTokens:      firstInt(payload, "input_tokens", "prompt_tokens"),
		OutputTokens:     firstInt(payload, "output_tokens", "completion_tokens"),
		CacheReadTokens:  firstInt(payload, "cache_read_input_tokens", "cached_input_tokens", "cache_read_tokens"),
		CacheWriteTokens: firstInt(payload, "cache_creation_input_tokens", "cache_write_tokens"),
		ReasoningTokens:  firstInt(payload, "reasoning_tokens"),
		CostMicros:       costMicrosFromPayload(payload),
	}
}

func costMicrosFromPayload(payload map[string]any) *int64 {
	if usage, ok := payload["usage"].(map[string]any); ok {
		if micros := costMicrosFromPayload(usage); micros != nil {
			return micros
		}
	}
	if v, ok := asFloat(payload["total_cost_usd"]); ok && v > 0 {
		micros := int64(math.Round(v * 1_000_000))
		return &micros
	}
	if v, ok := asFloat(payload["cost_usd"]); ok && v > 0 {
		micros := int64(math.Round(v * 1_000_000))
		return &micros
	}
	return nil
}

func firstString(payload map[string]any, keys ...string) string {
	if usage, ok := payload["usage"].(map[string]any); ok {
		if v := firstString(usage, keys...); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstInt(payload map[string]any, keys ...string) int64 {
	if usage, ok := payload["usage"].(map[string]any); ok {
		if v := firstInt(usage, keys...); v != 0 {
			return v
		}
	}
	for _, key := range keys {
		if v, ok := asInt(payload[key]); ok {
			return v
		}
	}
	return 0
}

func asInt(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func configurationString(configuration any, key string) string {
	cfg, ok := configuration.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := cfg[key].(string)
	return strings.TrimSpace(v)
}

func configurationCredentialsSource(configuration any) string {
	cfg, ok := configuration.(map[string]any)
	if !ok {
		return ""
	}
	credentials, _ := cfg["credentials"].(map[string]any)
	if credentials == nil {
		return ""
	}
	source, _ := credentials["source"].(string)
	return strings.TrimSpace(source)
}
