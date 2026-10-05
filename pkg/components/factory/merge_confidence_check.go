package factory

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const mergeConfidenceMaxScore = 5

var (
	ErrMergeConfidenceInvalid  = errors.New("merge confidence check is invalid")
	ErrMergeConfidenceDisabled = errors.New("merge confidence check is off")
)

type mergeConfidenceCheck struct {
	name       string
	key        string
	label      string
	direction  string
	cautionAt  float64
	criticalAt float64
}

// The names, keys, and thresholds used to live on canvas report nodes.
// The server owns them so the agent only sends a score and a summary.
var mergeConfidenceChecks = []mergeConfidenceCheck{
	{name: "risk", key: "risk-review", label: "Blast radius", direction: CheckDirectionLowerIsBetter, cautionAt: 3, criticalAt: 4},
	{name: "performance", key: "performance-review", label: "Performance", direction: CheckDirectionHigherIsBetter, cautionAt: 3, criticalAt: 2},
	{name: "security", key: "security-review", label: "Security", direction: CheckDirectionHigherIsBetter, cautionAt: 3, criticalAt: 2},
	{name: "drift", key: "drift-review", label: "Drift from Specification", direction: CheckDirectionLowerIsBetter, cautionAt: 3, criticalAt: 4},
	{name: "reversibility", key: "reversibility-review", label: "Reversibility", direction: CheckDirectionHigherIsBetter, cautionAt: 3, criticalAt: 2},
}

// ReportMergeConfidenceCheck stores one merge confidence check. enabled is the
// list from the runner token. A check that is not in that list is rejected.
func ReportMergeConfidenceCheck(
	tx *gorm.DB,
	order *models.FactoryWorkOrder,
	runID uuid.UUID,
	automation *factory.AutomationRef,
	check string,
	score float64,
	summary string,
	enabled []string,
) (*models.FactoryWorkOrderCheck, error) {
	params, err := mergeConfidenceCheckParams(check, score, summary, enabled)
	if err != nil {
		return nil, err
	}
	if runID == uuid.Nil || automation == nil || automation.AppID == uuid.Nil {
		return nil, fmt.Errorf("%w: run and automation are required", ErrMergeConfidenceInvalid)
	}
	if order == nil {
		return nil, models.ErrFactoryWorkOrderNotFound
	}
	params.Run = &factory.RunRef{ID: runID}
	params.Automation = automation
	return order.ReportCheck(tx, params)
}

func mergeConfidenceCheckParams(check string, score float64, summary string, enabled []string) (models.FactoryWorkOrderCheckParams, error) {
	name := strings.ToLower(strings.TrimSpace(check))
	spec, ok := mergeConfidenceCheckByName(name)
	if !ok {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: unknown check %q", ErrMergeConfidenceInvalid, check)
	}
	if !slices.Contains(enabled, spec.name) {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: %s", ErrMergeConfidenceDisabled, spec.name)
	}
	if err := validateMergeConfidenceScore(score); err != nil {
		return models.FactoryWorkOrderCheckParams{}, err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: summary is required", ErrMergeConfidenceInvalid)
	}
	if len(summary) > 1000 {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: summary is too long", ErrMergeConfidenceInvalid)
	}
	caution := spec.cautionAt
	critical := spec.criticalAt
	level, err := computeCheckLevel(score, spec.direction, &caution, &critical)
	if err != nil {
		return models.FactoryWorkOrderCheckParams{}, err
	}
	return models.FactoryWorkOrderCheckParams{
		Key:      spec.key,
		Name:     spec.label,
		Score:    score,
		MaxScore: mergeConfidenceMaxScore,
		Format:   factory.CheckFormatFraction,
		Level:    level,
		Summary:  summary,
	}, nil
}

func mergeConfidenceCheckByName(name string) (mergeConfidenceCheck, bool) {
	for _, spec := range mergeConfidenceChecks {
		if spec.name == name {
			return spec, true
		}
	}
	return mergeConfidenceCheck{}, false
}

func validateMergeConfidenceScore(score float64) error {
	if math.IsNaN(score) || math.IsInf(score, 0) || math.Trunc(score) != score || score < 1 || score > mergeConfidenceMaxScore {
		return fmt.Errorf("%w: score must be an integer from 1 through 5", ErrMergeConfidenceInvalid)
	}
	return nil
}
