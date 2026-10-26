package models

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"gorm.io/gorm"
)

const (
	PlanningComplexityCheckKey     = "complexity"
	PlanningComplexityCheckName    = "Complexity"
	PlanningVerifiabilityCheckKey  = "verifiability"
	PlanningVerifiabilityCheckName = "Verifiability"

	// Review sub-parameters use a 1 through 3 scale: 3 is good, 2 is
	// partial, 1 is bad. The legacy scores keep PlanningScoreMax.
	PlanningReviewScoreMax = 3
)

var planningReviewScoreKinds = []planningScoreKind{
	{key: PlanningClarityCheckKey, name: PlanningClarityCheckName, max: PlanningReviewScoreMax},
	{key: PlanningComplexityCheckKey, name: PlanningComplexityCheckName, max: PlanningReviewScoreMax},
	{key: PlanningVerifiabilityCheckKey, name: PlanningVerifiabilityCheckName, max: PlanningReviewScoreMax},
}

// PlanningScoreValue is one 1 through 3 sub-parameter with a one-sentence summary.
type PlanningScoreValue struct {
	Score   float64
	Summary string
}

// PlanningReviewScores is the full set of review sub-parameters. All three
// must be present when the group is sent.
type PlanningReviewScores struct {
	Clarity       PlanningScoreValue
	Complexity    PlanningScoreValue
	Verifiability PlanningScoreValue
}

// PlanningSessionUpdate is one atomic plan-turn publish: scores, spec, and
// an optional survey. At least one field is required.
type PlanningSessionUpdate struct {
	Scores *PlanningReviewScores
	Spec   string
	Survey *PlanningSessionSurvey
}

func (u PlanningSessionUpdate) hasField() bool {
	return u.Scores != nil || strings.TrimSpace(u.Spec) != "" || u.Survey != nil
}

func (s *PlanningReviewScores) value(key string) PlanningScoreValue {
	switch key {
	case PlanningClarityCheckKey:
		return s.Clarity
	case PlanningComplexityCheckKey:
		return s.Complexity
	case PlanningVerifiabilityCheckKey:
		return s.Verifiability
	default:
		return PlanningScoreValue{}
	}
}

// ProposeUpdate applies scores, spec, and survey in one transaction.
func (s *FactoryPlanningSession) ProposeUpdate(tx *gorm.DB, update PlanningSessionUpdate) error {
	if !update.hasField() {
		return fmt.Errorf("%w: scores, spec, or survey is required", ErrFactoryPlanningSessionInvalid)
	}
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		order, err := s.analysisWorkOrder(inner)
		if err != nil {
			return err
		}
		checks, err := order.ListChecks(inner)
		if err != nil {
			return err
		}
		if err := validatePlanningSessionUpdate(update, checks); err != nil {
			return err
		}
		if update.Scores != nil {
			for _, kind := range planningReviewScoreKinds {
				value := update.Scores.value(kind.key)
				if err := reportPlanningScore(inner, s, order, kind, value.Score, value.Summary); err != nil {
					return err
				}
			}
		}
		if spec := unwrapPlanningMarkdown(update.Spec); spec != "" {
			if err := s.ProposeSpec(inner, spec); err != nil {
				return err
			}
		}
		if update.Survey == nil {
			return nil
		}
		return s.ProposeSurvey(inner, *update.Survey)
	})
}

// validatePlanningSessionUpdate checks one propose_update call. Review
// scoring is not optional: the first plan turn must publish all three
// sub-parameters, and any scores sent later must be complete and valid.
func validatePlanningSessionUpdate(update PlanningSessionUpdate, checks []FactoryWorkOrderCheck) error {
	if update.Scores != nil {
		if err := validatePlanningReviewScores(*update.Scores); err != nil {
			return err
		}
	} else if !hasPlanningReviewScores(checks) {
		return fmt.Errorf("%w: the first plan turn must include scores", ErrFactoryPlanningSessionInvalid)
	}
	if update.Survey != nil && planningReviewAtMaximum(update.Scores, checks) {
		return fmt.Errorf("%w: a survey cannot be published when all review scores are 3; remove the survey or lower Clarity if a user decision is needed", ErrFactoryPlanningSessionInvalid)
	}
	return nil
}

func validatePlanningReviewScores(scores PlanningReviewScores) error {
	for _, kind := range planningReviewScoreKinds {
		value := scores.value(kind.key)
		if err := validatePlanningReviewScore(kind, value); err != nil {
			return err
		}
	}
	return nil
}

func planningReviewAtMaximum(scores *PlanningReviewScores, checks []FactoryWorkOrderCheck) bool {
	for _, kind := range planningReviewScoreKinds {
		if scores != nil {
			if scores.value(kind.key).Score != kind.max {
				return false
			}
			continue
		}
		if !slices.ContainsFunc(checks, func(check FactoryWorkOrderCheck) bool {
			return check.Key == kind.key && check.Score == kind.max
		}) {
			return false
		}
	}
	return true
}

func validatePlanningReviewScore(kind planningScoreKind, value PlanningScoreValue) error {
	if !isFiniteCheckNumber(value.Score) || math.Trunc(value.Score) != value.Score || value.Score < 1 || value.Score > PlanningReviewScoreMax {
		return fmt.Errorf("%w: %s score must be an integer from 1 through 3", ErrFactoryPlanningSessionInvalid, kind.key)
	}
	if strings.TrimSpace(value.Summary) == "" {
		return fmt.Errorf("%w: %s summary is required", ErrFactoryPlanningSessionInvalid, kind.key)
	}
	return nil
}

func hasPlanningReviewScores(checks []FactoryWorkOrderCheck) bool {
	found := 0
	for _, kind := range planningReviewScoreKinds {
		for i := range checks {
			if checks[i].Key == kind.key {
				found++
				break
			}
		}
	}
	return found == len(planningReviewScoreKinds)
}
