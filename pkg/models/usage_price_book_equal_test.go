package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__UsagePriceBookRatesEqual__MatchesIdenticalRates(t *testing.T) {
	rows := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
		{UsageKind: models.UsageKindCompute, MatchKey: "e1-large-amd64", MatchMode: models.UsagePriceBookMatchExact, MicrosPerSecond: 70},
	}

	assert.True(t, models.UsagePriceBookRatesEqual(rows, models.CloneUsagePriceBookRates(rows)))
}

func Test__UsagePriceBookRatesEqual__IgnoresOrder(t *testing.T) {
	left := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 250},
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	right := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 250},
	}

	assert.True(t, models.UsagePriceBookRatesEqual(left, right))
}

func Test__UsagePriceBookRatesEqual__DetectsChangedValues(t *testing.T) {
	before := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	after := models.CloneUsagePriceBookRates(before)
	after[0].InputCentsPerMillion = 301

	assert.False(t, models.UsagePriceBookRatesEqual(before, after))
}

func Test__UsagePriceBookRatesEqual__DetectsAddedRows(t *testing.T) {
	before := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	after := append(models.CloneUsagePriceBookRates(before), models.UsagePriceBookRate{
		UsageKind: models.UsageKindModel, MatchKey: "lab-model-1", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 12,
	})

	assert.False(t, models.UsagePriceBookRatesEqual(before, after))
}
