package factory

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const (
	mergeConfidenceMaxScore                   = 3
	mergeConfidenceCautionWhenLowerIsBetter   = 2
	mergeConfidenceCriticalWhenLowerIsBetter  = 3
	mergeConfidenceCautionWhenHigherIsBetter  = 2
	mergeConfidenceCriticalWhenHigherIsBetter = 1
)

// A custom check uses the same bands as blast radius: a higher score is worse.
var mergeConfidenceCheckID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)

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
	lowerIsBetterCheck("risk", "risk-review", "Blast radius"),
	higherIsBetterCheck("performance", "performance-review", "Performance"),
	higherIsBetterCheck("security", "security-review", "Security"),
	lowerIsBetterCheck("drift", "drift-review", "Drift from Specification"),
	higherIsBetterCheck("reversibility", "reversibility-review", "Reversibility"),
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
	labels map[string]string,
) (*models.FactoryWorkOrderCheck, error) {
	params, err := mergeConfidenceCheckParams(check, score, summary, enabled, labels)
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

func mergeConfidenceCheckParams(check string, score float64, summary string, enabled []string, labels map[string]string) (models.FactoryWorkOrderCheckParams, error) {
	name := strings.ToLower(strings.TrimSpace(check))
	if !mergeConfidenceCheckID.MatchString(name) {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: unknown check %q", ErrMergeConfidenceInvalid, check)
	}
	if !slices.Contains(enabled, name) {
		return models.FactoryWorkOrderCheckParams{}, fmt.Errorf("%w: %s", ErrMergeConfidenceDisabled, name)
	}
	spec, ok := mergeConfidenceCheckByName(name)
	if !ok {
		spec = customMergeConfidenceCheck(name, labels)
	} else if label := strings.TrimSpace(labels[name]); label != "" {
		spec.label = label
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

func customMergeConfidenceCheck(name string, labels map[string]string) mergeConfidenceCheck {
	label := strings.TrimSpace(labels[name])
	if label == "" {
		label = name
	}
	return lowerIsBetterCheck(name, name+"-review", label)
}

func lowerIsBetterCheck(name, key, label string) mergeConfidenceCheck {
	return mergeConfidenceCheck{
		name:       name,
		key:        key,
		label:      label,
		direction:  CheckDirectionLowerIsBetter,
		cautionAt:  mergeConfidenceCautionWhenLowerIsBetter,
		criticalAt: mergeConfidenceCriticalWhenLowerIsBetter,
	}
}

func higherIsBetterCheck(name, key, label string) mergeConfidenceCheck {
	return mergeConfidenceCheck{
		name:       name,
		key:        key,
		label:      label,
		direction:  CheckDirectionHigherIsBetter,
		cautionAt:  mergeConfidenceCautionWhenHigherIsBetter,
		criticalAt: mergeConfidenceCriticalWhenHigherIsBetter,
	}
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
		return fmt.Errorf("%w: score must be an integer from 1 through 3", ErrMergeConfidenceInvalid)
	}
	return nil
}
