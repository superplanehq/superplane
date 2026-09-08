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
	LLMCreditGrantKindWelcome     = "welcome"
	LLMCreditGrantKindAdmin       = "admin"
	LLMCreditGrantKindPolar       = "polar"
	LLMCreditGrantKindPolarRefund = "polar_refund"
)

var (
	ErrHostedCreditEmpty        = errors.New("hosted LLM credit is empty")
	ErrCreditGrantNotPositive   = errors.New("credit grant must be greater than zero")
	ErrFactoryHostedBudgetEmpty = errors.New("this workspace has no remaining hosted credit")
	ErrPolarOrderIDRequired     = errors.New("polar order id is required")
	ErrPolarRefundIDRequired    = errors.New("polar refund id is required")
)

// OrganizationLLMCreditGrant is one append-only credit addition.
type OrganizationLLMCreditGrant struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Kind           string
	AmountMicros   int64
	Note           string
	ActorAccountID *uuid.UUID
	PolarOrderID   *string
	PolarRefundID  *string
	CreatedAt      time.Time
	ExpiresAt      *time.Time
}

func (g OrganizationLLMCreditGrant) IsExpired(now time.Time) bool {
	return g.ExpiresAt != nil && !g.ExpiresAt.After(now)
}

func (OrganizationLLMCreditGrant) TableName() string {
	return "organization_llm_credit_grants"
}

// OrganizationLLMSettings holds the hidden per-org markup override.
type OrganizationLLMSettings struct {
	OrganizationID  uuid.UUID `gorm:"primary_key"`
	MarkupBPS       *int
	PolarCustomerID *string
	UpdatedAt       time.Time
}

func (OrganizationLLMSettings) TableName() string {
	return "organization_llm_settings"
}

// OrganizationLLMCreditSummary is remaining hosted credit for an org.
type OrganizationLLMCreditSummary struct {
	GrantMicros            int64
	SuperPlaneGrantMicros  int64
	PurchasedCreditMicros  int64
	BilledMicros           int64
	RemainingMicros        int64
	MarkupBPS              int
	Warning                bool
	WelcomeCreditExpiresAt *time.Time
}

func GrantWelcomeCredit(tx *gorm.DB, orgID, accountID uuid.UUID) error {
	if accountID == uuid.Nil {
		return errors.New("account is required for welcome credit")
	}

	var account Account
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", accountID).
		First(&account).Error
	if err != nil {
		return err
	}
	if account.HasReceivedWelcomeCredit() {
		return nil
	}

	settings, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return err
	}
	amount := CentsToMicros(settings.WelcomeGrantCents)
	if amount <= 0 {
		return nil
	}

	now := time.Now()
	var existing OrganizationLLMCreditGrant
	err = tx.Where("organization_id = ? AND kind = ?", orgID, LLMCreditGrantKindWelcome).
		First(&existing).Error
	if err == nil {
		return stampWelcomeCreditGrantedAt(tx, accountID, existing.CreatedAt)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	expiresAt := now.Add(DefaultWelcomeGrantTTL)
	grant := OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           LLMCreditGrantKindWelcome,
		AmountMicros:   amount,
		ActorAccountID: &accountID,
		CreatedAt:      now,
		ExpiresAt:      &expiresAt,
	}
	if err := tx.Create(&grant).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return stampWelcomeCreditGrantedAt(tx, accountID, now)
		}
		return err
	}

	return stampWelcomeCreditGrantedAt(tx, accountID, now)
}

func stampWelcomeCreditGrantedAt(tx *gorm.DB, accountID uuid.UUID, grantedAt time.Time) error {
	return tx.Model(&Account{}).
		Where("id = ? AND welcome_credit_granted_at IS NULL", accountID).
		Update("welcome_credit_granted_at", grantedAt).Error
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

func AddPolarLLMCreditGrant(tx *gorm.DB, orgID uuid.UUID, amountMicros int64, polarOrderID string) (*OrganizationLLMCreditGrant, error) {
	if amountMicros <= 0 {
		return nil, ErrCreditGrantNotPositive
	}
	orderID := strings.TrimSpace(polarOrderID)
	if orderID == "" {
		return nil, ErrPolarOrderIDRequired
	}

	var existing OrganizationLLMCreditGrant
	err := tx.Where("polar_order_id = ? AND kind = ?", orderID, LLMCreditGrantKindPolar).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	grant := OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           LLMCreditGrantKindPolar,
		AmountMicros:   amountMicros,
		PolarOrderID:   &orderID,
		CreatedAt:      time.Now(),
	}
	if err := tx.Create(&grant).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return FindLLMCreditGrantByPolarOrderID(tx, orderID)
		}
		return nil, err
	}
	return &grant, nil
}

func FindLLMCreditGrantByPolarOrderID(tx *gorm.DB, polarOrderID string) (*OrganizationLLMCreditGrant, error) {
	var grant OrganizationLLMCreditGrant
	err := tx.Where("polar_order_id = ? AND kind = ?", polarOrderID, LLMCreditGrantKindPolar).First(&grant).Error
	if err != nil {
		return nil, err
	}
	return &grant, nil
}

