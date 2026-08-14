package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	factoryNameUniqueConstraint = "factories_organization_id_name_key"
	factoryKeyUniqueConstraint  = "factories_organization_id_key_active_key"

	FactoryKeyMinLength = 2
	FactoryKeyMaxLength = 5
)

var ErrFactoryNameAlreadyExists = errors.New("factory name already exists")
var ErrFactoryNameRequired = errors.New("factory name is required")
var ErrFactoryNotFound = errors.New("factory not found")
var ErrFactoryWorkOrderTitleRequired = errors.New("title is required")
var ErrFactoryKeyRequired = errors.New("factory key is required")
var ErrFactoryKeyInvalid = errors.New("factory key must be 2 to 5 uppercase letters")
var ErrFactoryKeyAlreadyExists = errors.New("factory key already exists in this organization")

var factoryKeyPattern = regexp.MustCompile(`^[A-Z]{2,5}$`)

type Factory struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	Name                string
	Description         string
	Key                 string
	NextWorkOrderNumber int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

// NormalizeFactoryKey uppercases and trims whitespace so callers can accept
// user input in any case, then re-check it with ValidateFactoryKey.
func NormalizeFactoryKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}

// ValidateFactoryKey rejects empty or malformed keys with a stable error
// that API handlers translate into an `InvalidArgument` response. Callers
// should normalize input with NormalizeFactoryKey first.
func ValidateFactoryKey(key string) error {
	if key == "" {
		return ErrFactoryKeyRequired
	}
	if !factoryKeyPattern.MatchString(key) {
		return ErrFactoryKeyInvalid
	}
	return nil
}

// GenerateFactoryKeyFromName produces a stable candidate key from a factory
// name (letters only, uppercased, trimmed to the max length). Callers still
// need to check organization uniqueness before persisting.
func GenerateFactoryKeyFromName(name string) string {
	letters := regexp.MustCompile(`[^A-Za-z]`).ReplaceAllString(name, "")
	upper := strings.ToUpper(letters)
	if len(upper) > FactoryKeyMaxLength {
		upper = upper[:FactoryKeyMaxLength]
	}
	if len(upper) < FactoryKeyMinLength {
		return ""
	}
	return upper
}

// MapFactoryConstraintError converts Postgres unique-constraint violations
// into the domain-specific errors that upper layers know how to translate
// into user-facing responses.
func MapFactoryConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case factoryNameUniqueConstraint:
			return ErrFactoryNameAlreadyExists
		case factoryKeyUniqueConstraint:
			return ErrFactoryKeyAlreadyExists
		}
	}

	return err
}

// MapFactoryNameUniqueConstraintError is a compatibility shim; call
// MapFactoryConstraintError instead in new code.
func MapFactoryNameUniqueConstraintError(err error) error {
	return MapFactoryConstraintError(err)
}

