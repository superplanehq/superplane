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
