package contexts

import (
	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// UsageContext records workspace spend for the current node execution.
// Inserts use a committed connection, not the node-executor transaction.
// The provider already billed the tokens or VM seconds; a later
// Emit/Fail/rollback must not drop the ledger row.
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
	return models.RecordUsage(database.Conn(), models.WorkspaceUsageEventInput{
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
		IdempotencyKey:   usageIdempotencyKey(record.IdempotencyKey, c.execution.ID),
	})
}

func (c *UsageContext) RecordCompute(record core.ComputeUsageRecord) error {
	return models.RecordComputeUsage(database.Conn(), models.ComputeUsageEventInput{
		OrganizationID:  c.organizationID,
		CanvasRunID:     c.execution.RunID,
		NodeExecutionID: c.execution.ID,
		NodeID:          c.execution.NodeID,
		MachineType:     record.MachineType,
		FleetID:         record.FleetID,
		DurationSeconds: record.DurationSeconds,
		IdempotencyKey:  usageIdempotencyKey(record.IdempotencyKey, c.execution.ID),
	})
}

func (c *UsageContext) ReleaseHostedCreditHold() error {
	return models.ReleaseHostedCreditHold(database.Conn(), c.execution.ID)
}

func usageIdempotencyKey(key string, executionID uuid.UUID) string {
	if key == "" {
		return ""
	}
	return key + ":" + executionID.String()
}
