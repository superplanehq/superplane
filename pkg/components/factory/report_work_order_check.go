package factory

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ReportWorkOrderCheckComponentName = "reportWorkOrderCheck"

const (
	CheckDirectionHigherIsBetter = "higherIsBetter"
	CheckDirectionLowerIsBetter  = "lowerIsBetter"
)

func init() {
	registry.RegisterAction(ReportWorkOrderCheckComponentName, &ReportWorkOrderCheck{})
}

type ReportWorkOrderCheck struct{}

type ReportWorkOrderCheckConfiguration struct {
	OrderID  string `json:"orderId" mapstructure:"orderId"`
	CheckKey string `json:"checkKey" mapstructure:"checkKey"`
	Name     string `json:"name" mapstructure:"name"`
	// Score and MaxScore are strings so they can carry expressions
	// (e.g. {{ previous().data.risk.score }}); parsed at Execute time.
	Score    string `json:"score" mapstructure:"score"`
	MaxScore string `json:"maxScore" mapstructure:"maxScore"`
	Format   string `json:"format" mapstructure:"format"`
	// Passed replaces Score for boolean checks: a true/false verdict
	// (accepts expressions). Pass maps to level positive, fail to
	// critical.
	Passed string `json:"passed" mapstructure:"passed"`
	// Direction plus the two thresholds determine the check's level
	// declaratively: crossing CautionAt in the bad direction flags
	// caution, crossing CriticalAt flags critical. With no thresholds
	// the check stays neutral.
	Direction  string   `json:"direction" mapstructure:"direction"`
	CautionAt  *float64 `json:"cautionAt" mapstructure:"cautionAt"`
	CriticalAt *float64 `json:"criticalAt" mapstructure:"criticalAt"`
	Summary    string   `json:"summary" mapstructure:"summary"`
	Analysis   string   `json:"analysis" mapstructure:"analysis"`
}

func (c *ReportWorkOrderCheck) Name() string {
	return ReportWorkOrderCheckComponentName
}

func (c *ReportWorkOrderCheck) Label() string {
	return "Report Work Order Check"
}

func (c *ReportWorkOrderCheck) Description() string {
	return "Report a scored or pass/fail check (risk, coverage, CI) on a work order"
}

func (c *ReportWorkOrderCheck) Documentation() string {
	return `The Report Work Order Check component stores a scored check against a work order. Checks appear as scorecards on the work order page, next to the description.

Each check is identified by its ` + "`checkKey`" + ` (for example ` + "`risk-review`" + `). The first report creates the check. A later report with the same key updates the check in place and keeps the prior score, so the UI can show the movement between runs. Every report also adds an entry to the work order timeline.

A check is either scored or a pass/fail verdict, controlled by ` + "`format`" + `:

- ` + "`fraction`" + ` and ` + "`percent`" + ` are scored checks. Set ` + "`score`" + ` and ` + "`maxScore`" + `.
- ` + "`boolean`" + ` is a pass/fail verdict (e.g. CI status). Set ` + "`passed`" + ` to true or false instead of a score; a pass shows as healthy, a fail as critical.

For scored checks, the level (healthy / neutral / needs attention / critical) is computed from the score and the thresholds you configure:

- ` + "`direction`" + ` says which way is bad: ` + "`higherIsBetter`" + ` (e.g. coverage) or ` + "`lowerIsBetter`" + ` (e.g. risk).
- ` + "`cautionAt`" + ` and ` + "`criticalAt`" + ` are score thresholds. When the score crosses a threshold in the bad direction, the check shows that level.
- With no thresholds set, the check stays neutral.

For example, a risk score from 0 to 10 where lower is better, with ` + "`cautionAt: 5`" + ` and ` + "`criticalAt: 8`" + `: a score of 3 is healthy, 6 needs attention, 9 is critical.

` + "`score`" + `, ` + "`maxScore`" + `, and ` + "`passed`" + ` accept expressions, so a preceding automation run can feed them (e.g. ` + "`{{ previous().data.risk.score }}`" + `). Use ` + "`analysis`" + ` for the full markdown report — it renders when a user opens the check.

` + "`orderId`" + ` explicitly targets the work order — it defaults to ` + "`{{ order().id }}`" + `, the work order driving the current run. This component can only be used in factory-owned apps.`
}

func (c *ReportWorkOrderCheck) Icon() string {
	return "factory"
}

func (c *ReportWorkOrderCheck) Color() string {
	return "blue"
}

