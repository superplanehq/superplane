package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__UsagePriceBookRatesEqual__MatchesIdenticalRates(t *testing.T) {
	rows := []billingmodels.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
		{UsageKind: models.UsageKindCompute, MatchKey: "e1-large-amd64", MatchMode: billingmodels.UsagePriceBookMatchExact, MicrosPerSecond: 70},
	}

	assert.True(t, billingmodels.UsagePriceBookRatesEqual(rows, billingmodels.CloneUsagePriceBookRates(rows)))
}

func Test__UsagePriceBookRatesEqual__IgnoresOrder(t *testing.T) {
	left := []billingmodels.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 250},
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	right := []billingmodels.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 250},
	}

	assert.True(t, billingmodels.UsagePriceBookRatesEqual(left, right))
}

func Test__UsagePriceBookRatesEqual__DetectsChangedValues(t *testing.T) {
	before := []billingmodels.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	after := billingmodels.CloneUsagePriceBookRates(before)
	after[0].InputCentsPerMillion = 301

	assert.False(t, billingmodels.UsagePriceBookRatesEqual(before, after))
}

func Test__UsagePriceBookRatesEqual__DetectsAddedRows(t *testing.T) {
	before := []billingmodels.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}
	after := append(billingmodels.CloneUsagePriceBookRates(before), billingmodels.UsagePriceBookRate{
		UsageKind: models.UsageKindModel, MatchKey: "lab-model-1", MatchMode: billingmodels.UsagePriceBookMatchPrefix, InputCentsPerMillion: 12,
	})

	assert.False(t, billingmodels.UsagePriceBookRatesEqual(before, after))
}
