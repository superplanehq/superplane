package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const SurveyTypeSignup = "signup"

const (
	SourceChannelSearch   = "search"
	SourceChannelSocial   = "social"
	SourceChannelReferral = "referral"
	SourceChannelContent  = "content"
	SourceChannelEvent    = "event"
	SourceChannelPartner  = "partner"
	SourceChannelOther    = "other"
)

// Role enum — values stored in role column.
const (
	RoleEngineer = "engineer"
	RoleDevOps   = "devops"
	RoleManager  = "manager"
	RoleFounder  = "founder"
	RoleProduct  = "product"
	RoleOther    = "other"
)

var validSourceChannels = map[string]struct{}{
	SourceChannelSearch:   {},
	SourceChannelSocial:   {},
	SourceChannelReferral: {},
	SourceChannelContent:  {},
	SourceChannelEvent:    {},
	SourceChannelPartner:  {},
	SourceChannelOther:    {},
}

var validRoles = map[string]struct{}{
	RoleEngineer: {},
	RoleDevOps:   {},
	RoleManager:  {},
	RoleFounder:  {},
	RoleProduct:  {},
	RoleOther:    {},
}

func IsValidSourceChannel(v string) bool {
	_, ok := validSourceChannels[v]
	return ok
}

func IsValidRole(v string) bool {
	_, ok := validRoles[v]
	return ok
}

type AccountSurveyResponse struct {
	ID            uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	AccountID     uuid.UUID
	SurveyType    string
	Skipped       bool
	SourceChannel *string
	SourceOther   *string
	Role          *string
	UseCase       *string
	CreatedAt     *time.Time
}

func (AccountSurveyResponse) TableName() string { return "account_survey_responses" }

type AccountSurveyResponseInput struct {
	AccountID     uuid.UUID
	SurveyType    string
	Skipped       bool
	SourceChannel *string
	SourceOther   *string
	Role          *string
	UseCase       *string
}

func CreateAccountSurveyResponseInTransaction(tx *gorm.DB, in AccountSurveyResponseInput) (*AccountSurveyResponse, error) {
	row := &AccountSurveyResponse{
		AccountID:     in.AccountID,
		SurveyType:    in.SurveyType,
		Skipped:       in.Skipped,
		SourceChannel: in.SourceChannel,
		SourceOther:   in.SourceOther,
		Role:          in.Role,
		UseCase:       in.UseCase,
	}
	if err := tx.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}
