package analyze

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	taguchi "github.com/marijaaleksic/taguchi"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName            = "taguchiAnalyze"
	PayloadType              = "taguchi.analysis"
	ChannelNameConfident     = "confident"
	ChannelNameInconclusive  = "inconclusive"
	DirectionLargerIsBetter  = "larger"
	DirectionSmallerIsBetter = "smaller"
	MemoryKindArm            = "arm"
	MemoryKindTrial          = "trial"
)

func init() {
	registry.RegisterComponent(ComponentName, &Analyze{})
}

type Analyze struct{}

type Spec struct {
	ExperimentID        string  `json:"experimentId"`
	Metric              string  `json:"metric"`
	Direction           string  `json:"direction"`
	ConfidenceThreshold float64 `json:"confidenceThreshold,omitempty"`
}

func (c *Analyze) Name() string  { return ComponentName }
func (c *Analyze) Label() string { return "Taguchi Analyze" }
func (c *Analyze) Description() string {
	return "Rank arms, compute main effects, and pick a winner"
}
func (c *Analyze) Documentation() string {
	return `Reads arms and trials from canvas memory, computes per-arm mean for the target metric, runs Taguchi main-effects analysis, and emits ` + "`confident`" + ` when the gap between winner and runner-up exceeds the pooled standard deviation by ` + "`confidenceThreshold`" + ` (default 1.0). Otherwise emits ` + "`inconclusive`" + `.`
}
func (c *Analyze) Icon() string                  { return "chart-bar" }
func (c *Analyze) Color() string                 { return "amber" }
func (c *Analyze) ExampleOutput() map[string]any { return map[string]any{} }
func (c *Analyze) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameConfident, Label: "Confident"},
		{Name: ChannelNameInconclusive, Label: "Inconclusive"},
	}
}
func (c *Analyze) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "experimentId", Label: "Experiment ID", Type: configuration.FieldTypeExpression, Required: true},
		{Name: "metric", Label: "Metric Name", Type: configuration.FieldTypeString, Required: true,
			Description: "Name of the metric recorded on each trial row (e.g. rematch_rate)"},
		{Name: "direction", Label: "Direction", Type: configuration.FieldTypeSelect, Required: true,
			Default: DirectionLargerIsBetter,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Larger is better", Value: DirectionLargerIsBetter},
						{Label: "Smaller is better", Value: DirectionSmallerIsBetter},
					},
				},
			},
		},
		{Name: "confidenceThreshold", Label: "Confidence Threshold (z-ish)", Type: configuration.FieldTypeNumber,
			Default: 1.0, Description: "How many pooled std devs of gap required for 'confident'"},
	}
}

func (c *Analyze) Setup(ctx core.SetupContext) error {
	_, err := decode(ctx.Configuration)
	return err
}

