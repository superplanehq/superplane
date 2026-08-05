package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderArtifactTypePR       = factory.ArtifactTypePR
	FactoryWorkOrderArtifactTypeMarkdown = factory.ArtifactTypeMarkdown
)

var (
	ErrFactoryWorkOrderArtifactNotFound = errors.New("factory work order artifact not found")
	ErrFactoryWorkOrderArtifactInvalid  = errors.New("invalid work order artifact")
)

type FactoryWorkOrderArtifact struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Type           string
	URL            string
	Title          string
	Body           string
	Data           datatypes.JSON
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time

	CreatedBy *User `gorm:"foreignKey:CreatedByID"`
}

func (FactoryWorkOrderArtifact) TableName() string {
	return "factory_work_order_artifacts"
}

// FactoryWorkOrderArtifactParams collects everything callers can populate on a
// new artifact. Only Type and content required by that type are enforced.
type FactoryWorkOrderArtifactParams struct {
	Type      string
	URL       string
	Title     string
	Body      string
	Data      map[string]any
	CreatedBy *uuid.UUID
	Run       *factory.RunRef
}

// CreateArtifact appends a new artifact to the work order and records the
// matching `order.artifact.added` event in the same transaction so the
// timeline stays consistent with the artifact list.
func (o *FactoryWorkOrder) CreateArtifact(
	db *gorm.DB,
	params FactoryWorkOrderArtifactParams,
) (*FactoryWorkOrderArtifact, error) {
	artifactType := strings.TrimSpace(params.Type)
	url := strings.TrimSpace(params.URL)
	title := strings.TrimSpace(params.Title)
	body := params.Body

	switch artifactType {
	case FactoryWorkOrderArtifactTypePR:
		if url == "" {
			return nil, fmt.Errorf("%w: pull request artifacts require a url", ErrFactoryWorkOrderArtifactInvalid)
		}
	case FactoryWorkOrderArtifactTypeMarkdown:
		if strings.TrimSpace(body) == "" {
			return nil, fmt.Errorf("%w: markdown artifacts require a body", ErrFactoryWorkOrderArtifactInvalid)
		}
	default:
		return nil, fmt.Errorf("%w: unknown artifact type %q", ErrFactoryWorkOrderArtifactInvalid, params.Type)
	}

	//
	// PR links round-trip straight into the timeline / sidebar `href`. Only
	// http(s) is allowed so anyone with `factories:update` cannot smuggle a
	// `javascript:` or other scheme into a link teammates will click.
	//
	if url != "" && !isSafeArtifactURL(url) {
		return nil, fmt.Errorf("%w: artifact url must be http(s)", ErrFactoryWorkOrderArtifactInvalid)
	}

	dataJSON, err := encodeArtifactData(params.Data)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	artifact := &FactoryWorkOrderArtifact{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		Type:           artifactType,
		URL:            url,
		Title:          title,
		Body:           body,
		Data:           dataJSON,
		CreatedByID:    params.CreatedBy,
		CreatedAt:      now,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Returning{}).Create(artifact).Error; err != nil {
			return err
		}

		ref := &factory.ArtifactRef{
			ID:    artifact.ID,
			Type:  artifact.Type,
			URL:   artifact.URL,
			Title: artifact.Title,
		}
		if artifact.Type == FactoryWorkOrderArtifactTypeMarkdown {
			ref.Body = artifact.Body
		}

		return o.RecordArtifactAdded(tx, ref, params.CreatedBy, params.Run)
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

// IsValidWorkOrderArtifactType reports whether the given type is currently
// accepted by CreateArtifact / API handlers.
func IsValidWorkOrderArtifactType(t string) bool {
	switch t {
	case FactoryWorkOrderArtifactTypePR, FactoryWorkOrderArtifactTypeMarkdown:
		return true
	}
	return false
}

// isSafeArtifactURL reports whether the given URL is an absolute http(s)
// URL with a host. It rejects `javascript:`, `data:`, `file:`, `mailto:`,
// protocol-relative URLs, and anything without a parseable scheme + host.
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
