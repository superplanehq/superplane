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
	FactoryWorkOrderArtifactTypeMarkdown = factory.ArtifactTypeMarkdown
	FactoryWorkOrderArtifactTypeBranch   = factory.ArtifactTypeBranch
	FactoryWorkOrderArtifactTypeLink     = factory.ArtifactTypeLink
	FactoryWorkOrderArtifactTypeFile     = factory.ArtifactTypeFile

	// MaxFactoryWorkOrderArtifactDataBytes caps JSON-encoded artifact data.
	MaxFactoryWorkOrderArtifactDataBytes = 64 * 1024

	// MaxFactoryWorkOrderArtifactKeyBytes matches the `key` column width.
	MaxFactoryWorkOrderArtifactKeyBytes = 512
)

const factoryWorkOrderArtifactKeyUniqueConstraint = "idx_factory_work_order_artifacts_factory_key_unique"

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
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time

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

	dataJSON, err := encodeGuardedArtifactData(params.Data)
	if err != nil {
		return nil, err
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
	artifact := &FactoryWorkOrderArtifact{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           artifactType,
		Data:           dataJSON,
		Key:            key,
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

// UpsertArtifact creates the artifact, or replaces its data when a key
// already points at one on this work order. The bool is true when a new
// row was inserted. An empty key always inserts.
func (o *FactoryWorkOrder) UpsertArtifact(
	db *gorm.DB,
	params FactoryWorkOrderArtifactParams,
) (*FactoryWorkOrderArtifact, bool, error) {
	if strings.TrimSpace(params.Key) == "" {
		artifact, err := o.CreateArtifact(db, params)
		return artifact, true, err
	}

	var (
		artifact *FactoryWorkOrderArtifact
		created  bool
	)
	err := db.Transaction(func(tx *gorm.DB) error {
		existing, findErr := o.FindArtifactByKey(tx, params.Key)
		if findErr == nil {
			replaced, replaceErr := o.replaceArtifactData(tx, existing, params)
			if replaceErr != nil {
				return replaceErr
			}
			artifact = replaced
			created = false
			return nil
		}
		if !errors.Is(findErr, ErrFactoryWorkOrderArtifactNotFound) {
			return findErr
		}

		createdArtifact, createErr := o.CreateArtifact(tx, params)
		if createErr == nil {
			artifact = createdArtifact
			created = true
			return nil
		}
		if !errors.Is(createErr, ErrFactoryWorkOrderArtifactKeyAlreadyExists) {
			return createErr
		}

		replaced, retryErr := o.replaceKeyedArtifactAfterConflict(tx, params)
		if retryErr != nil {
			return retryErr
		}
		artifact = replaced
		created = false
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	return artifact, created, nil
}

// UpdateArtifactData resolves the artifact tagged with `key` under this
// work order (the same key an earlier CreateArtifact call set via
// FactoryWorkOrderArtifactParams.Key, typically the PR's URL), shallow-
// merges `updates` into its existing `data` map, and re-saves it in
// place. Unlike CreateArtifact, this does not append a new
// `order.artifact.*` timeline event — a PR flipping open → draft →
// merged should update the live chip, not spam the timeline with one
// entry per transition. Callers still notify the websocket channel
// (see FactoryContext websocket notify) so the UI refreshes.
func (o *FactoryWorkOrder) FindArtifactByKey(tx *gorm.DB, key string) (*FactoryWorkOrderArtifact, error) {
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
	return &artifact, nil
}

func (o *FactoryWorkOrder) UpdateArtifactData(
	tx *gorm.DB,
	key string,
	updates map[string]any,
) (*FactoryWorkOrderArtifact, error) {
	artifact, err := o.FindArtifactByKey(tx, key)
	if err != nil {
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

	dataJSON, err := encodeGuardedArtifactData(merged)
	if err != nil {
		return nil, err
	}

	if err := tx.Model(artifact).Update("data", dataJSON).Error; err != nil {
		return nil, err
	}
	artifact.Data = dataJSON

	return artifact, nil
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

// DeleteArtifacts removes every artifact row for this order and records
// one timeline event. It does not delete git branches.
func (o *FactoryWorkOrder) DeleteArtifacts(tx *gorm.DB, actor *uuid.UUID) (int, error) {
	artifacts, err := o.ListArtifacts(tx)
	if err != nil {
		return 0, err
	}
	if len(artifacts) == 0 {
		return 0, nil
	}

	err = tx.Where("work_order_id = ?", o.ID).Delete(&FactoryWorkOrderArtifact{}).Error
	if err != nil {
		return 0, err
	}

	if err := o.RecordArtifactsCleared(tx, len(artifacts), actor); err != nil {
		return 0, err
	}

	return len(artifacts), nil
}

// IsValidWorkOrderArtifactType reports whether CreateArtifact accepts t.
func IsValidWorkOrderArtifactType(t string) bool {
	switch t {
	case FactoryWorkOrderArtifactTypeMarkdown, FactoryWorkOrderArtifactTypeBranch, FactoryWorkOrderArtifactTypeLink:
		return true
	}
	return false
}

func (o *FactoryWorkOrder) replaceKeyedArtifactAfterConflict(
	tx *gorm.DB,
	params FactoryWorkOrderArtifactParams,
) (*FactoryWorkOrderArtifact, error) {
	existing, err := findFactoryWorkOrderArtifactByKey(tx, o.OrganizationID, o.FactoryID, params.Key)
	if err != nil {
		if errors.Is(err, ErrFactoryWorkOrderArtifactNotFound) {
			return nil, ErrFactoryWorkOrderArtifactKeyAlreadyExists
		}
		return nil, err
	}
	if existing.WorkOrderID != o.ID {
		return nil, ErrFactoryWorkOrderArtifactKeyAlreadyExists
	}

	return o.replaceArtifactData(tx, existing, params)
}

func (o *FactoryWorkOrder) replaceArtifactData(
	tx *gorm.DB,
	artifact *FactoryWorkOrderArtifact,
	params FactoryWorkOrderArtifactParams,
) (*FactoryWorkOrderArtifact, error) {
	requestedType := strings.TrimSpace(params.Type)
	if artifact.Type != requestedType {
		return nil, fmt.Errorf(
			"%w: stored type is %q, requested type is %q",
			ErrFactoryWorkOrderArtifactInvalid,
			artifact.Type,
			requestedType,
		)
	}

	if err := validateArtifactData(artifact.Type, params.Data); err != nil {
		return nil, err
	}

	dataJSON, err := encodeGuardedArtifactData(params.Data)
	if err != nil {
		return nil, err
	}

	if err := tx.Model(artifact).Update("data", dataJSON).Error; err != nil {
		return nil, err
	}
	artifact.Data = dataJSON

	return artifact, nil
}

func findFactoryWorkOrderArtifactByKey(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	key string,
) (*FactoryWorkOrderArtifact, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return nil, fmt.Errorf("%w: artifact key is required", ErrFactoryWorkOrderArtifactInvalid)
	}

	var artifact FactoryWorkOrderArtifact
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND key = ?", organizationID, factoryID, trimmedKey).
		First(&artifact).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderArtifactNotFound
		}
		return nil, err
	}

	return &artifact, nil
}

// validateArtifactData enforces the required-field rules for each
// artifact type (shared by CreateArtifact and UpdateArtifactData so a
// later merge can't drift from what a fresh attach requires) plus the
// URL-scheme guard that applies regardless of type.
func validateArtifactData(artifactType string, data map[string]any) error {
	switch artifactType {
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
	case FactoryWorkOrderArtifactTypeFile:
		if extractArtifactString(data, "fileId") == "" ||
			extractArtifactString(data, "filename") == "" ||
			extractArtifactString(data, "contentType") == "" ||
			extractArtifactString(data, "title") == "" ||
			extractArtifactString(data, "url") == "" {
			return fmt.Errorf("%w: file artifacts require fileId, filename, contentType, title, and url", ErrFactoryWorkOrderArtifactInvalid)
		}
		if _, err := uuid.Parse(extractArtifactString(data, "fileId")); err != nil {
			return fmt.Errorf("%w: file artifacts require a valid fileId", ErrFactoryWorkOrderArtifactInvalid)
		}
		if !IsAllowedArtifactContentType(extractArtifactString(data, "contentType")) {
			return fmt.Errorf("%w: file artifact content type is not supported", ErrFactoryWorkOrderArtifactInvalid)
		}
		if size, ok := extractArtifactSize(data); !ok || size <= 0 || size > int64(MaxArtifactFileBytes) {
			return fmt.Errorf("%w: file artifacts require a valid sizeBytes", ErrFactoryWorkOrderArtifactInvalid)
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

func encodeGuardedArtifactData(data map[string]any) (datatypes.JSON, error) {
	dataJSON, err := encodeArtifactData(data)
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

	return dataJSON, nil
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

func extractArtifactSize(data map[string]any) (int64, bool) {
	switch value := data["sizeBytes"].(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		if value != float64(int64(value)) {
			return 0, false
		}
		return int64(value), true
	case json.Number:
		size, err := value.Int64()
		return size, err == nil
	default:
		return 0, false
	}
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
