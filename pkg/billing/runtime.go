// Package billing defines the boundary between core workflows and billing.
// Implementations are installed by the server before initialization or work.
package billing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const BillingEnabledEnv = "SUPERPLANE_BILLING_ENABLED"

// Enabled preserves the installation's existing billing switch.
func Enabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(BillingEnabledEnv))) {
	case "no", "false":
		return false
	default:
		return true
	}
}

var ErrImplementationMissing = errors.New("billing implementation is not installed")

// DeletionGuard verifies the prepared provider snapshot under the caller's
// organization lock. Provider requests must finish before the transaction.
type DeletionGuard interface {
	Validate(tx *gorm.DB, organizationID uuid.UUID) error
}

// CapacityAlert describes a transition staged in a caller-owned transaction.
// Only the owner of the outermost transaction emits it after a successful commit.
type CapacityAlert interface{ Emit() }
type CapacityAlerts []CapacityAlert

func (alerts CapacityAlerts) Emit() {
	for _, alert := range alerts {
		if alert != nil {
			alert.Emit()
		}
	}
}

// Provider verifies external billing state and stages capacity transitions.
type Provider interface {
	PrepareDeletion(context.Context, uuid.UUID) (DeletionGuard, error)
	PrepareAdmission(context.Context, uuid.UUID) error
	ValidateMemberAdmission(*gorm.DB, uuid.UUID) error
	ValidateInvitationCapacity(*gorm.DB, uuid.UUID, bool, int) error
	RecheckSeatCapacity(*gorm.DB, uuid.UUID, string) (CapacityAlert, error)
	AssertOrganizationPaymentMethod(*gorm.DB, uuid.UUID) error
}

// Runtime owns billing rules and persistence. Core owns its workflow transactions.
type Runtime interface {
	JoinLimit(*gorm.DB, uuid.UUID) (*int, error)
	ValidateSpendingReportWindow(time.Time, time.Time) error
	GetPastDueGraceDays(*gorm.DB) (int, error)
	UpdatePastDueGraceDays(*gorm.DB, int) error
	ValidatePastDueGraceDays(int) error
	Initialize(*gorm.DB) error
	InitializeOrganization(*gorm.DB, uuid.UUID) error
	GrantWelcomeCredit(*gorm.DB, uuid.UUID, uuid.UUID) error
	PrepareDeletion(context.Context, uuid.UUID) (DeletionGuard, error)
	LocalDeletionGuard(*gorm.DB, uuid.UUID) (DeletionGuard, error)
	PrepareAdmission(context.Context, uuid.UUID) error
	ValidateMemberAdmission(*gorm.DB, uuid.UUID) error
	ValidateInvitationCapacity(*gorm.DB, uuid.UUID, bool, int) error
	ResolveSeats(*gorm.DB, uuid.UUID) (*SeatUsage, error)
	RecheckSeatCapacity(*gorm.DB, uuid.UUID, string) (CapacityAlert, error)
	AssertOrganizationPaymentMethod(*gorm.DB, uuid.UUID) error
	AssertOrganizationPlanAllowsRuns(*gorm.DB, uuid.UUID) error
	AssertHostedRunAllowed(*gorm.DB, uuid.UUID, *uuid.UUID) error
	ResolveParallelTaskLimit(*gorm.DB, uuid.UUID) (*int, error)
	ResolveProductFeatures(*gorm.DB, uuid.UUID) (map[string]bool, error)
	ResolveMarkupBPS(*gorm.DB, uuid.UUID) (int, error)
	RemainingCreditMicros(*gorm.DB, []uuid.UUID) (map[uuid.UUID]int64, error)
	DescribeCredit(*gorm.DB, uuid.UUID) (CreditSummary, error)
	DescribeFactoryBudget(*gorm.DB, uuid.UUID, uuid.UUID) (FactoryBudget, error)
	UpdateFactoryHostedBudget(*gorm.DB, uuid.UUID, uuid.UUID, *int64) error
	PriceUsage(*gorm.DB, uuid.UUID, Usage) (UsagePrice, error)
	PriceCompute(ComputeUsage) (UsagePrice, error)
	LookupRate(string, string) (Rate, bool)
	GetInstallationSettings(*gorm.DB) (InstallationSettings, error)
	UpdateInstallationSettings(*gorm.DB, InstallationSettings) error
	SummarizeUsage(*gorm.DB, UsageReportFilter) (UsageTotals, []UsageByModel, error)
	SummarizeComputeUsage(*gorm.DB, UsageReportFilter) (UsageTotals, []UsageByMachineType, error)
	ListWorkOrderRunUsage(*gorm.DB, UsageReportFilter, int, int) ([]WorkOrderRunUsage, int64, error)
	ListAllWorkOrderRunUsage(*gorm.DB, UsageReportFilter) ([]WorkOrderRunUsage, error)
	SummarizeUsageByLocalDay(*gorm.DB, UsageReportFilter, []time.Time) ([]UsageSplit, error)
}

var binding struct {
	sync.RWMutex
	implementation Runtime
}

// Bind installs billing even when enforcement is disabled, because usage still
// needs pricing. The returned function restores the previous binding in tests.
func Bind(implementation Runtime) func() {
	binding.Lock()
	previous := binding.implementation
	binding.implementation = implementation
	binding.Unlock()
	return func() { binding.Lock(); binding.implementation = previous; binding.Unlock() }
}

func Current() (Runtime, error) {
	binding.RLock()
	implementation := binding.implementation
	binding.RUnlock()
	if implementation == nil {
		return nil, ErrImplementationMissing
	}
	return implementation, nil
}

func Require() Runtime {
	implementation, err := Current()
	if err != nil {
		panic(fmt.Sprintf("billing runtime: %v", err))
	}
	return implementation
}