func (c *Analyze) Execute(ctx core.ExecutionContext) error {
	spec, err := decode(ctx.Configuration)
	if err != nil {
		return err
	}

	namespace := "taguchi:" + spec.ExperimentID
	armRows, err := ctx.CanvasMemory.Find(namespace, map[string]any{"kind": MemoryKindArm})
	if err != nil {
		return fmt.Errorf("failed to read arms: %w", err)
	}
	trialRows, err := ctx.CanvasMemory.Find(namespace, map[string]any{"kind": MemoryKindTrial})
	if err != nil {
		return fmt.Errorf("failed to read trials: %w", err)
	}
	if len(armRows) == 0 {
		return fmt.Errorf("no arms found for experiment %s", spec.ExperimentID)
	}

	// Group trial observations by arm.
	armObservations := map[string][]float64{}
	for _, row := range trialRows {
		m, _ := row.(map[string]any)
		id, ok := m["arm_id"].(string)
		if !ok {
			continue
		}
		if metricName, _ := m["metric"].(string); metricName != "" && metricName != spec.Metric {
			continue
		}
		v, ok := toFloat(m["value"])
		if !ok {
			continue
		}
		armObservations[id] = append(armObservations[id], v)
	}

	ranks := make([]armRank, 0, len(armRows))
	for _, row := range armRows {
		m, _ := row.(map[string]any)
		id, _ := m["arm_id"].(string)
		params, _ := m["params"].(map[string]any)
		obs := armObservations[id]
		mean, std := meanStd(obs)
		ranks = append(ranks, armRank{ID: id, Params: params, Mean: mean, StdDev: std, N: len(obs)})
	}

	// Sort: best first. Larger-is-better => descending mean; smaller-is-better => ascending.
	sort.Slice(ranks, func(i, j int) bool {
		if spec.Direction == DirectionSmallerIsBetter {
			return ranks[i].Mean < ranks[j].Mean
		}
		return ranks[i].Mean > ranks[j].Mean
	})

	// Pooled std across arms for confidence calculation.
	pooled := pooledStd(ranks)

	confidence := 0.0
	if len(ranks) >= 2 && pooled > 0 {
		confidence = math.Abs(ranks[0].Mean-ranks[1].Mean) / pooled
	}

	// Main-effects analysis via taguchi library.
	mainEffects := computeMainEffects(armRows, armObservations, spec)

	channel := ChannelNameInconclusive
	threshold := spec.ConfidenceThreshold
	if threshold == 0 {
		threshold = 1.0
	}
	if len(ranks) >= 2 && confidence >= threshold {
		channel = ChannelNameConfident
	}

	rankingPayload := make([]any, 0, len(ranks))
	for _, r := range ranks {
		rankingPayload = append(rankingPayload, map[string]any{
			"arm_id": r.ID,
			"params": r.Params,
			"mean":   r.Mean,
			"stddev": r.StdDev,
			"n":      r.N,
		})
	}

	winner := map[string]any{}
	if len(ranks) > 0 {
		winner = map[string]any{
			"arm_id": ranks[0].ID,
			"params": ranks[0].Params,
			"mean":   ranks[0].Mean,
		}
	}

	_ = ctx.Metadata.Set(map[string]any{
		"experimentId": spec.ExperimentID,
		"metric":       spec.Metric,
		"direction":    spec.Direction,
		"confidence":   confidence,
		"channel":      channel,
	})

	return ctx.ExecutionState.Emit(channel, PayloadType, []any{
		map[string]any{
			"experimentId":        spec.ExperimentID,
			"metric":              spec.Metric,
			"direction":           spec.Direction,
			"confidence":          confidence,
			"confidenceThreshold": threshold,
			"winner":              winner,
			"ranking":             rankingPayload,
			"mainEffects":         mainEffects,
		},
	})
}

