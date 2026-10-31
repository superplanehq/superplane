package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
)

// recordSetupAllocationUsage must commit with the allocation's terminal or
// reservation-release transition. It never relies on a connection or lease.
func (t *RunnerTask) recordSetupAllocationUsage(tx *gorm.DB, endedAt time.Time) error {
	if !t.SetupBillable || t.SetupPurpose == "" || t.ReservedAt == nil {
		return nil
	}
	if t.SetupPurpose != RunnerTaskSetupPurposeCodexLogin && t.SetupPurpose != RunnerTaskSetupPurposeCodexCatalog {
		return fmt.Errorf("unknown runner setup purpose %q", t.SetupPurpose)
	}
	startedAt := t.StartedAt
	if startedAt == nil {
		startedAt = t.ReservedAt
	}
	var seconds int64
	if elapsed := endedAt.Sub(*startedAt); elapsed > 0 {
		seconds = int64(elapsed / time.Second)
		if elapsed%time.Second != 0 {
			seconds++
		}
	}
	fleet, err := FindRunnerFleet(tx.Unscoped(), t.FleetID)
	if err != nil {
		return err
	}
	price, err := billing.PriceCompute(billing.ComputeUsage{
		MachineType: fleet.Slug, FleetID: fleet.Slug, DurationSeconds: seconds,
	})
	if err != nil {
		return fmt.Errorf("price ChatGPT setup compute: %w", err)
	}
	event := WorkspaceUsageEvent{
		ID: uuid.New(), OrganizationID: t.OrganizationID,
		RunnerTaskID: &t.ID, SetupPurpose: t.SetupPurpose,
		Provider: UsageProviderRunner, Model: fleet.Slug,
		UsageKind: UsageKindCompute, FundingSource: UsageFundingSourceHosted,
		DurationSeconds: seconds, MachineType: fleet.Slug, FleetID: fleet.Slug,
		CostMicros: price.CostMicros, ProviderCostMicros: price.ProviderCostMicros,
		Currency: "usd", PriceBookVersion: price.PriceBookVersion,
		IdempotencyKey: "runner:setup:" + t.ID.String() + ":" + t.ReservedAt.UTC().Format(time.RFC3339Nano),
		OccurredAt:     endedAt, CreatedAt: time.Now(),
	}
	if err := persistUsageEvent(tx, event, nil); err != nil {
		return fmt.Errorf("record ChatGPT setup compute: %w", err)
	}
	return nil
}
