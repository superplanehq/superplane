package polar

import (
	"context"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BillingDiagnostics contains independent, read-only Polar observations.
// SubscriptionVerification is the editor lock classification, not a Polar status.
type BillingDiagnostics struct {
	Configured               bool                     `json:"configured"`
	Environment              string                   `json:"environment"`
	CheckedAt                *time.Time               `json:"checked_at"`
	ExternalID               string                   `json:"external_id"`
	CustomerStatus           string                   `json:"customer_status"`
	CustomerID               string                   `json:"customer_id"`
	CustomerURL              string                   `json:"customer_url,omitempty"`
	CustomerEventsURL        string                   `json:"customer_events_url,omitempty"`
	CustomerError            string                   `json:"customer_error,omitempty"`
	CustomerIDMismatch       bool                     `json:"customer_id_mismatch"`
	CardStatus               string                   `json:"card_status"`
	CardError                string                   `json:"card_error,omitempty"`
	SubscriptionVerification string                   `json:"subscription_verification"`
	Subscriptions            []DiagnosticSubscription `json:"subscriptions"`
	SubscriptionError        string                   `json:"subscription_error,omitempty"`
}

type DiagnosticSubscription struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

func UncheckedBillingDiagnostics() BillingDiagnostics {
	state := "not_checked"
	if !Configured() {
		state = "not_configured"
	}
	environment := "sandbox"
	if strings.EqualFold(strings.TrimSpace(os.Getenv("POLAR_ENVIRONMENT")), "production") {
		environment = "production"
	}
	return BillingDiagnostics{
		Configured: Configured(), Environment: environment,
		CustomerStatus: state, CardStatus: state, SubscriptionVerification: state,
		Subscriptions: []DiagnosticSubscription{},
	}
}

// CheckBillingDiagnostics never creates a customer or changes local billing.
// Each section uses its own request timeout so one failure does not hide others.
func CheckBillingDiagnostics(ctx context.Context, orgID uuid.UUID, storedCustomerID string) BillingDiagnostics {
	result := UncheckedBillingDiagnostics()
	result.ExternalID = orgID.String()
	if !result.Configured {
		return result
	}
	now := time.Now().UTC()
	result.CheckedAt = &now
	client := NewClientFromEnv()
	result.checkCustomer(ctx, client, storedCustomerID)
	result.checkCard(ctx, client)
	result.checkSubscriptions(ctx, client, orgID)
	return result
}

func (d *BillingDiagnostics) checkCustomer(ctx context.Context, client *Client, storedID string) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	customer, err := client.GetCustomerByExternalID(ctx, d.ExternalID)
	if IsNotFound(err) {
		d.CustomerStatus = "absent"
		return
	}
	if err != nil || customer == nil || strings.TrimSpace(customer.ID) == "" || customer.ExternalID != d.ExternalID {
		d.CustomerStatus = "failed"
		d.CustomerError = "Could not verify the Polar customer. Check Polar and try again."
		return
	}
	d.CustomerStatus = "found"
	d.CustomerID = customer.ID
	d.CustomerURL = CustomerDashboardURL(customer.ID)
	d.CustomerEventsURL = CustomerEventsDashboardURL(customer.ID)
	d.CustomerIDMismatch = strings.TrimSpace(storedID) != "" && storedID != customer.ID
}

func (d *BillingDiagnostics) checkCard(ctx context.Context, client *Client) {
	// An absent customer proves no saved card. A failed customer lookup does not.
	if d.CustomerStatus == "absent" {
		d.CardStatus = "absent"
		return
	}
	if d.CustomerStatus != "found" {
		d.CardStatus = "unknown"
		d.CardError = "Saved-card status is unknown because the customer lookup failed."
		return
	}
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	hasCard, err := client.hasPaymentMethod(ctx, d.ExternalID)
	if err != nil {
		d.CardStatus = "failed"
		d.CardError = "Could not verify saved cards. Check Polar and try again."
		return
	}
	d.CardStatus = "absent"
	if hasCard {
		d.CardStatus = "usable"
	}
}

func (d *BillingDiagnostics) checkSubscriptions(ctx context.Context, client *Client, orgID uuid.UUID) {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	items, err := client.ListSubscriptions(ctx, d.ExternalID, "")
	if err == nil {
		for _, item := range items {
			d.Subscriptions = append(d.Subscriptions, DiagnosticSubscription{ID: item.ID, Status: item.Status, URL: SubscriptionDashboardURL(item.ID)})
		}
		d.SubscriptionVerification, err = classifyOrganizationSubscriptions(orgID, items)
	}
	if err != nil {
		d.SubscriptionVerification = "failed"
		d.SubscriptionError = "Could not verify Polar subscriptions. Check Polar and try again."
	}
}

func CustomerDashboardURL(customerID string) string {
	if strings.TrimSpace(customerID) == "" {
		return ""
	}
	return dashboardURL() + "/customers/" + url.PathEscape(customerID)
}

func CustomerEventsDashboardURL(customerID string) string {
	if strings.TrimSpace(customerID) == "" {
		return ""
	}
	return CustomerDashboardURL(customerID) + "/events"
}
