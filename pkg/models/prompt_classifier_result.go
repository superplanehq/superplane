package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ClassifierStatusPending   = "pending"
	ClassifierStatusRunning   = "running"
	ClassifierStatusCompleted = "completed"
	ClassifierStatusFailed    = "failed"
	ClassifierStatusSkipped   = "skipped"
)

type PromptClassifierResult struct {
	ID            uuid.UUID `gorm:"primaryKey;default:gen_random_uuid()"`
	ScanResultID  uuid.UUID `gorm:"not null"`

	Status string `gorm:"not null;default:'pending'"`

	SubmittedAt  time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time

	ClassifierModel   *string
	ClassifierVersion *string

	RiskScore   *int
	Findings    datatypes.JSONType[[]GuardrailFinding]
	RawResponse *string
	TokenCount  *int
	LatencyMs   *int

	ErrorCode    *string
	ErrorMessage *string
	RetryCount   int `gorm:"not null;default:0"`
}

func (r *PromptClassifierResult) TableName() string {
	return "prompt_classifier_results"
}

func CreatePromptClassifierResult(result *PromptClassifierResult) error {
	return CreatePromptClassifierResultInTransaction(database.Conn(), result)
}

func CreatePromptClassifierResultInTransaction(tx *gorm.DB, result *PromptClassifierResult) error {
	return tx.Create(result).Error
}

func FindPendingClassifierJobs(limit int) ([]PromptClassifierResult, error) {
	var results []PromptClassifierResult
	err := database.Conn().
		Where("status = ?", ClassifierStatusPending).
		Order("submitted_at ASC").
		Limit(limit).
		Find(&results).
		Error
	if err != nil {
		return nil, err
	}

	return results, nil
}