func AddPolarLLMCreditRefund(tx *gorm.DB, orgID uuid.UUID, amountMicros int64, polarOrderID, polarRefundID string) (*OrganizationLLMCreditGrant, error) {
	if amountMicros <= 0 {
		return nil, ErrCreditGrantNotPositive
	}
	orderID := strings.TrimSpace(polarOrderID)
	if orderID == "" {
		return nil, ErrPolarOrderIDRequired
	}
	refundID := strings.TrimSpace(polarRefundID)
	if refundID == "" {
		return nil, ErrPolarRefundIDRequired
	}

	var existing OrganizationLLMCreditGrant
	err := tx.Where("polar_refund_id = ?", refundID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	grant := OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Kind:           LLMCreditGrantKindPolarRefund,
		AmountMicros:   -amountMicros,
		PolarOrderID:   &orderID,
		PolarRefundID:  &refundID,
		CreatedAt:      time.Now(),
	}
	if err := tx.Create(&grant).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return FindLLMCreditRefundByPolarRefundID(tx, refundID)
		}
		return nil, err
	}
	return &grant, nil
}

func FindLLMCreditRefundByPolarRefundID(tx *gorm.DB, polarRefundID string) (*OrganizationLLMCreditGrant, error) {
	var grant OrganizationLLMCreditGrant
	err := tx.Where("polar_refund_id = ?", polarRefundID).First(&grant).Error
	if err != nil {
		return nil, err
	}
	return &grant, nil
}

func PolarRefundMicrosForOrder(tx *gorm.DB, polarOrderID string) (int64, error) {
	var refundedMicros int64
	err := tx.Model(&OrganizationLLMCreditGrant{}).
		Select("COALESCE(SUM(-amount_micros), 0) AS refunded_micros").
		Where("polar_order_id = ? AND kind = ?", polarOrderID, LLMCreditGrantKindPolarRefund).
		Scan(&refundedMicros).Error
	if err != nil {
		return 0, err
	}
	return refundedMicros, nil
}

// ReversePolarOrderCredit inserts a Polar refund so the reversed total equals
// reverseMicros, and never more than the purchase grant. Concurrent callers
// serialize on the purchase grant row.
func ReversePolarOrderCredit(tx *gorm.DB, orgID uuid.UUID, polarOrderID string, reverseMicros int64, polarRefundID string) error {
	if reverseMicros <= 0 {
		return nil
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		grant, err := FindLLMCreditGrantByPolarOrderID(
			inner.Clauses(clause.Locking{Strength: "UPDATE"}),
			polarOrderID,
		)
		if err != nil {
			return err
		}
		already, err := PolarRefundMicrosForOrder(inner, polarOrderID)
		if err != nil {
			return err
		}
		remaining := grant.AmountMicros - already
		if remaining <= 0 {
			return nil
		}
		additional := reverseMicros - already
		if additional <= 0 {
			return nil
		}
		if additional > remaining {
			additional = remaining
		}
		_, err = AddPolarLLMCreditRefund(inner, orgID, additional, polarOrderID, polarRefundID)
		return err
	})
}

func SetOrganizationPolarCustomerID(tx *gorm.DB, orgID uuid.UUID, customerID string) error {
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return errors.New("billing customer id is required")
	}
	settings := OrganizationLLMSettings{
		OrganizationID:  orgID,
		PolarCustomerID: &customerID,
		UpdatedAt:       time.Now(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"polar_customer_id", "updated_at"}),
	}).Create(&settings).Error
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

func ListOrganizationLLMCreditGrants(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationLLMCreditGrant, error) {
	var grants []OrganizationLLMCreditGrant
	err := tx.Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&grants).Error
	if err != nil {
		return nil, err
	}
	return grants, nil
}

func CreditGrantActorNames(tx *gorm.DB, grants []OrganizationLLMCreditGrant) (map[uuid.UUID]string, error) {
	ids := uniqueCreditGrantActorIDs(grants)
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	var accounts []Account
	err := tx.Select("id", "name", "email").Where("id IN ?", ids).Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	names := make(map[uuid.UUID]string, len(accounts))
	for _, account := range accounts {
		names[account.ID] = creditGrantActorDisplayName(account)
	}
	return names, nil
}

func uniqueCreditGrantActorIDs(grants []OrganizationLLMCreditGrant) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, grant := range grants {
		if grant.ActorAccountID == nil || *grant.ActorAccountID == uuid.Nil {
			continue
		}
		if _, ok := seen[*grant.ActorAccountID]; ok {
			continue
		}
		seen[*grant.ActorAccountID] = struct{}{}
		ids = append(ids, *grant.ActorAccountID)
	}
	return ids
}

func creditGrantActorDisplayName(account Account) string {
	if name := strings.TrimSpace(account.Name); name != "" {
		return name
	}
	return strings.TrimSpace(account.Email)
}

