package polar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"gorm.io/gorm"
)

// PaymentMethodSnapshot contains only fields needed to assess availability.
// Complete is false for a missing customer. Failed lists return an error.
type PaymentMethodSnapshot struct {
	ReadStartedAt time.Time
	ObservedAt    time.Time
	Complete      bool
	Methods       []PaymentMethodAvailability
}

type PaymentMethodAvailability struct {
	ID      string
	Usable  bool
	Expired bool
}

func (s PaymentMethodSnapshot) HasUsableMethod() bool {
	for _, method := range s.Methods {
		if method.Usable {
			return true
		}
	}
	return false
}

// HasPaymentMethod returns as soon as a usable card is confirmed.
func (c *Client) HasPaymentMethod(ctx context.Context, externalID string) (bool, error) {
	hasCard, err := c.hasPaymentMethod(ctx, externalID)
	if IsNotFound(err) {
		return false, nil
	}
	return hasCard, err
}

func (c *Client) hasPaymentMethod(ctx context.Context, externalID string) (bool, error) {
	now := time.Now().UTC()
	for page := 1; ; page++ {
		methods, maxPage, err := c.paymentMethodsPage(ctx, externalID, page, now)
		if err != nil {
			return false, err
		}
		for _, method := range methods {
			if method.Usable {
				return true, nil
			}
		}
		if page >= maxPage {
			return false, nil
		}
	}
}

// PaymentMethods reads every page before accepting a complete observation.
func (c *Client) PaymentMethods(ctx context.Context, externalID string) (PaymentMethodSnapshot, error) {
	snapshot := PaymentMethodSnapshot{ReadStartedAt: time.Now().UTC()}
	for page := 1; ; page++ {
		methods, maxPage, err := c.paymentMethodsPage(ctx, externalID, page, snapshot.ReadStartedAt)
		if err != nil {
			if IsNotFound(err) {
				return PaymentMethodSnapshot{ReadStartedAt: snapshot.ReadStartedAt}, nil
			}
			return PaymentMethodSnapshot{}, err
		}
		snapshot.Methods = append(snapshot.Methods, methods...)
		if page >= maxPage {
			snapshot.Complete = true
			snapshot.ObservedAt = time.Now().UTC()
			return snapshot, nil
		}
	}
}

func (c *Client) paymentMethodsPage(ctx context.Context, externalID string, page int, now time.Time) ([]PaymentMethodAvailability, int, error) {
	var result struct {
		Items []struct {
			ID       string          `json:"id"`
			Type     string          `json:"type"`
			Metadata json.RawMessage `json:"method_metadata"`
		} `json:"items"`
		Pagination struct {
			MaxPage *int `json:"max_page"`
		} `json:"pagination"`
	}
	path := fmt.Sprintf("/customers/external/%s/payment-methods?limit=100&page=%d", url.PathEscape(externalID), page)
	if err := c.get(ctx, path, &result); err != nil {
		return nil, 0, err
	}
	if result.Items == nil {
		return nil, 0, fmt.Errorf("polar payment methods response is missing items")
	}
	if result.Pagination.MaxPage == nil || *result.Pagination.MaxPage < 0 {
		return nil, 0, fmt.Errorf("payment method list has invalid pagination")
	}
	methods := make([]PaymentMethodAvailability, 0, len(result.Items))
	for _, method := range result.Items {
		if method.ID == "" {
			continue
		}
		methods = append(methods, paymentMethodAvailability(method.ID, method.Type, method.Metadata, now))
	}
	return methods, *result.Pagination.MaxPage, nil
}

func paymentMethodAvailability(id, methodType string, raw json.RawMessage, now time.Time) PaymentMethodAvailability {
	method := PaymentMethodAvailability{ID: id}
	if methodType == "kr_card" {
		method.Usable = true
		return method
	}
	if methodType != "card" {
		return method
	}
	var metadata struct {
		Month int `json:"exp_month"`
		Year  int `json:"exp_year"`
	}
	if json.Unmarshal(raw, &metadata) != nil || metadata.Month < 1 || metadata.Month > 12 || metadata.Year < 1 || metadata.Year > 9999 {
		return method
	}
	now = now.UTC()
	method.Expired = metadata.Year < now.Year() || metadata.Year == now.Year() && metadata.Month < int(now.Month())
	method.Usable = !method.Expired
	return method
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
