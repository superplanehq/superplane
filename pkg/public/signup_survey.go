package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm/clause"
)

const useCaseMaxLen = 500

type signupSurveyRequest struct {
	Skipped       bool    `json:"skipped"`
	SourceChannel *string `json:"source_channel,omitempty"`
	SourceOther   *string `json:"source_other,omitempty"`
	Role          *string `json:"role,omitempty"`
	UseCase       *string `json:"use_case,omitempty"`
}

func (s *Server) submitSignupSurvey(w http.ResponseWriter, r *http.Request) {
	if !config.IsSignupSurveyEnabled() {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	account, ok := middleware.GetEffectiveAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	var req signupSurveyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	input, err := buildSurveyInput(account.ID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Idempotency: account_survey_responses has UNIQUE(account_id), so a
	// repeated submit is a no-op. No transaction needed — a single row
	// per account is the only state we care about.
	row := &models.AccountSurveyResponse{
		AccountID:     input.AccountID,
		SurveyType:    input.SurveyType,
		Skipped:       input.Skipped,
		SourceChannel: input.SourceChannel,
		SourceOther:   input.SourceOther,
		Role:          input.Role,
		UseCase:       input.UseCase,
	}
	if err := database.Conn().Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
		log.Errorf("submitSignupSurvey: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func buildSurveyInput(accountID uuid.UUID, req *signupSurveyRequest) (*models.AccountSurveyResponseInput, error) {
	input := &models.AccountSurveyResponseInput{
		AccountID:  accountID,
		SurveyType: models.SurveyTypeSignup,
		Skipped:    req.Skipped,
	}

	if req.Skipped {
		if nonEmpty(req.SourceChannel) || nonEmpty(req.SourceOther) ||
			nonEmpty(req.Role) || nonEmpty(req.UseCase) {
			return nil, errors.New("skipped survey must not include response fields")
		}
		return input, nil
	}

	if !nonEmpty(req.SourceChannel) {
		return nil, errors.New("source_channel is required when not skipped")
	}
	if !models.IsValidSourceChannel(*req.SourceChannel) {
		return nil, errors.New("invalid source_channel")
	}
	input.SourceChannel = req.SourceChannel

	if *req.SourceChannel == models.SourceChannelOther && nonEmpty(req.SourceOther) {
		trimmed := strings.TrimSpace(*req.SourceOther)
		if len(trimmed) > useCaseMaxLen {
			trimmed = trimmed[:useCaseMaxLen]
		}
		input.SourceOther = &trimmed
	}

	if nonEmpty(req.Role) {
		if !models.IsValidRole(*req.Role) {
			return nil, errors.New("invalid role")
		}
		input.Role = req.Role
	}

	if nonEmpty(req.UseCase) {
		trimmed := strings.TrimSpace(*req.UseCase)
		if len(trimmed) > useCaseMaxLen {
			trimmed = trimmed[:useCaseMaxLen]
		}
		input.UseCase = &trimmed
	}

	return input, nil
}

func nonEmpty(s *string) bool {
	return s != nil && strings.TrimSpace(*s) != ""
}
