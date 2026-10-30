package billing

import (
	"time"

	"github.com/google/uuid"
)

// UsageReportFilter scopes ledger aggregates.
type UsageReportFilter struct {
	OrganizationID uuid.UUID
	FactoryID      *uuid.UUID
	WorkOrderID    *uuid.UUID
	UsageKind      string
	Since          time.Time
	Until          time.Time
	Provider       string
	Model          string
	MachineType    string
	TaskOwnerID    *uuid.UUID
	FundingSource  string
}

// WorkOrderRunUsage is one task-run spend row grouped from the ledger.
type WorkOrderRunUsage struct {
	WorkOrderExecutionID uuid.UUID
	WorkOrderID          uuid.UUID
	WorkOrderNumber      int64
	Title                string
	LastOccurredAt       time.Time
	UserID               *uuid.UUID
	UserName             string
	UserEmail            string
	TotalTokens          int64
	DurationSeconds      int64
	CostMicros           int64
	HostedCostMicros     int64
	BYOKCostMicros       int64
	Models               []string
	BYOKModels           []string
	MachineTypes         []string
}

func (r WorkOrderRunUsage) CostCents() int64 {
	return MicrosToCents(r.CostMicros)
}

func (r WorkOrderRunUsage) HostedCostCents() int64 {
	return MicrosToCents(r.HostedCostMicros)
}

func (r WorkOrderRunUsage) BYOKCostCents() int64 {
	return MicrosToCents(r.BYOKCostMicros)
}

func (r WorkOrderRunUsage) TokenCostMicros() int64 {
	return r.HostedCostMicros + r.BYOKCostMicros
}

func (r WorkOrderRunUsage) ComputeCostMicros() int64 {
	remainder := r.CostMicros - r.TokenCostMicros()
	if remainder < 0 {
		return 0
	}
	return remainder
}

// UsageTotals is a token, duration, and cost sum.
type UsageTotals struct {
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

// UsageByModel is one model bucket in a spend report.
type UsageByModel struct {
	Provider    string
	Model       string
	TotalTokens int64
	CostMicros  int64
}

// UsageByMachineType is one fleet machine-type bucket in a compute report.
type UsageByMachineType struct {
	MachineType     string
	DurationSeconds int64
	CostMicros      int64
}

// UsageSplit is the ledger of one subject, divided by what the spend paid for.
type UsageSplit struct {
	// Model is spend on model tokens.
	Model UsageTotals
	// Compute is spend on runner machine time.
	Compute UsageTotals
}

func (t UsageTotals) CostCents() int64 {
	return MicrosToCents(t.CostMicros)
}

// Add returns the field-wise sum of two ledger totals.
func (t UsageTotals) Add(other UsageTotals) UsageTotals {
	return UsageTotals{
		TotalTokens:     t.TotalTokens + other.TotalTokens,
		DurationSeconds: t.DurationSeconds + other.DurationSeconds,
		CostMicros:      t.CostMicros + other.CostMicros,
	}
}

func (r UsageByModel) CostCents() int64 {
	return MicrosToCents(r.CostMicros)
}

func (r UsageByMachineType) CostCents() int64 {
	return MicrosToCents(r.CostMicros)
}

// Total is the whole ledger of the subject, both bands together.
func (s UsageSplit) Total() UsageTotals {
	return s.Model.Add(s.Compute)
}
