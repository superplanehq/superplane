package pricesync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/test/support"
)

type stubModelSource struct {
	name  string
	rates []ModelRate
	err   error
}

func (s stubModelSource) Name() string { return s.name }

func (s stubModelSource) ScanModels(context.Context) ([]ModelRate, error) {
	return s.rates, s.err
}

type stubComputeSource struct {
	name  string
	rates []ComputeRate
	err   error
}

func (s stubComputeSource) Name() string { return s.name }

func (s stubComputeSource) ScanCompute(context.Context) ([]ComputeRate, error) {
	return s.rates, s.err
}

func TestScannerScan_MergesExactOverCatalog(t *testing.T) {
	scanner := NewScanner(Options{
		Models: []ModelSource{
			CatalogSource{},
			stubModelSource{
				name: "openrouter",
				rates: []ModelRate{{
					MatchKey:  "anthropic/claude-sonnet-4",
					MatchMode: models.UsagePriceBookMatchExact,
					Rate:      pricebook.Rate{Input: 321},
				}},
			},
		},
		Compute: CatalogSource{},
	})

	snapshot, err := scanner.Scan(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"superplane-catalog", "openrouter"}, snapshot.Sources)
	require.NotEmpty(t, snapshot.ComputeRates)

	var exact *ModelRate
	var prefix *ModelRate
	for i := range snapshot.ModelRates {
		rate := snapshot.ModelRates[i]
		if rate.MatchKey == "anthropic/claude-sonnet-4" && rate.MatchMode == models.UsagePriceBookMatchExact {
			exact = &rate
		}
		if rate.MatchKey == "claude-sonnet" && rate.MatchMode == models.UsagePriceBookMatchPrefix {
			prefix = &rate
		}
	}
	require.NotNil(t, exact)
	require.NotNil(t, prefix)
	assert.Equal(t, int64(321), exact.Rate.Input)
	assert.Equal(t, int64(300), prefix.Rate.Input)
}

func TestScannerScan_FailsWhenOpenRouterFails(t *testing.T) {
	scanner := NewScanner(Options{
		Models: []ModelSource{
			CatalogSource{},
			stubModelSource{name: "openrouter", err: assert.AnError},
		},
		Compute: CatalogSource{},
	})

	_, err := scanner.Scan(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openrouter")
}

func TestScanAndPublish_WritesVersionAndReloads(t *testing.T) {
	_ = support.Setup(t)
	t.Cleanup(pricebook.Reset)

	now := time.Date(2098, 6, 1, 15, 0, 0, 0, time.UTC)
	scanner := NewScanner(Options{
		Now: func() time.Time { return now },
		Models: []ModelSource{
			stubModelSource{
				name: "openrouter",
				rates: []ModelRate{{
					MatchKey:  "openai/gpt-4o",
					MatchMode: models.UsagePriceBookMatchExact,
					Rate:      pricebook.Rate{Input: 250, Output: 1000},
				}},
			},
		},
		Compute: CatalogSource{},
	})

	tx := database.Conn()
	result, err := ScanAndPublish(context.Background(), tx, scanner)
	require.NoError(t, err)
	assert.Equal(t, "2098-06-01.1", result.Version)
	assert.Equal(t, 1, result.ModelRateCount)
	assert.Greater(t, result.ComputeRateCount, 0)
	assert.Equal(t, int64(2_500_000), pricebook.EstimateMicros("openrouter", "openai/gpt-4o", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, 10*pricebook.MicrosPerSecondE1Large, pricebook.EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))

	second, err := ScanAndPublish(context.Background(), tx, scanner)
	require.NoError(t, err)
	assert.Equal(t, "2098-06-01.2", second.Version)
	assert.Equal(t, result.Version, second.PreviousVersion)

	t.Cleanup(func() {
		_ = tx.Where("version LIKE ?", "2098-06-01.%").Delete(&models.UsagePriceBookRate{})
		_ = tx.Where("version LIKE ?", "2098-06-01.%").Delete(&models.UsagePriceBook{})
	})
}
