package polar

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"gorm.io/gorm"
)

// HasPaymentMethod reads saved cards from Polar, so removing a card blocks the next start.
func (c *Client) HasPaymentMethod(ctx context.Context, externalID string) (bool, error) {
	hasCard, err := c.hasPaymentMethod(ctx, externalID)
	if IsNotFound(err) {
		return false, nil
	}
	return hasCard, err
}

func (c *Client) hasPaymentMethod(ctx context.Context, externalID string) (bool, error) {
	for page := 1; ; page++ {
		var result struct {
			Items []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Metadata struct {
					Month int `json:"exp_month"`
					Year  int `json:"exp_year"`
				} `json:"method_metadata"`
			} `json:"items"`
			Pagination struct {
				MaxPage int `json:"max_page"`
			} `json:"pagination"`
		}
		path := fmt.Sprintf("/customers/external/%s/payment-methods?limit=100&page=%d", url.PathEscape(externalID), page)
		if err := c.get(ctx, path, &result); err != nil {
			return false, err
		}
		if result.Items == nil {
			return false, fmt.Errorf("polar payment methods response is missing items")
		}
		now := time.Now().UTC()
		for _, method := range result.Items {
			if method.ID == "" {
				continue
			}
			if method.Type == "kr_card" {
				return true, nil
			}
			if method.Type == "card" && method.Metadata.Month >= 1 && method.Metadata.Month <= 12 &&
				(method.Metadata.Year > now.Year() || method.Metadata.Year == now.Year() && method.Metadata.Month >= int(now.Month())) {
				return true, nil
			}
		}
		if page >= result.Pagination.MaxPage || len(result.Items) == 0 {
			return false, nil
		}
	}
}

func AssertOrganizationPaymentMethod(tx *gorm.DB, orgID uuid.UUID) error {
	if !billingmodels.BillingEnabled() {
		return nil
	}
	plan, err := billingmodels.ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if !plan.Catalog.RequirePaymentMethod {
		return nil
	}
	hasCard, err := NewClientFromEnv().HasPaymentMethod(tx.Statement.Context, orgID.String())
	if err != nil {
		return fmt.Errorf("could not verify the organization payment method: %w", err)
	}
	if !hasCard {
		return billingmodels.ErrPaymentMethodRequired
	}
	return nil
}
