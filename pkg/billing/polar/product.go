package polar

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// ProductDetails describes catalog pricing without changing the credit-pack API.
type ProductDetails struct {
	ID                     string         `json:"id"`
	Name                   string         `json:"name"`
	IsArchived             bool           `json:"is_archived"`
	IsRecurring            bool           `json:"is_recurring"`
	RecurringInterval      string         `json:"recurring_interval"`
	RecurringIntervalCount int            `json:"recurring_interval_count"`
	Prices                 []ProductPrice `json:"prices"`
}

type ProductPrice struct {
	AmountType    string            `json:"amount_type"`
	IsArchived    bool              `json:"is_archived"`
	PriceCurrency string            `json:"price_currency"`
	SeatTiers     *ProductSeatTiers `json:"seat_tiers"`
}

type ProductSeatTiers struct {
	Tiers []ProductSeatTier `json:"tiers"`
}

type ProductSeatTier struct {
	MinSeats     int    `json:"min_seats"`
	MaxSeats     *int   `json:"max_seats"`
	PricePerSeat *int64 `json:"price_per_seat"`
}

func (c *Client) GetProduct(ctx context.Context, productID string) (*ProductDetails, error) {
	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, fmt.Errorf("product id is required")
	}
	var product ProductDetails
	if err := c.get(ctx, "/products/"+url.PathEscape(id), &product); err != nil {
		return nil, err
	}
	if product.ID != id || product.Prices == nil {
		return nil, fmt.Errorf("polar product response is incomplete")
	}
	return &product, nil
}
