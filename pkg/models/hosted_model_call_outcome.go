package models

import (
	"bytes"
	"encoding/json"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	HostedModelCallOutcomeAttempt1 = "attempt_1"
	HostedModelCallOutcomeAttempt2 = "attempt_2"
	HostedModelCallOutcomeAttempt3 = "attempt_3"
	HostedModelCallOutcomeAttempt4 = "attempt_4"
	HostedModelCallOutcomeError    = "error"

	HostedModelCallOutcomeWindow5m  = "5m"
	HostedModelCallOutcomeWindow15m = "15m"
	HostedModelCallOutcomeWindow1h  = "1h"
	HostedModelCallOutcomeWindow24h = "24h"
)

var hostedModelCallOutcomeScores = map[string]int{
	HostedModelCallOutcomeAttempt1: 100,
	HostedModelCallOutcomeAttempt2: 75,
	HostedModelCallOutcomeAttempt3: 50,
	HostedModelCallOutcomeAttempt4: 25,
	HostedModelCallOutcomeError:    0,
}

var hostedModelCallOutcomeWindows = map[string]time.Duration{
	HostedModelCallOutcomeWindow5m:  5 * time.Minute,
	HostedModelCallOutcomeWindow15m: 15 * time.Minute,
	HostedModelCallOutcomeWindow1h:  time.Hour,
	HostedModelCallOutcomeWindow24h: 24 * time.Hour,
}

// HostedModelCallOutcome is one SuperPlane-hosted model call result used for
// the installation-wide model status page.
type HostedModelCallOutcome struct {
	ID           uuid.UUID
	Provider     string
	Model        string
	Outcome      string
	OccurredAt   time.Time
	RunnerTaskID string
	Sequence     int
}

func (HostedModelCallOutcome) TableName() string {
	return "hosted_model_call_outcomes"
}

// HostedModelCallOutcomeInput is one sample to persist from a finished runner task.
type HostedModelCallOutcomeInput struct {
	Provider     string
	Model        string
	Outcome      string
	OccurredAt   time.Time
	RunnerTaskID string
	Sequence     int
}

// HostedModelStatusCell is the scored aggregate for one provider and model.
type HostedModelStatusCell struct {
	Provider      string
	Model         string
	Score         *int
	CallCount     int
	Attempt1Count int
	Attempt2Count int
	Attempt3Count int
	Attempt4Count int
	ErrorCount    int
}

type hostedModelCallOutcomeCount struct {
	Provider string
	Model    string
	Outcome  string
	Count    int
}

func IsHostedModelCallOutcome(outcome string) bool {
	_, ok := hostedModelCallOutcomeScores[outcome]
	return ok
}

func HostedModelCallOutcomeWindowDuration(window string) (string, time.Duration, bool) {
	normalized := strings.TrimSpace(window)
	if normalized == "" {
		normalized = HostedModelCallOutcomeWindow5m
	}
	duration, ok := hostedModelCallOutcomeWindows[normalized]
	return normalized, duration, ok
}

func ScoreHostedModelCallOutcomes(attempt1, attempt2, attempt3, attempt4, errors int) *int {
	total := attempt1 + attempt2 + attempt3 + attempt4 + errors
	if total == 0 {
		return nil
	}
	sum := attempt1*hostedModelCallOutcomeScores[HostedModelCallOutcomeAttempt1] +
		attempt2*hostedModelCallOutcomeScores[HostedModelCallOutcomeAttempt2] +
		attempt3*hostedModelCallOutcomeScores[HostedModelCallOutcomeAttempt3] +
		attempt4*hostedModelCallOutcomeScores[HostedModelCallOutcomeAttempt4]
	score := int(math.Round(float64(sum) / float64(total)))
	return &score
}

func ParseHostedModelCallLogChunk(taskID string, content []byte) []HostedModelCallOutcomeInput {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || !bytes.Contains(content, []byte(`"type":"model_call"`)) {
		return nil
	}
	var inputs []HostedModelCallOutcomeInput
	for _, line := range bytes.Split(content, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.Contains(line, []byte(`"type":"model_call"`)) {
			continue
		}
		var row struct {
			Type       string `json:"type"`
			Provider   string `json:"provider"`
			Model      string `json:"model"`
			Outcome    string `json:"outcome"`
			OccurredAt string `json:"occurredAt"`
			Sequence   int    `json:"sequence"`
		}
		if err := json.Unmarshal(line, &row); err != nil || strings.TrimSpace(row.Type) != "model_call" || row.Sequence < 1 {
			continue
		}
		occurredAt := time.Time{}
		if stamp := strings.TrimSpace(row.OccurredAt); stamp != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
				occurredAt = parsed
			} else if parsed, err := time.Parse(time.RFC3339, stamp); err == nil {
				occurredAt = parsed
			}
		}
		inputs = append(inputs, HostedModelCallOutcomeInput{
			Provider:     strings.TrimSpace(row.Provider),
			Model:        strings.TrimSpace(row.Model),
			Outcome:      strings.TrimSpace(row.Outcome),
			OccurredAt:   occurredAt,
			RunnerTaskID: taskID,
			Sequence:     row.Sequence,
		})
	}
	return inputs
}

