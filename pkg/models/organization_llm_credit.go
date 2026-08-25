package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	LLMCreditGrantKindWelcome = "welcome"
	LLMCreditGrantKindAdmin   = "admin"
)

var (
	ErrHostedCreditEmpty      = errors.New("hosted LLM credit is empty")
	ErrHostedRunInFlight      = errors.New("another hosted LLM run is already in progress")
	ErrCreditGrantNotPositive = errors.New("credit grant must be greater than zero")
)

// OrganizationLLMCreditGrant is one append-only credit addition.
type OrganizationLLMCreditGrant struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Kind           string
	AmountMicros   int64
	Note           string
	ActorAccountID *uuid.UUID
	CreatedAt      time.Time
}

func (OrganizationLLMCreditGrant) TableName() string {
	return "organization_llm_credit_grants"
}

// OrganizationLLMSettings holds the hidden per-org markup override.
type OrganizationLLMSettings struct {
	OrganizationID uuid.UUID `gorm:"primary_key"`
	MarkupBPS      *int
	UpdatedAt      time.Time
}

func (OrganizationLLMSettings) TableName() string {
	return "organization_llm_settings"
}

// OrganizationLLMCreditHold marks one in-flight hosted run so concurrent
// PrepareHostedRun calls cannot all pass the remaining-credit gate.
type OrganizationLLMCreditHold struct {
	NodeExecutionID uuid.UUID `gorm:"primary_key"`
	OrganizationID  uuid.UUID
	AmountMicros    int64
	CreatedAt       time.Time
}

func (OrganizationLLMCreditHold) TableName() string {
	return "organization_llm_credit_holds"
}

// OrganizationLLMCreditSummary is remaining hosted credit for an org.
type OrganizationLLMCreditSummary struct {
	GrantMicros     int64
	BilledMicros    int64
	RemainingMicros int64
	MarkupBPS       int
	Warning         bool
}

func GrantWelcomeCredit(tx *gorm.DB, orgID uuid.UUID) error {
	settings, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return err
	}
	amount := CentsToMicros(settings.WelcomeGrantCents)
	if amount <= 0 {
		return nil
	}

	var existing OrganizationLLMCreditGrant
	err = tx.Where("organization_id = ? AND kind = ?", orgID, LLMCreditGrantKindWelcome).
		First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	grant := OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           LLMCreditGrantKindWelcome,
		AmountMicros:   amount,
		CreatedAt:      time.Now(),
	}
	if err := tx.Create(&grant).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil
		}
		return err
	}
	return nil
}

func AddAdminLLMCreditGrant(tx *gorm.DB, orgID uuid.UUID, amountMicros int64, note string, actorAccountID *uuid.UUID) (*OrganizationLLMCreditGrant, error) {
	if amountMicros <= 0 {
		return nil, ErrCreditGrantNotPositive
	}

	grant := OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           LLMCreditGrantKindAdmin,
		AmountMicros:   amountMicros,
		Note:           strings.TrimSpace(note),
		ActorAccountID: actorAccountID,
		CreatedAt:      time.Now(),
	}
	if err := tx.Create(&grant).Error; err != nil {
		return nil, err
	}
	return &grant, nil
}

func UpsertOrganizationLLMMarkup(tx *gorm.DB, orgID uuid.UUID, markupBPS *int) error {
	if markupBPS != nil && *markupBPS < 0 {
		return errors.New("markup cannot be negative")
	}

	settings := OrganizationLLMSettings{
		OrganizationID: orgID,
		MarkupBPS:      markupBPS,
		UpdatedAt:      time.Now(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"markup_bps", "updated_at"}),
	}).Create(&settings).Error
}

func FindOrganizationLLMSettings(tx *gorm.DB, orgID uuid.UUID) (*OrganizationLLMSettings, error) {
	var settings OrganizationLLMSettings
	err := tx.Where("organization_id = ?", orgID).First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &settings, nil
}

func ResolveOrganizationMarkupBPS(tx *gorm.DB, orgID uuid.UUID) (int, error) {
	orgSettings, err := FindOrganizationLLMSettings(tx, orgID)
	if err != nil {
		return 0, err
	}
	if orgSettings != nil && orgSettings.MarkupBPS != nil {
		return *orgSettings.MarkupBPS, nil
	}

	installation, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return 0, err
	}
	return installation.MarkupBPS, nil
}

