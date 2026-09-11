package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PlanningSessionMessage struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Role      string
	Text      string
	Delivered bool
	CreatedAt time.Time
}

func (PlanningSessionMessage) TableName() string {
	return "factory_planning_session_messages"
}

func sessionWaitText(tx *gorm.DB, session *FactoryPlanningSession, text string, refined bool) string {
	return planningWaitTextForKind(text, refined, session.Draft(), session.usesAnalysisFollowUp(tx))
}

func (s *FactoryPlanningSession) usesAnalysisFollowUp(tx *gorm.DB) bool {
	if s.CanvasID == nil || s.DraftWorkOrderID == nil {
		return false
	}
	var name string
	if err := tx.Model(&Canvas{}).Select("name").Where("id = ?", *s.CanvasID).Scan(&name).Error; err != nil || name == "" {
		return false
	}
	return name != PlanningCanvasName
}

func ListPlanningSessionMessages(tx *gorm.DB, sessionID uuid.UUID) ([]PlanningSessionMessage, error) {
	var messages []PlanningSessionMessage
	err := tx.Where("session_id = ?", sessionID).Order("created_at ASC, id ASC").Find(&messages).Error
	return messages, err
}

func (s *FactoryPlanningSession) ProposeSurvey(tx *gorm.DB, survey PlanningSessionSurvey) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		normalized, err := normalizePlanningSurvey(survey)
		if err != nil {
			return err
		}
		id := uuid.New()
		s.SurveyID = &id
		s.Survey = datatypes.NewJSONType(normalized)
		return s.saveSurvey(inner)
	})
}

func (s *FactoryPlanningSession) SendUserMessage(tx *gorm.DB, text string) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		if err := s.lockAndReload(inner); err != nil {
			return err
		}
		if err := s.guardOpen(); err != nil {
			return err
		}
		body := strings.TrimSpace(text)
		if body == "" {
			return fmt.Errorf("%w: message is required", ErrFactoryPlanningSessionInvalid)
		}
		message := PlanningSessionMessage{
			ID:        uuid.New(),
			SessionID: s.ID,
			Role:      PlanningSessionMessageRoleUser,
			Text:      body,
			CreatedAt: time.Now(),
		}
		s.clearSurvey()
		refined, err := s.applyRefineNote(inner, body)
		if err != nil {
			return err
		}
		if s.WaitState == PlanningWaitPending {
			message.Delivered = true
			s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindMessage, Text: sessionWaitText(inner, s, body, refined)})
		}
		if err := inner.Create(&message).Error; err != nil {
			return err
		}
		if err := s.saveSessionMutation(inner); err != nil {
			return err
		}
		return s.reloadMessages(inner)
	})
}

func (s *FactoryPlanningSession) reloadMessages(tx *gorm.DB) error {
	messages, err := ListPlanningSessionMessages(tx, s.ID)
	if err != nil {
		return err
	}
	s.Messages = messages
	return nil
}

func (s *FactoryPlanningSession) nextUndeliveredUserMessage(tx *gorm.DB) (PlanningSessionMessage, bool, error) {
	var message PlanningSessionMessage
	err := tx.
		Where("session_id = ? AND role = ? AND delivered = ?", s.ID, PlanningSessionMessageRoleUser, false).
		Order("created_at ASC, id ASC").
		First(&message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PlanningSessionMessage{}, false, nil
	}
	if err != nil {
		return PlanningSessionMessage{}, false, err
	}
	return message, true, nil
}

func (s *FactoryPlanningSession) deliverUserMessage(tx *gorm.DB, message PlanningSessionMessage, refined bool) error {
	if err := tx.Model(&message).Select("Delivered").Updates(PlanningSessionMessage{Delivered: true}).Error; err != nil {
		return err
	}
	s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindMessage, Text: sessionWaitText(tx, s, message.Text, refined)})
	return nil
}

func normalizePlanningSurvey(survey PlanningSessionSurvey) (PlanningSessionSurvey, error) {
	if len(survey.Questions) == 0 {
		return PlanningSessionSurvey{}, fmt.Errorf("%w: survey needs a question", ErrFactoryPlanningSessionInvalid)
	}
	if len(survey.Questions) > maxPlanningSurveyQuestions {
		return PlanningSessionSurvey{}, fmt.Errorf("%w: survey has too many questions", ErrFactoryPlanningSessionInvalid)
	}
	questions := make([]PlanningSessionSurveyQuestion, 0, len(survey.Questions))
	for _, question := range survey.Questions {
		prompt := strings.TrimSpace(question.Prompt)
		if prompt == "" {
			return PlanningSessionSurvey{}, fmt.Errorf("%w: survey question is required", ErrFactoryPlanningSessionInvalid)
		}
		options := make([]string, 0, len(question.Options))
		for _, option := range question.Options {
			option = strings.TrimSpace(option)
			if option == "" {
				continue
			}
			options = append(options, option)
			if len(options) == maxPlanningSurveyOptions {
				break
			}
		}
		if len(options) == 0 {
			return PlanningSessionSurvey{}, fmt.Errorf("%w: survey question needs an option", ErrFactoryPlanningSessionInvalid)
		}
		questions = append(questions, PlanningSessionSurveyQuestion{Prompt: prompt, Options: options})
	}
	return PlanningSessionSurvey{Questions: questions}, nil
}
