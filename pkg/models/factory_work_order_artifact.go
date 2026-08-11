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

	switch artifactType {
	case FactoryWorkOrderArtifactTypePR:
		if extractArtifactString(params.Data, "url") == "" {
			return nil, fmt.Errorf("%w: pull request artifacts require a url", ErrFactoryWorkOrderArtifactInvalid)
		}
	case FactoryWorkOrderArtifactTypeMarkdown:
		if extractArtifactString(params.Data, "body") == "" {
			return nil, fmt.Errorf("%w: markdown artifacts require data.body", ErrFactoryWorkOrderArtifactInvalid)
		}
	case FactoryWorkOrderArtifactTypeBranch:
		if extractArtifactString(params.Data, "name") == "" {
			return nil, fmt.Errorf("%w: branch artifacts require data.name", ErrFactoryWorkOrderArtifactInvalid)
		}
	default:
		return nil, fmt.Errorf("%w: unknown artifact type %q", ErrFactoryWorkOrderArtifactInvalid, params.Type)
	}

	// `data.url` lands in a clickable `href` for every artifact type the
	// UI knows about (extractArtifactUrl reads it unconditionally), so
	// reject non-http(s) schemes here rather than only inside the PR
	// branch — no caller should be able to smuggle `javascript:` past
	// the model.
	if artifactURL := extractArtifactString(params.Data, "url"); artifactURL != "" {
		if !isSafeArtifactURL(artifactURL) {
			return nil, fmt.Errorf("%w: artifact url must be http(s)", ErrFactoryWorkOrderArtifactInvalid)
		}
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

// IsValidWorkOrderArtifactType reports whether CreateArtifact accepts t.
func IsValidWorkOrderArtifactType(t string) bool {
	switch t {
	case FactoryWorkOrderArtifactTypePR, FactoryWorkOrderArtifactTypeMarkdown, FactoryWorkOrderArtifactTypeBranch:
		return true
	}
	return false
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
