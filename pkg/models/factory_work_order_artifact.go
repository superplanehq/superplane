package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderArtifactTypePR       = factory.ArtifactTypePR
	FactoryWorkOrderArtifactTypeMarkdown = factory.ArtifactTypeMarkdown
	FactoryWorkOrderArtifactTypeBranch   = factory.ArtifactTypeBranch
	FactoryWorkOrderArtifactTypeLink     = factory.ArtifactTypeLink

	// MaxFactoryWorkOrderArtifactDataBytes caps JSON-encoded artifact data.
	MaxFactoryWorkOrderArtifactDataBytes = 64 * 1024

	// MaxFactoryWorkOrderArtifactKeyBytes matches the `key` column width.
	MaxFactoryWorkOrderArtifactKeyBytes = 512
)

const factoryWorkOrderArtifactKeyUniqueConstraint = "idx_factory_work_order_artifacts_factory_key_unique"

// Valid values for a PR artifact's optional `data.state` field. Mirrors
// GitHub's own pull request lifecycle states so the UI can render the
// matching icon/color (see WorkOrderArtifactInline.tsx).
const (
	PrArtifactStateOpen   = "open"
	PrArtifactStateDraft  = "draft"
	PrArtifactStateClosed = "closed"
	PrArtifactStateMerged = "merged"
)

var validPrArtifactStates = map[string]bool{
	PrArtifactStateOpen:   true,
	PrArtifactStateDraft:  true,
	PrArtifactStateClosed: true,
	PrArtifactStateMerged: true,
}

var (
	ErrFactoryWorkOrderArtifactNotFound         = errors.New("factory work order artifact not found")
	ErrFactoryWorkOrderArtifactInvalid          = errors.New("invalid work order artifact")
	ErrFactoryWorkOrderArtifactKeyAlreadyExists = errors.New("factory work order artifact key already exists")
)

type FactoryWorkOrderArtifact struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Type           string
	Data           datatypes.JSON
	Key            *string
	// MergedAt / ClosedAt are append-only. A later reopen does not clear them.
	MergedAt    *time.Time
	ClosedAt    *time.Time
	CreatedByID *uuid.UUID
	CreatedAt   time.Time

	CreatedBy *User `gorm:"foreignKey:CreatedByID"`
}

func (FactoryWorkOrderArtifact) TableName() string {
	return "factory_work_order_artifacts"
}

type FactoryWorkOrderArtifactParams struct {
	Type       string
	Data       map[string]any
	Key        string
	CreatedBy  *uuid.UUID
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

// MapFactoryWorkOrderArtifactKeyUniqueConstraintError maps a violation of
// the per-factory unique `key` index to a sentinel error, mirroring
// MapFactoryNameUniqueConstraintError.
func MapFactoryWorkOrderArtifactKeyUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryWorkOrderArtifactKeyUniqueConstraint {
		return ErrFactoryWorkOrderArtifactKeyAlreadyExists
	}

	return err
}