func DescribeOrganizationLLMCredit(tx *gorm.DB, orgID uuid.UUID) (OrganizationLLMCreditSummary, error) {
	var grantMicros int64
	err := tx.Model(&OrganizationLLMCreditGrant{}).
		Select("COALESCE(SUM(amount_micros), 0)").
		Where("organization_id = ?", orgID).
		Scan(&grantMicros).Error
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	var billedMicros int64
	err = tx.Model(&LLMUsageEvent{}).
		Select("COALESCE(SUM(cost_micros), 0)").
		Where("organization_id = ? AND funding_source = ?", orgID, UsageFundingSourceHosted).
		Scan(&billedMicros).Error
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	remaining := grantMicros - billedMicros
	if remaining < 0 {
		remaining = 0
	}

	markupBPS, err := ResolveOrganizationMarkupBPS(tx, orgID)
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	installation, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	warning := false
	if grantMicros > 0 {
		threshold := grantMicros * int64(installation.WarningThresholdBPS) / int64(MarkupBaseBPS)
		warning = remaining <= threshold
	}

	return OrganizationLLMCreditSummary{
		GrantMicros:     grantMicros,
		BilledMicros:    billedMicros,
		RemainingMicros: remaining,
		MarkupBPS:       markupBPS,
		Warning:         warning,
	}, nil
}

func AssertHostedCreditAvailable(tx *gorm.DB, orgID uuid.UUID) error {
	summary, err := DescribeOrganizationLLMCredit(tx, orgID)
	if err != nil {
		return err
	}
	if summary.RemainingMicros <= 0 {
		return fmt.Errorf("%w: ask an installation admin to add credit", ErrHostedCreditEmpty)
	}
	return nil
}

// ReserveHostedCredit takes a short FOR UPDATE lock on organization LLM settings.
// Pass a committed connection, not a long-lived executor transaction.
func ReserveHostedCredit(tx *gorm.DB, orgID, nodeExecutionID uuid.UUID) error {
	if orgID == uuid.Nil {
		return fmt.Errorf("organization is required for hosted LLM credit")
	}
	if nodeExecutionID == uuid.Nil {
		return AssertHostedCreditAvailable(tx, orgID)
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		if err := lockOrganizationLLMSettings(inner, orgID); err != nil {
			return err
		}
		if err := releaseStaleHostedCreditHolds(inner, orgID); err != nil {
			return err
		}
		if err := AssertHostedCreditAvailable(inner, orgID); err != nil {
			return err
		}

		var inFlight int64
		err := inner.Model(&OrganizationLLMCreditHold{}).
			Where("organization_id = ? AND node_execution_id <> ?", orgID, nodeExecutionID).
			Count(&inFlight).Error
		if err != nil {
			return err
		}
		if inFlight > 0 {
			return fmt.Errorf("%w: wait for the current hosted run to finish", ErrHostedRunInFlight)
		}

		hold := OrganizationLLMCreditHold{
			NodeExecutionID: nodeExecutionID,
			OrganizationID:  orgID,
			AmountMicros:    1,
			CreatedAt:       time.Now(),
		}
		return inner.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_execution_id"}},
			DoNothing: true,
		}).Create(&hold).Error
	})
}

func ReleaseHostedCreditHold(tx *gorm.DB, nodeExecutionID uuid.UUID) error {
	if nodeExecutionID == uuid.Nil {
		return nil
	}
	return tx.Where("node_execution_id = ?", nodeExecutionID).Delete(&OrganizationLLMCreditHold{}).Error
}

func lockOrganizationLLMSettings(tx *gorm.DB, orgID uuid.UUID) error {
	settings := OrganizationLLMSettings{
		OrganizationID: orgID,
		UpdatedAt:      time.Now(),
	}
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoNothing: true,
	}).Create(&settings).Error
	if err != nil {
		return err
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", orgID).
		First(&settings).Error
}

// releaseStaleHostedCreditHolds deletes holds that no longer belong to an
// active node execution. Holds have no foreign key, so canvas deletion can
// leave orphan rows that would otherwise block every later hosted run.
func releaseStaleHostedCreditHolds(tx *gorm.DB, orgID uuid.UUID) error {
	return tx.
		Where("organization_id = ?", orgID).
		Where(`NOT EXISTS (
			SELECT 1
			FROM workflow_node_executions AS executions
			WHERE executions.id = organization_llm_credit_holds.node_execution_id
			  AND executions.state IN ?
		)`, CanvasNodeExecutionActiveStates).
		Delete(&OrganizationLLMCreditHold{}).Error
}
