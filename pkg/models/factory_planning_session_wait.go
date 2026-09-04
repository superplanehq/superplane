package models

import (
	"strings"

	"gorm.io/gorm"
)

func (s *FactoryPlanningSession) BeginWait(tx *gorm.DB) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		if s.WaitState == PlanningWaitPending || s.WaitState == PlanningWaitResolved {
			return nil
		}
		message, found, err := s.nextUndeliveredUserMessage(inner)
		if err != nil {
			return err
		}
		if found {
			refined, refineErr := s.applyRefineNote(inner, message.Text)
			if refineErr != nil {
				return refineErr
			}
			if err := s.deliverUserMessage(inner, message, refined); err != nil {
				return err
			}
			return s.saveSessionMutation(inner)
		}
		s.clearWait()
		s.WaitState = PlanningWaitPending
		return s.saveWait(inner)
	})
}

func (s *FactoryPlanningSession) ConsumeWait(tx *gorm.DB) (PlanningWaitResult, error) {
	var result PlanningWaitResult
	err := s.withLockedSession(tx, func(inner *gorm.DB) error {
		if s.WaitState != PlanningWaitResolved {
			return ErrFactoryPlanningWaitIdle
		}
		result = s.Wait()
		s.clearWait()
		return s.saveWait(inner)
	})
	return result, err
}

func (s *FactoryPlanningSession) RestoreWait(tx *gorm.DB, result PlanningWaitResult) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if s.WaitState == PlanningWaitResolved {
			return nil
		}
		if err := s.guardOpen(); err != nil {
			return err
		}
		_, found, err := s.nextUndeliveredUserMessage(inner)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		s.resolveWait(result)
		return s.saveWait(inner)
	})
}

func PlanningRefineNote(key, title string) string {
	return "Refine " + strings.TrimSpace(key) + ": " + strings.TrimSpace(title) + "."
}

func planningRefinePrompt(text string, draft PlanningSessionDraft) string {
	key := planningRefineKey(text)
	if key == "" {
		return text
	}
	var b strings.Builder
	b.WriteString("The user started refining task ")
	b.WriteString(key)
	b.WriteString(".\n\n")
	if title := strings.TrimSpace(draft.Title); title != "" {
		b.WriteString("Current title: ")
		b.WriteString(title)
		b.WriteByte('\n')
	}
	if description := strings.TrimSpace(draft.Description); description != "" {
		b.WriteString("Current description:\n")
		b.WriteString(description)
		b.WriteByte('\n')
	}
	b.WriteString("\nRead this task. Tell the user you are ready to refine it. Ask what they want to change. Do not call propose_draft. Do not change the draft. Then stop.")
	return b.String()
}

func planningRefineKey(text string) string {
	body := strings.TrimSpace(text)
	rest, ok := strings.CutPrefix(body, "Refine ")
	if !ok {
		return ""
	}
	key, remainder, ok := strings.Cut(rest, ": ")
	if !ok || !strings.HasSuffix(remainder, ".") {
		return ""
	}
	return strings.TrimSpace(key)
}

func planningWaitText(text string, refined bool, draft PlanningSessionDraft) string {
	if refined {
		return planningRefinePrompt(text, draft)
	}
	return text
}