// CreateArtifact writes the row and its `order.artifact.added` event
// in the same transaction so the timeline can't diverge from the list.
func (o *FactoryWorkOrder) CreateArtifact(
	db *gorm.DB,
	params FactoryWorkOrderArtifactParams,
) (*FactoryWorkOrderArtifact, error) {
	artifactType := strings.TrimSpace(params.Type)

	if err := validateArtifactData(artifactType, params.Data); err != nil {
		return nil, err
	}

	dataJSON, err := encodeArtifactData(params.Data)
	if err != nil {
		return nil, err
	}
	if len(dataJSON) > MaxFactoryWorkOrderArtifactDataBytes {
		return nil, fmt.Errorf(
			"%w: artifact data exceeds %d bytes",
			ErrFactoryWorkOrderArtifactInvalid,
			MaxFactoryWorkOrderArtifactDataBytes,
		)
	}

	// An explicitly empty key must land as NULL, not "" — the partial
	// unique index only excludes NULL, and two artifacts with key=""
	// in the same factory would otherwise collide.
	var key *string
	if trimmedKey := strings.TrimSpace(params.Key); trimmedKey != "" {
		if len(trimmedKey) > MaxFactoryWorkOrderArtifactKeyBytes {
			return nil, fmt.Errorf(
				"%w: artifact key exceeds %d bytes",
				ErrFactoryWorkOrderArtifactInvalid,
				MaxFactoryWorkOrderArtifactKeyBytes,
			)
		}
		key = &trimmedKey
	}

	now := time.Now()
	mergedAt, closedAt, err := extractPrLifecycleTimestamps(artifactType, params.Data, now, nil, nil, "")
	if err != nil {
		return nil, err
	}
	artifact := &FactoryWorkOrderArtifact{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           artifactType,
		Data:           dataJSON,
		Key:            key,
		MergedAt:       mergedAt,
		ClosedAt:       closedAt,
		CreatedByID:    params.CreatedBy,
		CreatedAt:      now,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		createErr := tx.Clauses(clause.Returning{}).Create(artifact).Error
		if createErr != nil {
			return MapFactoryWorkOrderArtifactKeyUniqueConstraintError(createErr)
		}

		ref := &factory.ArtifactRef{
			ID:   artifact.ID,
			Type: artifact.Type,
			Data: params.Data,
		}

		return o.RecordArtifactAdded(tx, ref, params.CreatedBy, params.Automation, params.Run)
	})
	if err != nil {
		return nil, err
	}

	return artifact, nil
}

// UpdateArtifactData resolves the artifact tagged with `key` under this
// work order (the same key an earlier CreateArtifact call set via
// FactoryWorkOrderArtifactParams.Key, typically the PR's URL), shallow-
// merges `updates` into its existing `data` map, and re-saves it in
// place. Unlike CreateArtifact, this does not append a new
// `order.artifact.*` timeline event — a PR flipping open → draft →
// merged should update the live chip, not spam the timeline with one
// entry per transition. Callers still notify the websocket channel
// (see FactoryContext.UpdateWorkOrderArtifact) so the UI refreshes.
func (o *FactoryWorkOrder) UpdateArtifactData(
	tx *gorm.DB,
	key string,
	updates map[string]any,
) (*FactoryWorkOrderArtifact, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, fmt.Errorf("%w: artifact key is required", ErrFactoryWorkOrderArtifactInvalid)
	}

	var artifact FactoryWorkOrderArtifact
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND work_order_id = ? AND key = ?", o.OrganizationID, o.FactoryID, o.ID, trimmedKey).
		First(&artifact).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderArtifactNotFound
		}
		return nil, err
	}

	merged := map[string]any{}
	if len(artifact.Data) > 0 {
		if err := json.Unmarshal(artifact.Data, &merged); err != nil {
			return nil, err
		}
	}
	for updateKey, value := range updates {
		merged[updateKey] = value
	}

	if err := validateArtifactData(artifact.Type, merged); err != nil {
		return nil, err
	}

	dataJSON, err := encodeArtifactData(merged)
	if err != nil {
		return nil, err
	}
	if len(dataJSON) > MaxFactoryWorkOrderArtifactDataBytes {
		return nil, fmt.Errorf(
			"%w: artifact data exceeds %d bytes",
			ErrFactoryWorkOrderArtifactInvalid,
			MaxFactoryWorkOrderArtifactDataBytes,
		)
	}

	mergedAt, closedAt, err := extractPrLifecycleTimestamps(
		artifact.Type,
		merged,
		time.Now(),
		artifact.MergedAt,
		artifact.ClosedAt,
		readArtifactState(artifact.Data),
	)
	if err != nil {
		return nil, err
	}

	columnUpdates := map[string]any{"data": dataJSON}
	if mergedAt != nil && artifact.MergedAt == nil {
		columnUpdates["merged_at"] = mergedAt
		artifact.MergedAt = mergedAt
	}
	if closedAt != nil && artifact.ClosedAt == nil {
		columnUpdates["closed_at"] = closedAt
		artifact.ClosedAt = closedAt
	}

	if err := tx.Model(&artifact).Updates(columnUpdates).Error; err != nil {
		return nil, err
	}
	artifact.Data = dataJSON

	return &artifact, nil
}