func (c *ReportWorkOrderCheck) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.checkReported",
		"data": map[string]any{
			"check": map[string]any{
				"id":       "chk-123",
				"key":      "risk-review",
				"name":     "Risk review",
				"score":    3,
				"maxScore": 10,
				"format":   "fraction",
				"level":    "positive",
			},
		},
	}
}

func (c *ReportWorkOrderCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReportWorkOrderCheck) Configuration() []configuration.Field {
	// The empty value keeps numeric fields visible for configs saved
	// before the format field existed (they default to fraction).
	numericFormats := []configuration.VisibilityCondition{
		{Field: "format", Values: []string{"", factory.CheckFormatFraction, factory.CheckFormatPercent}},
	}
	booleanFormat := []configuration.VisibilityCondition{
		{Field: "format", Values: []string{factory.CheckFormatBoolean}},
	}

	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Work Order ID",
			Description: "Work order to target. Defaults to the work order driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "checkKey",
			Label:       "Check Key",
			Description: "Stable identifier for this check on the work order (e.g. risk-review). Reports with the same key update the same check.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "name",
			Label:       "Name",
			Description: "Display name shown on the check card (e.g. Risk review)",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "format",
			Label:       "Format",
			Description: "How the check is rendered: as score/maxScore, as a percentage, or as a pass/fail verdict",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     factory.CheckFormatFraction,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Fraction (score/max)", Value: factory.CheckFormatFraction},
						{Label: "Percent", Value: factory.CheckFormatPercent},
						{Label: "Boolean (pass/fail)", Value: factory.CheckFormatBoolean},
					},
				},
			},
		},
		{
			Name:                 "passed",
			Label:                "Passed",
			Description:          "The pass/fail verdict: true or false. Accepts expressions, e.g. {{ previous().data.ci.passed }}.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: booleanFormat,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "format", Values: []string{factory.CheckFormatBoolean}},
			},
		},
		{
			Name:                 "score",
			Label:                "Score",
			Description:          "The score to report. Accepts expressions, e.g. {{ previous().data.risk.score }}.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			VisibilityConditions: numericFormats,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "format", Values: []string{"", factory.CheckFormatFraction, factory.CheckFormatPercent}},
			},
		},
		{
			Name:                 "maxScore",
			Label:                "Max Score",
			Description:          "The scale's maximum (e.g. 10, or 100 for percentages). Accepts expressions.",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Default:              "100",
			VisibilityConditions: numericFormats,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "format", Values: []string{"", factory.CheckFormatFraction, factory.CheckFormatPercent}},
			},
		},
		{
			Name:        "direction",
			Label:       "Direction",
			Description: "Which way is bad: higher is better (e.g. coverage) or lower is better (e.g. risk). Used with the thresholds below to compute the check's level.",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     CheckDirectionHigherIsBetter,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Higher is better", Value: CheckDirectionHigherIsBetter},
						{Label: "Lower is better", Value: CheckDirectionLowerIsBetter},
					},
				},
			},
			VisibilityConditions: numericFormats,
		},
		{
			Name:                 "cautionAt",
			Label:                "Caution Threshold",
			Description:          "Score at which the check shows \"needs attention\". Leave both thresholds unset to keep the check neutral.",
			Type:                 configuration.FieldTypeNumber,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: numericFormats,
		},
		{
			Name:                 "criticalAt",
			Label:                "Critical Threshold",
			Description:          "Score at which the check shows \"critical\"",
			Type:                 configuration.FieldTypeNumber,
			Required:             false,
			Togglable:            true,
			VisibilityConditions: numericFormats,
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Description: "One-line takeaway shown when the check is opened",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
		{
			Name:        "analysis",
			Label:       "Analysis",
			Description: "Full markdown report behind the check — rendered when a user opens it",
			Type:        configuration.FieldTypeText,
			Required:    false,
		},
	}
}

