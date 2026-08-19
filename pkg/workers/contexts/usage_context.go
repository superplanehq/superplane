package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// UsageContext records LLM spend for the current node execution.
type UsageContext struct {
	tx             *gorm.DB
	organizationID uuid.UUID
	execution      *models.CanvasNodeExecution
}

func NewUsageContext(tx *gorm.DB, organizationID uuid.UUID, execution *models.CanvasNodeExecution) *UsageContext {
	return &UsageContext{
		tx:             tx,
		organizationID: organizationID,
		execution:      execution,
	}
}

func (c *UsageContext) Record(record core.UsageRecord) error {
	return models.RecordUsage(c.tx, models.LLMUsageEventInput{
		OrganizationID:   c.organizationID,
		CanvasRunID:      c.execution.RunID,
		NodeExecutionID:  c.execution.ID,
		NodeID:           c.execution.NodeID,
		Provider:         record.Provider,
		Model:            record.Model,
		InputTokens:      record.InputTokens,
		OutputTokens:     record.OutputTokens,
		CacheReadTokens:  record.CacheReadTokens,
		CacheWriteTokens: record.CacheWriteTokens,
		ReasoningTokens:  record.ReasoningTokens,
		TotalTokens:      record.TotalTokens,
		CostMicros:       record.CostMicros,
	})
}
