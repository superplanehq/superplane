package pricesync

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

// CatalogSource republishes SuperPlane prefix, family, and runner VM rates.
type CatalogSource struct{}

func (CatalogSource) Name() string { return "superplane-catalog" }

func (CatalogSource) ScanModels(context.Context) ([]ModelRate, error) {
	fallbacks := pricebook.CatalogFallbacks()
	rates := make([]ModelRate, 0, len(fallbacks.PrefixRates)+len(fallbacks.FamilyRates))
	for _, item := range fallbacks.PrefixRates {
		rates = append(rates, ModelRate{
			MatchKey:  item.Prefix,
			MatchMode: models.UsagePriceBookMatchPrefix,
			Rate:      item.Rate,
		})
	}
	for _, item := range fallbacks.FamilyRates {
		rates = append(rates, ModelRate{
			MatchKey:  item.Token,
			MatchMode: models.UsagePriceBookMatchFamily,
			Rate:      item.Rate,
		})
	}
	return rates, nil
}

func (CatalogSource) ScanCompute(context.Context) ([]ComputeRate, error) {
	fallbacks := pricebook.CatalogFallbacks()
	rates := make([]ComputeRate, 0, len(fallbacks.ComputeRates))
	for machineType, micros := range fallbacks.ComputeRates {
		rates = append(rates, ComputeRate{
			MachineType:     machineType,
			MicrosPerSecond: micros,
		})
	}
	return rates, nil
}