func CreateFactory(tx *gorm.DB, organizationID uuid.UUID, name, description, key string) (*Factory, error) {
	normalizedKey := NormalizeFactoryKey(key)
	if normalizedKey == "" {
		// Callers that omit the key (tests, CLI, backfill utilities) get a
		// deterministic name-derived key with a numeric suffix on collision.
		generated, err := GenerateUniqueFactoryKey(tx, organizationID, name)
		if err != nil {
			return nil, err
		}
		normalizedKey = generated
	}
	if err := ValidateFactoryKey(normalizedKey); err != nil {
		return nil, err
	}

	now := time.Now()
	factory := &Factory{
		ID:                  uuid.New(),
		OrganizationID:      organizationID,
		Name:                name,
		Description:         description,
		Key:                 normalizedKey,
		NextWorkOrderNumber: 1,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(factory).Error; err != nil {
		return nil, MapFactoryConstraintError(err)
	}

	return factory, nil
}

// GenerateUniqueFactoryKey picks a key that is unique among active
// factories in the given organization. It first tries the name-derived
// seed, then walks through letter-only variants. Used by tests and by the
// CLI, where callers may not care about picking a specific key.
//
// Keys are letters-only by the schema check constraint, so we cannot fall
// back to numeric suffixes. Instead we pad with `X` and cycle through the
// last character.
func GenerateUniqueFactoryKey(tx *gorm.DB, organizationID uuid.UUID, name string) (string, error) {
	seed := GenerateFactoryKeyFromName(name)
	if seed == "" {
		seed = "WS"
	}
	if len(seed) < FactoryKeyMinLength {
		seed = seed + strings.Repeat("X", FactoryKeyMinLength-len(seed))
	}

	baseCandidates := []string{seed}
	for length := FactoryKeyMinLength; length <= FactoryKeyMaxLength; length++ {
		if length == len(seed) {
			continue
		}
		if len(seed) >= length {
			baseCandidates = append(baseCandidates, seed[:length])
			continue
		}
		baseCandidates = append(baseCandidates, seed+strings.Repeat("X", length-len(seed)))
	}

	tried := map[string]bool{}
	for _, base := range baseCandidates {
		for letter := 'A'; letter <= 'Z'; letter++ {
			candidate := base
			if tried[candidate] {
				candidate = base[:len(base)-1] + string(letter)
			}
			if tried[candidate] {
				continue
			}
			tried[candidate] = true

			var count int64
			err := tx.Model(&Factory{}).
				Where("organization_id = ? AND key = ?", organizationID, candidate).
				Count(&count).Error
			if err != nil {
				return "", err
			}
			if count == 0 {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("could not generate unique factory key from %q", name)
}

func FindFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Where("organization_id = ? AND id = ?", organizationID, factoryID).
		First(&factory).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryNotFound
		}
		return nil, err
	}

	return &factory, nil
}

func ListFactories(tx *gorm.DB, organizationID uuid.UUID) ([]Factory, error) {
	var factories []Factory
	err := tx.
		Where("organization_id = ?", organizationID).
		Order("name ASC").
		Order("id ASC").
		Find(&factories).
		Error
	if err != nil {
		return nil, err
	}

	return factories, nil
}

func (f *Factory) SoftDelete(tx *gorm.DB) error {
	now := time.Now()
	newName := fmt.Sprintf("%s (deleted-%d)", f.Name, now.Unix())

	err := tx.Model(f).Updates(map[string]any{
		"name":       newName,
		"deleted_at": now,
		"updated_at": now,
	}).Error
	if err != nil {
		return err
	}

	f.Name = newName
	f.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	f.UpdatedAt = now
	return nil
}

func (f *Factory) Update(tx *gorm.DB, name, description, key *string) error {
	updates := map[string]any{}

	if name != nil {
		nextName := strings.TrimSpace(*name)
		if nextName == "" {
			return ErrFactoryNameRequired
		}
		if f.Name != nextName {
			updates["name"] = nextName
		}
	}

	if description != nil && f.Description != *description {
		updates["description"] = *description
	}

	if key != nil {
		nextKey := NormalizeFactoryKey(*key)
		if err := ValidateFactoryKey(nextKey); err != nil {
			return err
		}
		if f.Key != nextKey {
			updates["key"] = nextKey
		}
	}

	if len(updates) == 0 {
		return nil
	}

	now := time.Now()
	updates["updated_at"] = now

	err := MapFactoryConstraintError(
		tx.Model(f).
			Where("organization_id = ? AND id = ?", f.OrganizationID, f.ID).
			Updates(updates).
			Error,
	)
	if err != nil {
		return err
	}

	if nextName, ok := updates["name"].(string); ok {
		f.Name = nextName
	}
	if nextDescription, ok := updates["description"].(string); ok {
		f.Description = nextDescription
	}
	if nextKey, ok := updates["key"].(string); ok {
		f.Key = nextKey
	}
	f.UpdatedAt = now
	return nil
}

func (f *Factory) ListCanvases(tx *gorm.DB) ([]Canvas, error) {
	var canvases []Canvas
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Order("name ASC").
		Order("id ASC").
		Find(&canvases).
		Error
	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListDeletedFactories(tx *gorm.DB) ([]Factory, error) {
	var factories []Factory
	err := tx.
		Model(&Factory{}).
		Unscoped().
		Joins("JOIN organizations ON organizations.id = factories.organization_id").
		Select(
			"factories.id",
			"factories.organization_id",
			"factories.name",
			"factories.description",
			"factories.key",
			"factories.next_work_order_number",
			"factories.created_at",
			"factories.updated_at",
			// Earliest deletion wins so neither factory nor org soft-delete resets grace.
			"LEAST(factories.deleted_at, organizations.deleted_at) AS deleted_at",
		).
		Where("factories.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		Find(&factories).
		Error
	if err != nil {
		return nil, err
	}

	return factories, nil
}

func LockDeletedFactory(tx *gorm.DB, id uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Unscoped().
		Model(&Factory{}).
		Joins("JOIN organizations ON organizations.id = factories.organization_id").
		Select(
			"factories.id",
			"factories.organization_id",
			"factories.name",
			"factories.description",
			"factories.key",
			"factories.next_work_order_number",
			"factories.created_at",
			"factories.updated_at",
			"LEAST(factories.deleted_at, organizations.deleted_at) AS deleted_at",
		).
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "factories"},
			Options:  "SKIP LOCKED",
		}).
		Where("factories.id = ?", id).
		Where("factories.deleted_at IS NOT NULL OR organizations.deleted_at IS NOT NULL").
		First(&factory).
		Error
	if err != nil {
		return nil, err
	}

	return &factory, nil
}

