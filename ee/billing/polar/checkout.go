package polar

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
)

// Checkout reads follow the Polar API version 2026-10:
//
//   - https://polar.sh/docs/api-reference/2026-10/checkouts/get-checkout-session.md
//     GET /v1/checkouts/{id} returns one checkout, or 404.
//   - https://polar.sh/docs/api-reference/2026-10/checkouts/list-checkout-sessions.md
//     GET /v1/checkouts/ accepts external_customer_id, product_id, status,
//     page, and limit (at most 100). It returns items and pagination.max_page.
//   - The create page documents the metadata that these reads return.

// CheckoutStatus is the status of a Polar checkout session.
type CheckoutStatus string

const (
	CheckoutStatusOpen CheckoutStatus = "open"
	// CheckoutStatusConfirmed means the customer clicked Pay. Polar documents
	// that this does not show that the payment succeeded.
	CheckoutStatusConfirmed CheckoutStatus = "confirmed"
	CheckoutStatusExpired   CheckoutStatus = "expired"
	CheckoutStatusFailed    CheckoutStatus = "failed"
	CheckoutStatusSucceeded CheckoutStatus = "succeeded"
)

const checkoutListPageSize = 100

// CheckoutIntentMetadataKey names the checkout metadata that carries the ID of
// the local checkout intent. It identifies a checkout after a lost response.
const CheckoutIntentMetadataKey = "superplane_checkout_intent_id"

func (s CheckoutStatus) known() bool {
	switch s {
	case CheckoutStatusOpen, CheckoutStatusConfirmed, CheckoutStatusExpired, CheckoutStatusFailed, CheckoutStatusSucceeded:
		return true
	default:
		return false
	}
}

// Payable reports that the customer can still pay the checkout. A confirmed
// checkout stays unresolved until Polar reports its final status.
func (s CheckoutStatus) Payable() bool {
	return s == CheckoutStatusOpen || s == CheckoutStatusConfirmed
}

// GetCheckout reads one checkout session. It rejects a response that has no
// id or a status that this client does not know, so a caller never acts on a
// state that it cannot verify.
func (c *Client) GetCheckout(ctx context.Context, checkoutID string) (*CheckoutSession, error) {
	id := strings.TrimSpace(checkoutID)
	if id == "" {
		return nil, fmt.Errorf("checkout id is required")
	}

	var payload checkoutJSON
	if err := c.get(ctx, "/checkouts/"+url.PathEscape(id), &payload); err != nil {
		return nil, err
	}
	session := payload.toSession()
	if err := validateCheckoutSession(session); err != nil {
		return nil, err
	}
	if session.ID != id {
		return nil, fmt.Errorf("polar checkout response does not match requested id %q", id)
	}
	return session, nil
}

// ListCheckouts lists the checkout sessions of one customer in any status.
// An empty product ID includes all products. Polar returns the newest first.
func (c *Client) ListCheckouts(ctx context.Context, externalCustomerID, productID string) ([]CheckoutSession, error) {
	externalID := strings.TrimSpace(externalCustomerID)
	if externalID == "" {
		return nil, fmt.Errorf("external customer id is required")
	}
	product := strings.TrimSpace(productID)

	var sessions []CheckoutSession
	for page := 1; ; page++ {
		query := url.Values{}
		query.Set("external_customer_id", externalID)
		if product != "" {
			query.Set("product_id", product)
		}
		query.Set("limit", strconv.Itoa(checkoutListPageSize))
		query.Set("page", strconv.Itoa(page))
		var payload listCheckoutsJSON
		if err := c.get(ctx, "/checkouts/?"+query.Encode(), &payload); err != nil {
			return nil, err
		}
		if payload.Items == nil {
			return nil, fmt.Errorf("polar checkouts response is missing items")
		}
		for _, item := range payload.Items {
			session := item.toSession()
			if err := validateCheckoutSession(session); err != nil {
				return nil, err
			}
			sessions = append(sessions, *session)
		}
		if page >= payload.Pagination.MaxPage {
			return sessions, nil
		}
	}
}

func validateCheckoutSession(session *CheckoutSession) error {
	if session.ID == "" {
		return fmt.Errorf("polar checkout response has no id")
	}
	if !session.Status.known() {
		return fmt.Errorf("polar checkout %q has unsupported status %q", session.ID, session.Status)
	}
	return nil
}

// IntentID returns the local checkout intent ID, or an empty string for a
// checkout that SuperPlane opened before intents existed.
func (s *CheckoutSession) IntentID() string {
	return s.Metadata[CheckoutIntentMetadataKey]
}

// IntentState maps a final Polar status to the state of a resolved intent.
// It reports false while the checkout can still be paid.
func (s CheckoutStatus) IntentState() (string, bool) {
	switch s {
	case CheckoutStatusSucceeded:
		return billingmodels.BillingCheckoutStateSucceeded, true
	case CheckoutStatusExpired:
		return billingmodels.BillingCheckoutStateExpired, true
	case CheckoutStatusFailed:
		return billingmodels.BillingCheckoutStateFailed, true
	default:
		return "", false
	}
}

func (c checkoutJSON) toSession() *CheckoutSession {
	return &CheckoutSession{
		ID:         strings.TrimSpace(c.ID),
		URL:        c.URL,
		CustomerID: c.CustomerID,
		ProductID:  c.ProductID,
		Status:     CheckoutStatus(strings.ToLower(strings.TrimSpace(c.Status))),
		ExpiresAt:  c.ExpiresAt.TimePtr(),
		Seats:      c.Seats,
		Metadata:   stringMetadata(c.Metadata),
	}
}

// stringMetadata converts Polar metadata values, which can be strings,
// numbers, or booleans, to strings.
func stringMetadata(metadata map[string]any) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	values := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if text, ok := value.(string); ok {
			values[key] = text
			continue
		}
		values[key] = fmt.Sprint(value)
	}
	return values
}

type listCheckoutsJSON struct {
	Items      []checkoutJSON `json:"items"`
	Pagination struct {
		MaxPage int `json:"max_page"`
	} `json:"pagination"`
}
