package models

import (
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/gorm"
)

const (
	PlanningComplexityCheckKey  = "complexity"
	PlanningComplexityCheckName = "Complexity"
	PlanningRiskCheckKey        = "risk"
	PlanningRiskCheckName       = "Risk"
)

var planningReviewScoreKinds = []planningScoreKind{
	{key: PlanningClarityCheckKey, name: PlanningClarityCheckName},
	{key: PlanningComplexityCheckKey, name: PlanningComplexityCheckName},
	{key: PlanningRiskCheckKey, name: PlanningRiskCheckName},
}

// PlanningScoreValue is one 1 through 5 sub-parameter with a one-sentence summary.
type PlanningScoreValue struct {
	Score   float64
	Summary string
}

// PlanningReviewScores is the full set of review sub-parameters. All three
// must be present when the group is sent.
type PlanningReviewScores struct {
	Clarity    PlanningScoreValue
	Complexity PlanningScoreValue
	Risk       PlanningScoreValue
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
	case PlanningRiskCheckKey:
		return s.Risk
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
		factoryModel, err := FindFactory(inner, s.OrganizationID, s.FactoryID)
		if err != nil {
			return err
		}
		checks, err := order.ListChecks(inner)
		if err != nil {
			return err
		}
		if err := validatePlanningSessionUpdate(update, factoryModel, checks); err != nil {
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

func validatePlanningSessionUpdate(
	update PlanningSessionUpdate,
	factoryModel *Factory,
	checks []FactoryWorkOrderCheck,
) error {
	if update.Scores != nil {
		if factoryModel != nil && !factoryModel.PlanningConfidence {
			return fmt.Errorf("%w: scoring is disabled", ErrFactoryPlanningSessionInvalid)
		}
		if err := validatePlanningReviewScores(*update.Scores); err != nil {
			return err
		}
	}
	if hasPlanningReviewScores(checks) || update.Scores != nil {
		return nil
	}
	if factoryModel != nil && !factoryModel.PlanningConfidence {
		return nil
	}
	return fmt.Errorf("%w: the first plan turn must include scores", ErrFactoryPlanningSessionInvalid)
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

func validatePlanningReviewScore(kind planningScoreKind, value PlanningScoreValue) error {
	if !isFiniteCheckNumber(value.Score) || math.Trunc(value.Score) != value.Score || value.Score < 1 || value.Score > PlanningScoreMax {
		return fmt.Errorf("%w: %s score must be an integer from 1 through 5", ErrFactoryPlanningSessionInvalid, kind.key)
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

func organizationHasPlanningReview(tx *gorm.DB, organizationID uuid.UUID) bool {
	org, err := FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		return false
	}
	return org.HasExperimentalFeature(features.FeatureTaskPlanningReview)
}