func (f *Factory) CountCanvases(tx *gorm.DB) (int64, error) {
	var count int64
	err := tx.Unscoped().
		Model(&Canvas{}).
		Where("factory_id = ?", f.ID).
		Count(&count).
		Error
	return count, err
}

func (f *Factory) SoftDeleteCanvases(tx *gorm.DB) error {
	canvases, err := f.ListCanvases(tx)
	if err != nil {
		return err
	}

	for i := range canvases {
		if err := canvases[i].SoftDeleteInTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

func CountFactoriesByOrganization(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Unscoped().
		Model(&Factory{}).
		Where("organization_id = ?", organizationID).
		Count(&count).
		Error
	return count, err
}

func SoftDeleteOrganizationFactories(tx *gorm.DB, organizationID uuid.UUID) error {
	var factories []Factory
	err := tx.
		Where("organization_id = ?", organizationID).
		Find(&factories).
		Error
	if err != nil {
		return err
	}

	for i := range factories {
		if factories[i].DeletedAt.Valid {
			continue
		}
		if err := factories[i].SoftDelete(tx); err != nil {
			return err
		}
	}

	return nil
}

func (f *Factory) CreateWorkOrder(tx *gorm.DB, title, description string, createdBy *uuid.UUID, assignees []uuid.UUID, sourceRunID *uuid.UUID) (*FactoryWorkOrder, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrFactoryWorkOrderTitleRequired
	}

	// Allocate the sequence number atomically: the UPDATE ... RETURNING
	// increments `next_work_order_number` and hands back the previous
	// value in one round-trip, so concurrent inserts cannot collide even
	// without an explicit row lock. The parent transaction rolls the
	// counter back if anything below fails.
	nextNumber, err := f.allocateNextWorkOrderNumber(tx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	order := &FactoryWorkOrder{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Number:         nextNumber,
		Title:          title,
		Description:    description,
		State:          FactoryWorkOrderStateDraft,
		Result:         "",
		CreatedByID:    createdBy,
		SourceRunID:    sourceRunID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(order).Error; err != nil {
		return nil, err
	}

	if len(assignees) > 0 {
		if err := order.ReplaceAssignees(tx, assignees); err != nil {
			return nil, err
		}
	}

	// Creation is a status transition into `draft` (fromState == "").
	// For orders spawned by a canvas run we snapshot the originating
	// run + app here so the very first timeline entry links back to
	// the run that created the order — matching the enrichment
	// UpdateStatus performs on the draft → open promotion.
	initialStatus := statusUpdatedRecord{
		Actor:   createdBy,
		ToState: FactoryWorkOrderStateDraft,
	}
	if sourceRunID != nil {
		sourceRun, sourceApp, err := order.loadSourceRunRefs(tx)
		if err != nil {
			return nil, err
		}
		initialStatus.Run = sourceRun
		initialStatus.App = sourceApp
	}
	if err := order.RecordStatusUpdated(tx, initialStatus); err != nil {
		return nil, err
	}

	return f.FindWorkOrder(tx, order.ID)
}

// FindWorkOrderByArtifactKey resolves a work order from one of its
// artifacts' `key` values, then delegates to FindWorkOrder so the result
// gets the same preloads/scoping as every other lookup path.
func (f *Factory) FindWorkOrderByArtifactKey(tx *gorm.DB, key string) (*FactoryWorkOrder, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, ErrFactoryWorkOrderNotFound
	}

	var artifact FactoryWorkOrderArtifact
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND key = ?", f.OrganizationID, f.ID, trimmedKey).
		First(&artifact).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderNotFound
		}
		return nil, err
	}

	return f.FindWorkOrder(tx, artifact.WorkOrderID)
}