func InsertHostedModelCallOutcomes(tx *gorm.DB, inputs []HostedModelCallOutcomeInput) error {
	if len(inputs) == 0 {
		return nil
	}
	rows := make([]HostedModelCallOutcome, 0, len(inputs))
	for _, input := range inputs {
		provider := strings.TrimSpace(input.Provider)
		model := strings.TrimSpace(input.Model)
		outcome := strings.TrimSpace(input.Outcome)
		taskID := strings.TrimSpace(input.RunnerTaskID)
		if !isHostedModelStatusProvider(provider) || model == "" || !IsHostedModelCallOutcome(outcome) || taskID == "" {
			continue
		}
		occurredAt := input.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		rows = append(rows, HostedModelCallOutcome{
			ID:           uuid.New(),
			Provider:     provider,
			Model:        model,
			Outcome:      outcome,
			OccurredAt:   occurredAt,
			RunnerTaskID: taskID,
			Sequence:     input.Sequence,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "runner_task_id"}, {Name: "sequence"}},
		DoNothing: true,
	}).Create(&rows).Error
}

func DeleteExpiredHostedModelCallOutcomes(tx *gorm.DB, before time.Time) (int64, error) {
	result := tx.Where("occurred_at < ?", before).Delete(&HostedModelCallOutcome{})
	return result.RowsAffected, result.Error
}

func ListHostedModelStatusCells(tx *gorm.DB, since time.Time) ([]HostedModelStatusCell, error) {
	counts, err := listHostedModelCallOutcomeCounts(tx, since)
	if err != nil {
		return nil, err
	}
	allowlist, err := listHostedModelAllowlistPairs(tx)
	if err != nil {
		return nil, err
	}

	cells := map[string]*HostedModelStatusCell{}
	addCell := func(provider, model string) *HostedModelStatusCell {
		key := provider + "\x00" + model
		cell, ok := cells[key]
		if ok {
			return cell
		}
		cell = &HostedModelStatusCell{Provider: provider, Model: model}
		cells[key] = cell
		return cell
	}
	for _, pair := range allowlist {
		addCell(pair.provider, pair.model)
	}
	for _, row := range counts {
		cell := addCell(row.Provider, row.Model)
		switch row.Outcome {
		case HostedModelCallOutcomeAttempt1:
			cell.Attempt1Count = row.Count
		case HostedModelCallOutcomeAttempt2:
			cell.Attempt2Count = row.Count
		case HostedModelCallOutcomeAttempt3:
			cell.Attempt3Count = row.Count
		case HostedModelCallOutcomeAttempt4:
			cell.Attempt4Count = row.Count
		case HostedModelCallOutcomeError:
			cell.ErrorCount = row.Count
		}
	}

	out := make([]HostedModelStatusCell, 0, len(cells))
	for _, cell := range cells {
		cell.CallCount = cell.Attempt1Count + cell.Attempt2Count + cell.Attempt3Count + cell.Attempt4Count + cell.ErrorCount
		cell.Score = ScoreHostedModelCallOutcomes(cell.Attempt1Count, cell.Attempt2Count, cell.Attempt3Count, cell.Attempt4Count, cell.ErrorCount)
		out = append(out, *cell)
	}
	slices.SortFunc(out, func(a, b HostedModelStatusCell) int {
		if a.Provider != b.Provider {
			return strings.Compare(a.Provider, b.Provider)
		}
		return strings.Compare(a.Model, b.Model)
	})
	return out, nil
}

func listHostedModelCallOutcomeCounts(tx *gorm.DB, since time.Time) ([]hostedModelCallOutcomeCount, error) {
	var rows []hostedModelCallOutcomeCount
	err := tx.Model(&HostedModelCallOutcome{}).
		Select("provider, model, outcome, count(*) as count").
		Where("occurred_at >= ?", since).
		Group("provider, model, outcome").
		Find(&rows).Error
	return rows, err
}

type hostedModelAllowlistPair struct {
	provider string
	model    string
}

func listHostedModelAllowlistPairs(tx *gorm.DB) ([]hostedModelAllowlistPair, error) {
	providers, err := ListHostedLLMProviders(tx)
	if err != nil {
		return nil, err
	}
	var pairs []hostedModelAllowlistPair
	for _, provider := range providers {
		if !isHostedModelStatusProvider(provider.Provider) {
			continue
		}
		for _, model := range provider.AllowedModels {
			trimmed := strings.TrimSpace(model)
			if trimmed == "" {
				continue
			}
			pairs = append(pairs, hostedModelAllowlistPair{provider: provider.Provider, model: trimmed})
		}
	}
	return pairs, nil
}

func isHostedModelStatusProvider(provider string) bool {
	switch provider {
	case UsageProviderAnthropic, UsageProviderOpenAI, UsageProviderOpenRouter:
		return true
	default:
		return false
	}
}
