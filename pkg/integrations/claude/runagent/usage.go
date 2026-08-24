package runagent

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

func RecordSessionUsage(usage core.UsageRecorder, logger *log.Entry, session *ManagedSession) {
	if usage == nil || session == nil || session.Usage == nil {
		return
	}
	model := session.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	input := session.Usage.InputTokens
	output := session.Usage.OutputTokens
	cacheRead := session.Usage.CacheReadInputTokens
	cacheWrite := session.Usage.CacheCreationInputTokens
	if input == 0 && output == 0 && cacheRead == 0 && cacheWrite == 0 {
		return
	}
	record := core.UsageRecord{
		Provider:         "anthropic",
		Model:            model,
		InputTokens:      input,
		OutputTokens:     output,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		TotalTokens:      input + output + cacheRead + cacheWrite,
	}
	if err := usage.Record(record); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record managed agent LLM usage")
	}
}