func (f *Factory) FindWorkOrder(tx *gorm.DB, orderID uuid.UUID) (*FactoryWorkOrder, error) {
	var order FactoryWorkOrder
	err := tx.
		Preload("CreatedBy").
		Preload("Assignees").
		Preload("Assignees.User").
		Where("organization_id = ? AND factory_id = ? AND id = ?", f.OrganizationID, f.ID, orderID).
		First(&order).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

type ListFactoryWorkOrdersFilters struct {
	AssigneeIDs []uuid.UUID
	States      []string
	Results     []string
	Unassigned  *bool
}

func (f *Factory) ListWorkOrders(tx *gorm.DB, filters ListFactoryWorkOrdersFilters) ([]FactoryWorkOrder, error) {
	query := tx.
		Model(&FactoryWorkOrder{}).
		Preload("CreatedBy").
		Preload("Assignees").
		Preload("Assignees.User").
		Where("factory_work_orders.organization_id = ?", f.OrganizationID).
		Where("factory_work_orders.factory_id = ?", f.ID)

	if len(filters.States) > 0 {
		query = query.Where("factory_work_orders.state IN ?", filters.States)
	}

	if len(filters.Results) > 0 {
		query = query.Where("factory_work_orders.result IN ?", filters.Results)
	}

	if filters.Unassigned != nil && *filters.Unassigned {
		query = query.Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_assignees
				WHERE factory_work_order_assignees.work_order_id = factory_work_orders.id
			)`)
	}

	if len(filters.AssigneeIDs) > 0 {
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM factory_work_order_assignees
				WHERE factory_work_order_assignees.work_order_id = factory_work_orders.id
				AND factory_work_order_assignees.user_id IN ?
			)`, filters.AssigneeIDs)
	}

	var orders []FactoryWorkOrder
	err := query.
		Order("factory_work_orders.created_at DESC").
		Order("factory_work_orders.id DESC").
		Find(&orders).
		Error
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// allocateNextWorkOrderNumber atomically increments the factory's counter
// and returns the value that the new work order should use. The UPDATE ...
// RETURNING pattern serializes concurrent inserts inside Postgres without
// requiring an application-level lock; if the surrounding transaction
// rolls back the counter reverts with it.
func (f *Factory) allocateNextWorkOrderNumber(tx *gorm.DB) (int64, error) {
	var allocated int64
	err := tx.Raw(`
		UPDATE factories
		SET next_work_order_number = next_work_order_number + 1,
		    updated_at = NOW()
		WHERE id = ? AND organization_id = ?
		RETURNING next_work_order_number - 1
	`, f.ID, f.OrganizationID).Scan(&allocated).Error
	if err != nil {
		return 0, err
	}
	if allocated <= 0 {
		return 0, fmt.Errorf("factory %s: could not allocate work order number", f.ID)
	}

	f.NextWorkOrderNumber = allocated + 1
	return allocated, nil
}

// WorkOrderKey returns the display identifier used for a work order that
// belongs to this factory. Format matches `<KEY>-<number>` (for example
// `SP-42`).
func (f *Factory) WorkOrderKey(number int64) string {
	return fmt.Sprintf("%s-%d", f.Key, number)
}