func DescribeOrganizationLLMCredit(tx *gorm.DB, orgID uuid.UUID) (OrganizationLLMCreditSummary, error) {
	grants, err := ListOrganizationLLMCreditGrants(tx, orgID)
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	var grantMicros, superplaneGrantMicros, purchasedCreditMicros int64
	var welcomeExpiresAt *time.Time
	now := time.Now()
	for _, grant := range grants {
		grantMicros += grant.AmountMicros
		switch grant.Kind {
		case LLMCreditGrantKindWelcome:
			superplaneGrantMicros += grant.AmountMicros
			welcomeExpiresAt = grant.ExpiresAt
		case LLMCreditGrantKindAdmin:
			superplaneGrantMicros += grant.AmountMicros
		case LLMCreditGrantKindPolar, LLMCreditGrantKindPolarRefund:
			purchasedCreditMicros += grant.AmountMicros
		}
	}

	billedMicros, err := sumHostedModelBilledMicros(tx, orgID, nil)
	if err != nil {
		return OrganizationLLMCreditSummary{}, err
	}

	billedBeforeExpiry := int64(0)
	if horizon := expiredGrantHorizon(grants, now); horizon != nil {
		billedBeforeExpiry, err = sumHostedModelBilledMicros(tx, orgID, horizon)
		if err != nil {
			return OrganizationLLMCreditSummary{}, err
		}
	}

	remaining := remainingHostedCreditMicros(grants, billedMicros, billedBeforeExpiry, now)

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
		GrantMicros:            grantMicros,
		SuperPlaneGrantMicros:  superplaneGrantMicros,
		PurchasedCreditMicros:  purchasedCreditMicros,
		BilledMicros:           billedMicros,
		RemainingMicros:        remaining,
		MarkupBPS:              markupBPS,
		Warning:                warning,
		WelcomeCreditExpiresAt: welcomeExpiresAt,
	}, nil
}

func sumHostedModelBilledMicros(tx *gorm.DB, orgID uuid.UUID, atOrBefore *time.Time) (int64, error) {
	query := tx.Model(&WorkspaceUsageEvent{}).
		Select("COALESCE(SUM(cost_micros), 0)").
		Where("organization_id = ? AND funding_source = ? AND usage_kind = ?", orgID, UsageFundingSourceHosted, UsageKindModel)
	if atOrBefore != nil {
		query = query.Where("occurred_at <= ?", *atOrBefore)
	}

	var billedMicros int64
	err := query.Scan(&billedMicros).Error
	if err != nil {
		return 0, err
	}
	return billedMicros, nil
}

func expiredGrantHorizon(grants []OrganizationLLMCreditGrant, now time.Time) *time.Time {
	var horizon *time.Time
	for _, grant := range grants {
		if !grant.IsExpired(now) || grant.ExpiresAt == nil {
			continue
		}
		if horizon == nil || grant.ExpiresAt.After(*horizon) {
			expiresAt := *grant.ExpiresAt
			horizon = &expiresAt
		}
	}
	return horizon
}

func remainingHostedCreditMicros(grants []OrganizationLLMCreditGrant, billedMicros, billedBeforeExpiryMicros int64, now time.Time) int64 {
	var expired, usable int64
	for _, grant := range grants {
		if grant.IsExpired(now) {
			expired += grant.AmountMicros
			continue
		}
		usable += grant.AmountMicros
	}

	forgiven := int64(0)
	if expired > 0 {
		forgiven = billedBeforeExpiryMicros
		if forgiven > expired {
			forgiven = expired
		}
	}

	spendAgainstUsable := billedMicros - forgiven
	if spendAgainstUsable < 0 {
		spendAgainstUsable = 0
	}
	remaining := usable - spendAgainstUsable
	if remaining < 0 {
		return 0
	}
	return remaining
}

func AssertHostedCreditAvailable(tx *gorm.DB, orgID uuid.UUID) error {
	summary, err := DescribeOrganizationLLMCredit(tx, orgID)
	if err != nil {
		return err
	}
	if summary.RemainingMicros <= 0 {
		return ErrHostedCreditEmpty
	}
	return nil
}

func AssertFactoryHostedBudgetAvailable(tx *gorm.DB, factory *Factory) error {
	if factory == nil {
		return nil
	}
	summary, err := DescribeFactoryHostedBudget(tx, factory)
	if err != nil {
		return err
	}
	if !summary.Capped {
		return nil
	}
	if summary.RemainingMicros <= 0 {
		return ErrFactoryHostedBudgetEmpty
	}
	return nil
}

// AssertHostedRunAllowed rejects a new hosted start when org remaining credit
// is empty or the factory hosted budget is exhausted. Pass a committed
// connection so remaining credit includes billed spend from other runs.
func AssertHostedRunAllowed(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID) error {
	if orgID == uuid.Nil {
		return fmt.Errorf("organization is required for hosted LLM credit")
	}
	if err := AssertHostedCreditAvailable(tx, orgID); err != nil {
		return err
	}
	if factoryID == nil || *factoryID == uuid.Nil {
		return nil
	}
	factory, err := FindFactory(tx, orgID, *factoryID)
	if err != nil {
		return err
	}
	return AssertFactoryHostedBudgetAvailable(tx, factory)
}