func (o *FactoryWorkOrder) ListArtifacts(tx *gorm.DB) ([]FactoryWorkOrderArtifact, error) {
	var artifacts []FactoryWorkOrderArtifact
	err := tx.
		Preload("CreatedBy").
		Where("work_order_id = ?", o.ID).
		Order("created_at DESC").
		Order("id DESC").
		Find(&artifacts).
		Error
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

type FactoryPRArtifactFilter struct {
	MergedFrom *time.Time
	MergedTo   *time.Time
	ClosedFrom *time.Time
	ClosedTo   *time.Time
	State      string
}

func ListFactoryPRArtifacts(tx *gorm.DB, factoryID uuid.UUID, filter FactoryPRArtifactFilter) ([]FactoryWorkOrderArtifact, error) {
	query := tx.
		Where("factory_id = ? AND type = ?", factoryID, FactoryWorkOrderArtifactTypePR)

	if filter.State != "" {
		if _, ok := validPrArtifactStates[filter.State]; !ok {
			return nil, fmt.Errorf("%w: invalid state filter %q", ErrFactoryWorkOrderArtifactInvalid, filter.State)
		}
	}

	switch filter.State {
	case PrArtifactStateMerged:
		query = query.Where("merged_at IS NOT NULL")
	case PrArtifactStateClosed:
		query = query.Where("closed_at IS NOT NULL AND merged_at IS NULL")
	}

	if filter.MergedFrom != nil {
		query = query.Where("merged_at >= ?", *filter.MergedFrom)
	}
	if filter.MergedTo != nil {
		query = query.Where("merged_at < ?", *filter.MergedTo)
	}
	if filter.ClosedFrom != nil {
		query = query.Where("closed_at >= ?", *filter.ClosedFrom)
	}
	if filter.ClosedTo != nil {
		query = query.Where("closed_at < ?", *filter.ClosedTo)
	}

	var artifacts []FactoryWorkOrderArtifact
	err := query.
		Order("merged_at DESC NULLS LAST").
		Order("closed_at DESC NULLS LAST").
		Find(&artifacts).
		Error
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

// IsValidWorkOrderArtifactType reports whether CreateArtifact accepts t.
func IsValidWorkOrderArtifactType(t string) bool {
	switch t {
	case FactoryWorkOrderArtifactTypePR, FactoryWorkOrderArtifactTypeMarkdown, FactoryWorkOrderArtifactTypeBranch, FactoryWorkOrderArtifactTypeLink:
		return true
	}
	return false
}

// validateArtifactData enforces the required-field rules for each
// artifact type (shared by CreateArtifact and UpdateArtifactData so a
// later merge can't drift from what a fresh attach requires) plus the
// URL-scheme guard that applies regardless of type.
func validateArtifactData(artifactType string, data map[string]any) error {
	switch artifactType {
	case FactoryWorkOrderArtifactTypePR:
		if extractArtifactString(data, "url") == "" {
			return fmt.Errorf("%w: pull request artifacts require a url", ErrFactoryWorkOrderArtifactInvalid)
		}
		if state := extractArtifactString(data, "state"); state != "" && !validPrArtifactStates[state] {
			return fmt.Errorf(
				"%w: invalid pull request state %q (want one of open, draft, closed, merged)",
				ErrFactoryWorkOrderArtifactInvalid,
				state,
			)
		}
		if err := validateOptionalTimestamp(data, "mergedAt"); err != nil {
			return err
		}
		if err := validateOptionalTimestamp(data, "closedAt"); err != nil {
			return err
		}
	case FactoryWorkOrderArtifactTypeMarkdown:
		if extractArtifactString(data, "body") == "" {
			return fmt.Errorf("%w: markdown artifacts require data.body", ErrFactoryWorkOrderArtifactInvalid)
		}
	case FactoryWorkOrderArtifactTypeBranch:
		if extractArtifactString(data, "name") == "" {
			return fmt.Errorf("%w: branch artifacts require data.name", ErrFactoryWorkOrderArtifactInvalid)
		}
	case FactoryWorkOrderArtifactTypeLink:
		if extractArtifactString(data, "url") == "" {
			return fmt.Errorf("%w: link artifacts require a url", ErrFactoryWorkOrderArtifactInvalid)
		}
	default:
		return fmt.Errorf("%w: unknown artifact type %q", ErrFactoryWorkOrderArtifactInvalid, artifactType)
	}

	// `data.url` lands in a clickable `href` for every artifact type the
	// UI knows about (extractArtifactUrl reads it unconditionally), so
	// reject non-http(s) schemes here rather than only inside the PR
	// branch — no caller should be able to smuggle `javascript:` past
	// the model.
	if artifactURL := extractArtifactString(data, "url"); artifactURL != "" {
		if !isSafeArtifactURL(artifactURL) {
			return fmt.Errorf("%w: artifact url must be http(s)", ErrFactoryWorkOrderArtifactInvalid)
		}
	}

	return nil
}

// isSafeArtifactURL requires an absolute http(s) URL with a host —
// rejects `javascript:`, `data:`, `file:`, `mailto:`, and protocol-
// relative URLs.
func isSafeArtifactURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	return parsed.Host != ""
}

func encodeArtifactData(data map[string]any) (datatypes.JSON, error) {
	if len(data) == 0 {
		return datatypes.JSON([]byte("{}")), nil
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(encoded), nil
}

func extractArtifactString(data map[string]any, key string) string {
	if len(data) == 0 {
		return ""
	}

	raw, ok := data[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func validateOptionalTimestamp(data map[string]any, key string) error {
	raw := extractArtifactString(data, key)
	if raw == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, raw); err != nil {
		return fmt.Errorf("%w: %s must be RFC3339 (got %q)", ErrFactoryWorkOrderArtifactInvalid, key, raw)
	}
	return nil
}

func extractPrLifecycleTimestamps(
	artifactType string,
	data map[string]any,
	now time.Time,
	existingMerged, existingClosed *time.Time,
	previousState string,
) (mergedAt *time.Time, closedAt *time.Time, err error) {
	if artifactType != FactoryWorkOrderArtifactTypePR {
		return nil, nil, nil
	}

	state := extractArtifactString(data, "state")

	if existingMerged == nil {
		mergedAt, err = stampPRLifecycleTime(data, "mergedAt", state, previousState, PrArtifactStateMerged, now)
		if err != nil {
			return nil, nil, err
		}
	}
	if existingClosed == nil {
		closedAt, err = stampPRLifecycleTime(data, "closedAt", state, previousState, PrArtifactStateClosed, now)
		if err != nil {
			return nil, nil, err
		}
	}
	return mergedAt, closedAt, nil
}

func stampPRLifecycleTime(
	data map[string]any,
	key, state, previousState, targetState string,
	now time.Time,
) (*time.Time, error) {
	if extractArtifactString(data, key) != "" {
		return pickTimestamp(data, key, now)
	}
	if state != targetState || previousState == targetState {
		return nil, nil
	}
	return pickTimestamp(data, key, now)
}

func readArtifactState(raw datatypes.JSON) string {
	if len(raw) == 0 {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	return extractArtifactString(data, "state")
}

func pickTimestamp(data map[string]any, key string, fallback time.Time) (*time.Time, error) {
	raw := extractArtifactString(data, key)
	if raw == "" {
		t := fallback
		return &t, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s must be RFC3339 (got %q)", ErrFactoryWorkOrderArtifactInvalid, key, raw)
	}
	return &parsed, nil
}
