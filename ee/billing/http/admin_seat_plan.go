package billinghttp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func validateSeatPlanProduct(ctx context.Context, plan *billingmodels.BillingPlan) (int, error) {
	if !plan.PerSeat || plan.PolarProductID == nil || strings.TrimSpace(*plan.PolarProductID) == "" {
		return 0, nil
	}
	if !polar.Configured() {
		return http.StatusBadRequest, errors.New("Configure Polar before linking a per-seat plan to a product")
	}
	product, err := polar.NewClientFromEnv().GetProduct(ctx, *plan.PolarProductID)
	if polar.IsNotFound(err) {
		return http.StatusBadRequest, errors.New("Polar product not found. Check the product ID and Polar environment")
	}
	if err != nil {
		middleware.SetServerError(ctx, err, nil)
		return http.StatusBadGateway, errors.New("Could not validate the Polar product. Check the Polar configuration and try again")
	}
	if err := validateSeatProductPricing(product, plan); err != nil {
		return http.StatusBadRequest, err
	}
	return 0, nil
}

func validateSeatProductPricing(product *polar.ProductDetails, plan *billingmodels.BillingPlan) error {
	if product.IsArchived {
		return errors.New("Polar product is archived. Select an active product")
	}
	if !product.IsRecurring || product.RecurringInterval != "month" || product.RecurringIntervalCount != 1 {
		return errors.New("Polar product must be a monthly subscription with interval count 1")
	}
	var activePrices []polar.ProductPrice
	for _, price := range product.Prices {
		if !price.IsArchived {
			activePrices = append(activePrices, price)
		}
	}
	if len(activePrices) != 1 || activePrices[0].AmountType != "seat_based" || activePrices[0].PriceCurrency != "usd" {
		return errors.New("Polar product must have one active USD seat-based price and no additional charges")
	}
	seatTiers := activePrices[0].SeatTiers
	if seatTiers == nil || len(seatTiers.Tiers) != 1 {
		return errors.New("Polar price must have one fixed-rate seat tier. Variable tiers are not supported")
	}
	tier := seatTiers.Tiers[0]
	if tier.MinSeats != 1 || (plan.SeatLimit != nil && tier.MaxSeats != nil && *tier.MaxSeats < *plan.SeatLimit) {
		return errors.New("Polar price tier must cover seats 1 through the catalog maximum")
	}
	if tier.PricePerSeat == nil || *tier.PricePerSeat != plan.PriceCents {
		return errors.New("Polar price per seat must match the catalog price. Create a new plan and product for price changes")
	}
	return nil
}