func decode(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	spec.ExperimentID = strings.TrimSpace(spec.ExperimentID)
	spec.Metric = strings.TrimSpace(spec.Metric)
	if spec.Direction == "" {
		spec.Direction = DirectionLargerIsBetter
	}
	if spec.ExperimentID == "" {
		return Spec{}, fmt.Errorf("experimentId is required")
	}
	if spec.Metric == "" {
		return Spec{}, fmt.Errorf("metric is required")
	}
	if spec.Direction != DirectionLargerIsBetter && spec.Direction != DirectionSmallerIsBetter {
		return Spec{}, fmt.Errorf("direction must be %q or %q", DirectionLargerIsBetter, DirectionSmallerIsBetter)
	}
	return spec, nil
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case string:
		// Trial values written via template expressions (e.g. `{{ $[...].body.value }}`)
		// are stringified by the config resolver; parse them back into a number.
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func meanStd(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	if len(xs) < 2 {
		return mean, 0
	}
	s := 0.0
	for _, x := range xs {
		s += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(s / float64(len(xs)-1))
}

type armRank struct {
	ID     string
	Params map[string]any
	Mean   float64
	StdDev float64
	N      int
}

// pooledStd computes a rough pooled standard deviation across all arms.
func pooledStd(ranks []armRank) float64 {
	num, den := 0.0, 0
	for _, r := range ranks {
		if r.N < 2 {
			continue
		}
		num += float64(r.N-1) * r.StdDev * r.StdDev
		den += r.N - 1
	}
	if den == 0 {
		return 0
	}
	return math.Sqrt(num / float64(den))
}

// computeMainEffects runs the taguchi library's Analyze() on the observed means
// to recover per-factor / per-level contributions.
func computeMainEffects(armRows []any, obs map[string][]float64, spec Spec) map[string]any {
	// Collect factor definitions from arm params.
	factorOrder := []string{}
	levelIndex := map[string]map[string]int{} // factor -> level -> 1-based index
	for _, row := range armRows {
		m, _ := row.(map[string]any)
		params, _ := m["params"].(map[string]any)
		for name, val := range params {
			v := fmt.Sprintf("%v", val)
			if _, ok := levelIndex[name]; !ok {
				levelIndex[name] = map[string]int{}
				factorOrder = append(factorOrder, name)
			}
			if _, ok := levelIndex[name][v]; !ok {
				levelIndex[name][v] = len(levelIndex[name]) + 1
			}
		}
	}
	sort.Strings(factorOrder)

	controls := make([]taguchi.ControlFactor, 0, len(factorOrder))
	for _, name := range factorOrder {
		levels := make([]float64, len(levelIndex[name]))
		for i := range levels {
			levels[i] = float64(i + 1)
		}
		controls = append(controls, taguchi.ControlFactor{Name: name, Levels: levels})
	}

	// Build an ad-hoc orthogonal array: one row per actual arm with observed mean.
	orth := [][]int{}
	observations := [][]float64{}
	for _, row := range armRows {
		m, _ := row.(map[string]any)
		id, _ := m["arm_id"].(string)
		params, _ := m["params"].(map[string]any)
		armObs := obs[id]
		if len(armObs) == 0 {
			continue
		}
		rowLevels := make([]int, len(factorOrder))
		for i, name := range factorOrder {
			rowLevels[i] = levelIndex[name][fmt.Sprintf("%v", params[name])]
		}
		orth = append(orth, rowLevels)
		observations = append(observations, armObs)
	}

	if len(orth) == 0 {
		return map[string]any{}
	}

	var goal taguchi.OptimizationGoal = taguchi.LargerTheBetter{}
	if spec.Direction == DirectionSmallerIsBetter {
		goal = taguchi.SmallerTheBetter{}
	}

	exp, err := taguchi.NewExperimentFromFactorsUsingArray(goal, controls, orth, nil)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	trials := exp.GenerateTrials()
	for i, trial := range trials {
		if i >= len(observations) {
			break
		}
		exp.AddResult(trial, observations[i])
	}
	res := exp.Analyze()

	// Translate numeric level indices back to string labels.
	effectsByFactor := map[string]any{}
	for factor, arr := range res.MainEffects {
		labels := map[string]float64{}
		for idx, snr := range arr {
			for label, one := range levelIndex[factor] {
				if one == idx+1 {
					labels[label] = snr
				}
			}
		}
		effectsByFactor[factor] = labels
	}

	optimal := map[string]any{}
	for factor, levelNum := range res.OptimalLevels {
		for label, one := range levelIndex[factor] {
			if float64(one) == levelNum {
				optimal[factor] = label
			}
		}
	}

	return map[string]any{
		"effects":       effectsByFactor,
		"optimalLevels": optimal,
		"contributions": res.Contributions,
	}
}

func (c *Analyze) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *Analyze) Actions() []core.Action                  { return nil }
func (c *Analyze) HandleAction(_ core.ActionContext) error { return fmt.Errorf("%s does not support actions", ComponentName) }
func (c *Analyze) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *Analyze) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *Analyze) Cleanup(_ core.SetupContext) error { return nil }