func (c *ReportWorkOrderCheck) Execute(ctx core.ExecutionContext) error {
	config := ReportWorkOrderCheckConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	score, maxScore, level, err := resolveCheckScore(config)
	if err != nil {
		return err
	}

	check, err := ctx.Factory.ReportWorkOrderCheck(core.ReportWorkOrderCheckParams{
		OrderID:  config.OrderID,
		CheckKey: config.CheckKey,
		Name:     config.Name,
		Score:    score,
		MaxScore: maxScore,
		Format:   config.Format,
		Level:    level,
		Summary:  config.Summary,
		Analysis: config.Analysis,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.checkReported",
		[]any{map[string]any{
			"check": check,
		}},
	)
}

func (c *ReportWorkOrderCheck) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ReportWorkOrderCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReportWorkOrderCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ReportWorkOrderCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ReportWorkOrderCheck) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ReportWorkOrderCheck) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// resolveCheckScore turns the configuration into the (score, maxScore,
// level) triple to store. Numeric formats parse the score fields and
// compute the level from the thresholds; the boolean format derives all
// three from the pass/fail verdict.
func resolveCheckScore(config ReportWorkOrderCheckConfiguration) (float64, float64, string, error) {
	if config.Format == factory.CheckFormatBoolean {
		return resolveBooleanCheckScore(config)
	}

	if strings.TrimSpace(config.Passed) != "" {
		return 0, 0, "", fmt.Errorf("passed only applies to the boolean format")
	}

	score, err := parseCheckScore("score", config.Score)
	if err != nil {
		return 0, 0, "", err
	}
	maxScore, err := parseCheckScore("maxScore", config.MaxScore)
	if err != nil {
		return 0, 0, "", err
	}

	level, err := computeCheckLevel(score, config.Direction, config.CautionAt, config.CriticalAt)
	if err != nil {
		return 0, 0, "", err
	}

	return score, maxScore, level, nil
}

// resolveBooleanCheckScore maps the pass/fail verdict onto the score
// model: pass is 1/1 with level positive, fail is 0/1 with level
// critical. Score thresholds do not apply — a verdict has no bands.
func resolveBooleanCheckScore(config ReportWorkOrderCheckConfiguration) (float64, float64, string, error) {
	if config.CautionAt != nil || config.CriticalAt != nil {
		return 0, 0, "", fmt.Errorf("cautionAt and criticalAt do not apply to the boolean format")
	}

	passed, err := parseCheckPassed(config.Passed)
	if err != nil {
		return 0, 0, "", err
	}

	if passed {
		return 1, 1, factory.CheckLevelPositive, nil
	}
	return 0, 1, factory.CheckLevelCritical, nil
}

// parseCheckPassed converts a resolved pass/fail expression to a bool.
// Expressions resolve booleans to "true"/"false"; 1/0 are accepted for
// automations that emit numeric flags.
func parseCheckPassed(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return false, fmt.Errorf("passed is required for the boolean format")
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("passed must be true or false, got %q", raw)
	}
}

// parseCheckScore converts a resolved score expression to a number,
// failing with the field name so a broken expression is easy to trace.
func parseCheckScore(fieldName, raw string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", fieldName, raw)
	}

	return value, nil
}

// computeCheckLevel resolves the declarative thresholds into a level.
// With no thresholds the check is neutral; otherwise crossing a
// threshold in the configured bad direction escalates the level.
func computeCheckLevel(score float64, direction string, cautionAt, criticalAt *float64) (string, error) {
	if direction == "" {
		direction = CheckDirectionHigherIsBetter
	}
	if direction != CheckDirectionHigherIsBetter && direction != CheckDirectionLowerIsBetter {
		return "", fmt.Errorf("unknown direction %q", direction)
	}

	if cautionAt == nil && criticalAt == nil {
		return factory.CheckLevelNeutral, nil
	}

	if err := validateCheckThresholdOrder(direction, cautionAt, criticalAt); err != nil {
		return "", err
	}

	crossed := func(threshold float64) bool {
		if direction == CheckDirectionLowerIsBetter {
			return score >= threshold
		}
		return score <= threshold
	}

	if criticalAt != nil && crossed(*criticalAt) {
		return factory.CheckLevelCritical, nil
	}
	if cautionAt != nil && crossed(*cautionAt) {
		return factory.CheckLevelCaution, nil
	}

	return factory.CheckLevelPositive, nil
}

// validateCheckThresholdOrder rejects threshold pairs where the caution
// band could never be reached (e.g. lowerIsBetter with criticalAt below
// cautionAt) — almost certainly a configuration mistake.
func validateCheckThresholdOrder(direction string, cautionAt, criticalAt *float64) error {
	if cautionAt == nil || criticalAt == nil {
		return nil
	}

	if direction == CheckDirectionLowerIsBetter && *criticalAt < *cautionAt {
		return fmt.Errorf(
			"criticalAt (%v) must be greater than or equal to cautionAt (%v) when lower is better",
			*criticalAt, *cautionAt,
		)
	}
	if direction == CheckDirectionHigherIsBetter && *criticalAt > *cautionAt {
		return fmt.Errorf(
			"criticalAt (%v) must be less than or equal to cautionAt (%v) when higher is better",
			*criticalAt, *cautionAt,
		)
	}

	return nil
}
