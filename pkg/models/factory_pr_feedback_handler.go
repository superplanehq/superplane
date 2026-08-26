package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryPRFeedbackHandlerSourceGitHubPullRequests = "github-pull-requests"

	factoryPRFeedbackHandlerCanvasUniqueConstraint = "idx_factory_pr_feedback_handlers_canvas_id"
)

var (
	ErrFactoryPRFeedbackHandlerNotFound       = errors.New("factory PR feedback handler not found")
	ErrFactoryPRFeedbackHandlerCanvasInUse    = errors.New("canvas already implements a factory PR feedback handler")
	ErrFactoryPRFeedbackHandlerSourceInvalid  = errors.New("factory PR feedback handler source is not valid")
	ErrFactoryPRFeedbackHandlerCanvasRequired = errors.New("factory PR feedback handler canvas is required")
)

var factoryPRFeedbackHandlerSources = []string{
	FactoryPRFeedbackHandlerSourceGitHubPullRequests,
}

// FactoryPRFeedbackHandler declares that a factory canvas addresses pull
// request feedback. The row owns identity; the canvas graph owns behavior.
type FactoryPRFeedbackHandler struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	CanvasID       uuid.UUID
	Source         string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Canvas *Canvas `gorm:"foreignKey:CanvasID"`
}

func ValidFactoryPRFeedbackHandlerSource(source string) bool {
	return slices.Contains(factoryPRFeedbackHandlerSources, source)
}

func (FactoryPRFeedbackHandler) TableName() string {
	return "factory_pr_feedback_handlers"
}

// Name is the handler's display name, which is the canvas name.
func (h *FactoryPRFeedbackHandler) Name() string {
	if h.Canvas == nil {
		return ""
	}
	return h.Canvas.Name
}

func MapFactoryPRFeedbackHandlerCanvasUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryPRFeedbackHandlerCanvasUniqueConstraint {
		return ErrFactoryPRFeedbackHandlerCanvasInUse
	}

	return err
}

func (f *Factory) CreatePRFeedbackHandler(tx *gorm.DB, canvasID uuid.UUID, source string) (*FactoryPRFeedbackHandler, error) {
	if canvasID == uuid.Nil {
		return nil, ErrFactoryPRFeedbackHandlerCanvasRequired
	}
	if !ValidFactoryPRFeedbackHandlerSource(source) {
		return nil, ErrFactoryPRFeedbackHandlerSourceInvalid
	}

	now := time.Now()
	handler := &FactoryPRFeedbackHandler{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		CanvasID:       canvasID,
		Source:         source,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(handler).Error; err != nil {
		return nil, MapFactoryPRFeedbackHandlerCanvasUniqueConstraintError(err)
	}

	return handler, nil
}

func (f *Factory) FindPRFeedbackHandler(tx *gorm.DB, handlerID uuid.UUID) (*FactoryPRFeedbackHandler, error) {
	var handler FactoryPRFeedbackHandler
	err := liveCanvasPRFeedbackHandlers(tx).
		Where("factory_pr_feedback_handlers.organization_id = ? AND factory_pr_feedback_handlers.factory_id = ? AND factory_pr_feedback_handlers.id = ?", f.OrganizationID, f.ID, handlerID).
		First(&handler).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryPRFeedbackHandlerNotFound
		}
		return nil, err
	}

	return &handler, nil
}

func (f *Factory) ListPRFeedbackHandlers(tx *gorm.DB) ([]FactoryPRFeedbackHandler, error) {
	var handlers []FactoryPRFeedbackHandler
	err := liveCanvasPRFeedbackHandlers(tx).
		Where("factory_pr_feedback_handlers.organization_id = ? AND factory_pr_feedback_handlers.factory_id = ?", f.OrganizationID, f.ID).
		Order("factory_pr_feedback_handlers.created_at ASC").
		Order("factory_pr_feedback_handlers.id ASC").
		Find(&handlers).
		Error
	if err != nil {
		return nil, err
	}

	return handlers, nil
}

func (h *FactoryPRFeedbackHandler) Delete(tx *gorm.DB) error {
	return tx.Where("id = ?", h.ID).Delete(&FactoryPRFeedbackHandler{}).Error
}

func (h *FactoryPRFeedbackHandler) Touch(tx *gorm.DB) error {
	now := time.Now()
	if err := tx.Model(h).Update("updated_at", now).Error; err != nil {
		return err
	}
	h.UpdatedAt = now
	return nil
}

// DeleteFactoryPRFeedbackHandlersByCanvas removes the handlers a canvas
// implements. The canvas reference is a RESTRICT foreign key, so canvas
// deletion has to call this before the canvas row goes away.
func DeleteFactoryPRFeedbackHandlersByCanvas(tx *gorm.DB, canvasID uuid.UUID) error {
	return tx.Where("canvas_id = ?", canvasID).Delete(&FactoryPRFeedbackHandler{}).Error
}

func liveCanvasPRFeedbackHandlers(tx *gorm.DB) *gorm.DB {
	return tx.
		Joins("JOIN workflows ON workflows.id = factory_pr_feedback_handlers.canvas_id AND workflows.deleted_at IS NULL").
		Preload("Canvas")
}
