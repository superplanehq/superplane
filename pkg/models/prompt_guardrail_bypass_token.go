package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PromptGuardrailBypassToken struct {
	ID         uuid.UUID  `gorm:"primaryKey;default:gen_random_uuid()"`
	Token      string     `gorm:"not null;uniqueIndex"`
	OrgID      uuid.UUID  `gorm:"not null"`
	WorkflowID *uuid.UUID
	NodeID     *string
	Rules      datatypes.JSONType[[]string]
	IssuedBy   uuid.UUID `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	UsageLimit int       `gorm:"not null;default:1"`
	UsageCount int       `gorm:"not null;default:0"`
	CreatedAt  time.Time
}

func (t *PromptGuardrailBypassToken) TableName() string {
	return "prompt_guardrail_bypass_tokens"
}

func (t *PromptGuardrailBypassToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *PromptGuardrailBypassToken) IsExhausted() bool {
	return t.UsageCount >= t.UsageLimit
}

func (t *PromptGuardrailBypassToken) IsValid() bool {
	return !t.IsExpired() && !t.IsExhausted()
}

func FindBypassTokenByValue(token string) (*PromptGuardrailBypassToken, error) {
	return FindBypassTokenByValueInTransaction(database.Conn(), token)
}

func FindBypassTokenByValueInTransaction(tx *gorm.DB, token string) (*PromptGuardrailBypassToken, error) {
	var bt PromptGuardrailBypassToken
	err := tx.
		Where("token = ?", token).
		First(&bt).
		Error
	if err != nil {
		return nil, err
	}

	return &bt, nil
}

func IncrementBypassTokenUsage(tx *gorm.DB, id uuid.UUID) error {
	return tx.Model(&PromptGuardrailBypassToken{}).
		Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).
		Error
}
