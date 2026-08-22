package contexts

import (
	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// UsageContext records LLM spend for the current node execution.
// Inserts use a committed connection, not the node-executor transaction.
// The provider already billed the tokens; a later Emit/Fail/rollback must
// not drop the ledger row.
type UsageContext struct {
	organizationID uuid.UUID
	execution      *models.CanvasNodeExecution
}

func NewUsageContext(organizationID uuid.UUID, execution *models.CanvasNodeExecution) *UsageContext {
	return &UsageContext{
		organizationID: organizationID,
		execution:      execution,
	}
}

func (c *UsageContext) Record(record core.UsageRecord) error {
	return models.RecordUsage(database.Conn(), models.LLMUsageEventInput{
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
		FundingSource:    record.FundingSource,
	})
}
