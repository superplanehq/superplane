package polar

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	sandboxAPIBaseURL    = "https://sandbox-api.polar.sh/v1"
	productionAPIBaseURL = "https://api.polar.sh/v1"
	creditPackMetadata   = "superplane_credit_pack"
	httpTimeout          = 15 * time.Second

	// Polar removes each API version about nine months after release.
	// Upgrade before removal: see docs/prd/token-usage-and-billing.md.
	defaultAPIVersion = "2026-10"
	apiVersionHeader  = "Polar-Version"
)

var (
	errNotFound              = fmt.Errorf("polar resource not found")
	ErrNotCreditPack         = errors.New("product is not a hosted credit pack")
	ErrRateLimited           = errors.New("polar rate limited")
	ErrConflict              = errors.New("polar customer conflict")
	ErrUnauthorized          = errors.New("polar unauthorized")
	ErrTeamMemberRequired    = errors.New("polar team member required")
	ErrUnsupportedAPIVersion = errors.New("polar api version is not supported")

	// Subscription update failures. Polar leaves the subscription unchanged
	// for each of them.
	ErrPaymentFailed         = errors.New("polar payment failed")
	ErrPaymentActionRequired = errors.New("polar payment needs customer action")
	ErrSubscriptionLocked    = errors.New("polar subscription is locked by another update")
	ErrSubscriptionInactive  = errors.New("polar subscription is canceled or inactive")

	// ErrOutcomeUnknown marks a request that Polar may have applied. The
	// connection failed, timed out, answered 5xx, or returned an unreadable body.
	ErrOutcomeUnknown = errors.New("polar request outcome is unknown")

	// ErrInvalidRequest marks a 422. Polar rejected the request values.
	ErrInvalidRequest = errors.New("polar rejected the request values")
)

// prorationInvoice makes Polar invoice a seat change at once. SuperPlane
// never schedules a seat change for the next period.
const prorationInvoice = "invoice"

// Polar error codes in the "error" field of a response body.
const (
	errorCodePaymentFailed         = "PaymentFailed"
	errorCodePaymentActionRequired = "PaymentActionRequired"
	errorCodeSubscriptionLocked    = "SubscriptionLocked"
	errorCodeAlreadyCanceled       = "AlreadyCanceledSubscription"
	errorCodeInactiveSubscription  = "InactiveSubscription"
)

type Client struct {
	baseURL     string
	accessToken string
	apiVersion  string
	httpClient  *http.Client
}

type ProductKind string

const (
	ProductKindOneTime      ProductKind = "one_time"
	ProductKindSubscription ProductKind = "subscription"
)

type Product struct {
	ID                 string
	Name               string
	AmountCents        int64
	CustomPrice        bool
	Currency           string
	MinimumAmountCents int64
	MaximumAmountCents int64
}

type Customer struct {
	ID         string
	ExternalID string
	Email      string
	Name       string
}

type CreateCustomerInput struct {
	ExternalID string
	Name       string
	OwnerEmail string
}

type CheckoutSession struct {
	ID         string
	URL        string
	CustomerID string
	ProductID  string
	Status     CheckoutStatus
	ExpiresAt  *time.Time
	Seats      *int
	Metadata   map[string]string
}

type CustomerSession struct {
	Token      string
	PortalURL  string
	CustomerID string
}

type CustomerSessionRequest struct {
	CustomerID         string
	ExternalCustomerID string
	MemberID           string
	ExternalMemberID   string
}

type Order struct {
	CheckoutID     string
	Currency       string
	NetAmountCents int64
	TaxAmountCents int64
	ID             string
	CreatedAt      string
	AmountCents    int64
	Status         string
	ProductName    string
}

func Configured() bool {
	return strings.TrimSpace(os.Getenv("POLAR_ACCESS_TOKEN")) != ""
}

func BusinessProductID() string {
	return strings.TrimSpace(os.Getenv("POLAR_BUSINESS_PRODUCT_ID"))
}

func NewClientFromEnv() *Client {
	return NewClient(APIBaseURL(), os.Getenv("POLAR_ACCESS_TOKEN"), nil)
}

func NewClient(baseURL, accessToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: httpTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		accessToken: strings.TrimSpace(accessToken),
		apiVersion:  APIVersion(),
		httpClient:  httpClient,
	}
}

func APIVersion() string {
	if override := strings.TrimSpace(os.Getenv("POLAR_API_VERSION")); override != "" {
		return override
	}
	return defaultAPIVersion
}

func APIBaseURL() string {
	if override := strings.TrimRight(strings.TrimSpace(os.Getenv("POLAR_API_BASE_URL")), "/"); override != "" {
		return override
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("POLAR_ENVIRONMENT")), "production") {
		return productionAPIBaseURL
	}
	return sandboxAPIBaseURL
}

func (c *Client) listProducts(ctx context.Context, kind ProductKind) ([]productJSON, error) {
	var items []productJSON
	page := 1
	for {
		query := url.Values{}
		query.Set("is_archived", "false")
		query.Set("is_recurring", strconv.FormatBool(kind == ProductKindSubscription))
		query.Set("limit", "100")
		query.Set("page", strconv.Itoa(page))
		var payload listProductsJSON
		if err := c.get(ctx, "/products/?"+query.Encode(), &payload); err != nil {
			return nil, err
		}
		items = append(items, payload.Items...)
		if len(payload.Items) == 0 || len(payload.Items) < 100 {
			break
		}
		if payload.Pagination.MaxPage > 0 && page >= payload.Pagination.MaxPage {
			break
		}
		page++
	}

	return items, nil
}

func (c *Client) ListProducts(ctx context.Context, kind ProductKind) ([]Product, error) {
	items, err := c.listProducts(ctx, kind)
	if err != nil {
		return nil, err
	}
	products := make([]Product, 0, len(items))
	for _, item := range items {
		product, ok := catalogProduct(item, kind)
		if !ok {
			continue
		}
		products = append(products, product)
	}
	slices.SortFunc(products, func(left, right Product) int {
		if order := strings.Compare(left.Name, right.Name); order != 0 {
			return order
		}
		return strings.Compare(left.ID, right.ID)
	})
	return products, nil
}

func (c *Client) ListCreditPacks(ctx context.Context) ([]Product, error) {
	items, err := c.listProducts(ctx, ProductKindOneTime)
	if err != nil {
		return nil, err
	}

	packs := make([]Product, 0, len(items))
	for _, item := range items {
		if !isCreditPack(item.Metadata) {
			continue
		}
		pack, ok := creditPackFromProduct(item)
		if !ok {
			continue
		}
		packs = append(packs, pack)
	}
	slices.SortFunc(packs, func(left, right Product) int {
		if left.CustomPrice != right.CustomPrice {
			if left.CustomPrice {
				return 1
			}
			return -1
		}
		return cmp.Compare(left.AmountCents, right.AmountCents)
	})
	return packs, nil
}

func catalogProduct(item productJSON, kind ProductKind) (Product, bool) {
	if kind == ProductKindOneTime {
		product, ok := creditPackFromProduct(item)
		return product, ok && strings.EqualFold(product.Currency, "usd")
	}
	if kind != ProductKindSubscription || !item.IsRecurring || item.IsArchived {
		return Product{}, false
	}
	// Subscription plans can use fixed, per-seat, or other recurring prices.
	if !slices.ContainsFunc(item.Prices, func(price priceJSON) bool {
		return !price.IsArchived && strings.EqualFold(price.currency(), "usd")
	}) {
		return Product{}, false
	}
	return Product{ID: item.ID, Name: item.Name, Currency: "usd"}, true
}

func (c *Client) GetCreditPack(ctx context.Context, productID string) (*Product, error) {
	payload, err := c.getProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !isCreditPack(payload.Metadata) {
		return nil, ErrNotCreditPack
	}
	return checkoutProduct(*payload)
}

// GetCheckoutProduct validates a product linked by an installation admin.
// The local catalog identifies credit packs, so Polar metadata is not required.
func (c *Client) GetCheckoutProduct(ctx context.Context, productID string) (*Product, error) {
	payload, err := c.getProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	return checkoutProduct(*payload)
}

func (c *Client) getProduct(ctx context.Context, productID string) (*productJSON, error) {
	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, fmt.Errorf("product id is required")
	}

	var payload productJSON
	if err := c.get(ctx, "/products/"+url.PathEscape(id), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func checkoutProduct(payload productJSON) (*Product, error) {
	pack, ok := creditPackFromProduct(payload)
	if !ok {
		return nil, fmt.Errorf("%w: an active one-time product with USD pricing is required", ErrNotCreditPack)
	}
	return &pack, nil
}

func (c *Client) GetCustomerByExternalID(ctx context.Context, externalID string) (*Customer, error) {
	var payload customerJSON
	err := c.get(ctx, "/customers/external/"+url.PathEscape(externalID), &payload)
	if err != nil {
		return nil, err
	}
	return payload.toCustomer(), nil
}

func (c *Client) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error) {
	body := map[string]any{
		"external_id": input.ExternalID,
		"type":        "team",
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		body["name"] = name
	}
	if ownerEmail := strings.TrimSpace(input.OwnerEmail); ownerEmail != "" {
		body["owner"] = map[string]any{"email": ownerEmail}
	}
	var payload customerJSON
	err := c.post(ctx, "/customers/", body, &payload)
	if err != nil {
		return nil, err
	}
	return payload.toCustomer(), nil
}

func (c *Client) EnsureCustomer(ctx context.Context, input CreateCustomerInput) (*Customer, error) {
	customer, err := c.GetCustomerByExternalID(ctx, input.ExternalID)
	if err == nil {
		return customer, nil
	}
	if !IsNotFound(err) {
		return nil, err
	}

	customer, err = c.CreateCustomer(ctx, input)
	if err == nil {
		return customer, nil
	}
	if !isCustomerCreateConflict(err) {
		return nil, err
	}

	existing, lookupErr := c.GetCustomerByExternalID(ctx, input.ExternalID)
	if lookupErr == nil {
		return existing, nil
	}
	return nil, fmt.Errorf("%w: polar customer already exists for a different organization", ErrConflict)
}

type CheckoutInput struct {
	ProductID          string
	ExternalCustomerID string
	SuccessURL         string
	CustomerIP         string
	Seats              *int
	// Metadata is stored on the Polar checkout and returned by get and list.
	Metadata map[string]string
}

func (c *Client) CreateCheckout(ctx context.Context, input CheckoutInput) (*CheckoutSession, error) {
	body := map[string]any{
		"products":             []string{input.ProductID},
		"external_customer_id": input.ExternalCustomerID,
		"success_url":          input.SuccessURL,
	}
	if strings.TrimSpace(input.CustomerIP) != "" {
		body["customer_ip_address"] = input.CustomerIP
	}
	if input.Seats != nil {
		body["seats"] = *input.Seats
		body["min_seats"] = *input.Seats
		body["max_seats"] = *input.Seats
	}
	if len(input.Metadata) > 0 {
		body["metadata"] = input.Metadata
	}
	session, err := c.createCheckout(ctx, body)
	if err != nil {
		return nil, err
	}
	// Polar created the checkout, so the identity must come from its response.
	if strings.TrimSpace(session.ID) == "" {
		return nil, classified(errors.New("polar checkout did not return an id"), ErrOutcomeUnknown)
	}
	return session, nil
}

func (c *Client) CreateCreditCheckout(ctx context.Context, productID, externalCustomerID, successURL, customerIP string, quote *CreditPurchaseQuote) (*CheckoutSession, error) {
	if quote == nil || quote.CreditAmountCents <= 0 || quote.CreditAmountCents > quote.SubtotalCents ||
		quote.SubtotalCents < minimumUSDPriceCents || quote.SubtotalCents > maximumUSDPriceCents ||
		quote.Currency != "usd" || quote.ProcessingFeeCents != quote.SubtotalCents-quote.CreditAmountCents {
		return nil, fmt.Errorf("credit purchase quote is invalid")
	}
	body := map[string]any{
		"products":             []string{productID},
		"external_customer_id": externalCustomerID,
		"success_url":          successURL,
		"metadata":             quote.metadata(),
		"prices": map[string]any{
			productID: []map[string]any{{
				"amount_type":    "fixed",
				"price_amount":   quote.SubtotalCents,
				"price_currency": "usd",
				"tax_behavior":   "exclusive",
			}},
		},
	}
	if strings.TrimSpace(customerIP) != "" {
		body["customer_ip_address"] = customerIP
	}
	return c.createCheckout(ctx, body)
}

func (c *Client) createCheckout(ctx context.Context, body map[string]any) (*CheckoutSession, error) {
	var payload checkoutJSON
	if err := c.post(ctx, "/checkouts/", body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.URL) == "" {
		return nil, classified(errors.New("polar checkout did not return a url"), ErrOutcomeUnknown)
	}
	return payload.toSession(), nil
}

func (c *Client) CreateCustomerSession(ctx context.Context, req CustomerSessionRequest) (*CustomerSession, error) {
	body := map[string]any{}
	if externalID := strings.TrimSpace(req.ExternalCustomerID); externalID != "" {
		body["external_customer_id"] = externalID
	} else {
		body["customer_id"] = strings.TrimSpace(req.CustomerID)
	}
	if memberID := strings.TrimSpace(req.MemberID); memberID != "" {
		body["member_id"] = memberID
	}
	if externalMemberID := strings.TrimSpace(req.ExternalMemberID); externalMemberID != "" {
		body["external_member_id"] = externalMemberID
	}
	var payload customerSessionJSON
	if err := c.post(ctx, "/customer-sessions/", body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.CustomerPortalURL) == "" {
		return nil, fmt.Errorf("polar customer session did not return a portal url")
	}
	return &CustomerSession{
		Token:      payload.Token,
		PortalURL:  payload.CustomerPortalURL,
		CustomerID: payload.CustomerID,
	}, nil
}

func (c *Client) GetOwnerMember(ctx context.Context, externalCustomerID, customerID string) (string, error) {
	query := url.Values{}
	query.Set("role", "owner")
	query.Set("limit", "1")
	if externalID := strings.TrimSpace(externalCustomerID); externalID != "" {
		query.Set("external_customer_id", externalID)
	} else {
		id := strings.TrimSpace(customerID)
		if id == "" {
			return "", fmt.Errorf("customer id is required")
		}
		query.Set("customer_id", id)
	}

	var payload listMembersJSON
	if err := c.get(ctx, "/members/?"+query.Encode(), &payload); err != nil {
		return "", err
	}
	for _, item := range payload.Items {
		if id := strings.TrimSpace(item.ID); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: polar owner member not found", errNotFound)
}

func (c *Client) ListSubscriptions(ctx context.Context, externalCustomerID, productID string) ([]SubscriptionData, error) {
	externalID := strings.TrimSpace(externalCustomerID)
	if externalID == "" {
		return nil, fmt.Errorf("external customer id is required")
	}

	var items []SubscriptionData
	page := 1
	for {
		query := url.Values{}
		query.Set("external_customer_id", externalID)
		query.Set("limit", "100")
		query.Set("page", strconv.Itoa(page))
		if id := strings.TrimSpace(productID); id != "" {
			query.Set("product_id", id)
		}
		var payload listSubscriptionsJSON
		if err := c.get(ctx, "/subscriptions/?"+query.Encode(), &payload); err != nil {
			return nil, err
		}
		if payload.Items == nil {
			return nil, fmt.Errorf("polar subscriptions response is missing items")
		}
		for _, item := range payload.Items {
			if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Status) == "" {
				return nil, fmt.Errorf("polar subscriptions response has an invalid subscription")
			}
		}
		items = append(items, payload.Items...)
		if payload.Pagination.MaxPage > page {
			page++
			continue
		}
		if len(payload.Items) < 100 {
			break
		}
		if payload.Pagination.MaxPage > 0 && page >= payload.Pagination.MaxPage {
			break
		}
		page++
	}

	return items, nil
}

func (c *Client) CancelSubscriptionAtPeriodEnd(ctx context.Context, subscriptionID string) (*SubscriptionData, error) {
	return c.setSubscriptionCancelAtPeriodEnd(ctx, subscriptionID, true)
}

func (c *Client) ResumeSubscription(ctx context.Context, subscriptionID string) (*SubscriptionData, error) {
	return c.setSubscriptionCancelAtPeriodEnd(ctx, subscriptionID, false)
}

func (c *Client) setSubscriptionCancelAtPeriodEnd(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) (*SubscriptionData, error) {
	id := strings.TrimSpace(subscriptionID)
	if id == "" {
		return nil, fmt.Errorf("subscription id is required")
	}

	var payload SubscriptionData
	err := c.patch(ctx, "/subscriptions/"+url.PathEscape(id), map[string]any{
		"cancel_at_period_end": cancelAtPeriodEnd,
	}, &payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.ID) == "" {
		return nil, fmt.Errorf("polar subscription update did not return an id")
	}
	return &payload, nil
}

// UpdateSubscriptionSeats sets the absolute seat count and invoices the
// prorated difference at once. Polar applies the change only when that payment
// succeeds. The renewal date does not change.
func (c *Client) UpdateSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) (*SubscriptionData, error) {
	id := strings.TrimSpace(subscriptionID)
	if id == "" {
		return nil, fmt.Errorf("subscription id is required")
	}

	var payload SubscriptionData
	err := c.patch(ctx, "/subscriptions/"+url.PathEscape(id), map[string]any{
		"seats":              seats,
		"proration_behavior": prorationInvoice,
	}, &payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.ID) == "" {
		return nil, classified(fmt.Errorf("polar subscription update did not return an id"), ErrOutcomeUnknown)
	}
	return &payload, nil
}

func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionData, error) {
	id := strings.TrimSpace(subscriptionID)
	if id == "" {
		return nil, fmt.Errorf("subscription id is required")
	}

	var payload SubscriptionData
	if err := c.get(ctx, "/subscriptions/"+url.PathEscape(id), &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.ID) == "" {
		return nil, fmt.Errorf("polar subscription response did not return an id")
	}
	return &payload, nil
}

func (c *Client) ListOrders(ctx context.Context, externalCustomerID string) ([]Order, error) {
	externalID := strings.TrimSpace(externalCustomerID)
	if externalID == "" {
		return nil, fmt.Errorf("external customer id is required")
	}

	var items []orderJSON
	page := 1
	for {
		query := url.Values{}
		query.Set("external_customer_id", externalID)
		query.Set("limit", "100")
		query.Set("page", strconv.Itoa(page))
		var payload listOrdersJSON
		if err := c.get(ctx, "/orders/?"+query.Encode(), &payload); err != nil {
			return nil, err
		}
		items = append(items, payload.Items...)
		if len(payload.Items) == 0 || len(payload.Items) < 100 {
			break
		}
		if payload.Pagination.MaxPage > 0 && page >= payload.Pagination.MaxPage {
			break
		}
		page++
	}

	orders := make([]Order, 0, len(items))
	for _, item := range items {
		orders = append(orders, item.toOrder())
	}
	return orders, nil
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	return c.do(ctx, http.MethodGet, path, nil, dest)
}

func (c *Client) post(ctx context.Context, path string, body any, dest any) error {
	return c.do(ctx, http.MethodPost, path, body, dest)
}

func (c *Client) patch(ctx context.Context, path string, body any, dest any) error {
	return c.do(ctx, http.MethodPatch, path, body, dest)
}

func (c *Client) do(ctx context.Context, method, path string, body any, dest any) error {
	if c.accessToken == "" {
		return fmt.Errorf("hosted billing is not configured")
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SuperPlane")
	req.Header.Set(apiVersionHeader, c.apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return classified(err, ErrOutcomeUnknown)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return classified(err, ErrOutcomeUnknown)
	}
	servedVersion := strings.TrimSpace(resp.Header.Get(apiVersionHeader))
	// Polar echoes Polar-Version on every response it routes. A 404 without it
	// means Polar rejected the pinned version, not that the resource is missing.
	if resp.StatusCode == http.StatusNotFound && servedVersion == "" {
		return fmt.Errorf("%w: %s", ErrUnsupportedAPIVersion, c.apiVersion)
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", errNotFound, strings.TrimSpace(string(payload)))
	}
	if servedVersion != "" && servedVersion != c.apiVersion {
		log.WithFields(log.Fields{
			"polar_api_version_pinned": c.apiVersion,
			"polar_api_version_served": servedVersion,
			"polar_path":               path,
		}).Warn("polar served a different api version than the pinned one")
	}
	errorCode := responseErrorCode(payload)
	if resp.StatusCode == http.StatusConflict {
		if errorCode == errorCodeSubscriptionLocked {
			return fmt.Errorf("%w: %w: %s", ErrSubscriptionLocked, ErrConflict, strings.TrimSpace(string(payload)))
		}
		return fmt.Errorf("%w: %s", ErrConflict, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		switch errorCode {
		case errorCodePaymentFailed:
			return fmt.Errorf("%w: %s", ErrPaymentFailed, strings.TrimSpace(string(payload)))
		case errorCodePaymentActionRequired:
			return fmt.Errorf("%w: %s", ErrPaymentActionRequired, strings.TrimSpace(string(payload)))
		}
	}
	if resp.StatusCode == http.StatusForbidden && isSubscriptionInactiveCode(errorCode) {
		return fmt.Errorf("%w: %s", ErrSubscriptionInactive, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After"))
		if retryAfter != "" {
			return fmt.Errorf("%w: retry after %s", ErrRateLimited, retryAfter)
		}
		return ErrRateLimited
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: %s", ErrUnauthorized, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode == http.StatusUnprocessableEntity && isTeamMemberRequiredPayload(payload) {
		return fmt.Errorf("%w: %s", ErrTeamMemberRequired, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("polar api %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
			return classified(statusErr, ErrOutcomeUnknown)
		}
		if resp.StatusCode == http.StatusUnprocessableEntity {
			return classified(statusErr, ErrInvalidRequest)
		}
		return statusErr
	}
	if dest == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, dest); err != nil {
		return classified(err, ErrOutcomeUnknown)
	}
	return nil
}

// classifiedError keeps the original message and also matches a sentinel.
type classifiedError struct {
	err   error
	class error
}

func classified(err, class error) error {
	return &classifiedError{err: err, class: class}
}

func (e *classifiedError) Error() string {
	return e.err.Error()
}

func (e *classifiedError) Unwrap() error {
	return e.err
}

func (e *classifiedError) Is(target error) bool {
	return target == e.class
}

func responseErrorCode(payload []byte) string {
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.Error)
}

func isSubscriptionInactiveCode(code string) bool {
	return code == errorCodeAlreadyCanceled || code == errorCodeInactiveSubscription
}

func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, errNotFound)
}

func IsConflict(err error) bool {
	return err != nil && errors.Is(err, ErrConflict)
}

func IsRateLimited(err error) bool {
	return err != nil && errors.Is(err, ErrRateLimited)
}

func IsUnauthorized(err error) bool {
	return err != nil && errors.Is(err, ErrUnauthorized)
}

func IsPaymentFailed(err error) bool {
	return err != nil && errors.Is(err, ErrPaymentFailed)
}

func IsPaymentActionRequired(err error) bool {
	return err != nil && errors.Is(err, ErrPaymentActionRequired)
}

func IsSubscriptionLocked(err error) bool {
	return err != nil && errors.Is(err, ErrSubscriptionLocked)
}

func IsSubscriptionInactive(err error) bool {
	return err != nil && errors.Is(err, ErrSubscriptionInactive)
}

func IsInvalidRequest(err error) bool {
	return err != nil && errors.Is(err, ErrInvalidRequest)
}

// IsOutcomeUnknown reports that Polar may have applied the request.
func IsOutcomeUnknown(err error) bool {
	return err != nil && errors.Is(err, ErrOutcomeUnknown)
}

func IsUnsupportedAPIVersion(err error) bool {
	return err != nil && errors.Is(err, ErrUnsupportedAPIVersion)
}

func IsTeamMemberRequired(err error) bool {
	return err != nil && errors.Is(err, ErrTeamMemberRequired)
}

func isTeamMemberRequiredPayload(payload []byte) bool {
	return strings.Contains(strings.ToLower(string(payload)), "member_id is required")
}

func isCustomerCreateConflict(err error) bool {
	if IsConflict(err) {
		return true
	}
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

type listProductsJSON struct {
	Items      []productJSON `json:"items"`
	Pagination struct {
		MaxPage int `json:"max_page"`
	} `json:"pagination"`
}

type productJSON struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	IsRecurring bool           `json:"is_recurring"`
	IsArchived  bool           `json:"is_archived"`
	Metadata    map[string]any `json:"metadata"`
	Prices      []priceJSON    `json:"prices"`
}

func creditPackFromProduct(item productJSON) (Product, bool) {
	if item.IsRecurring || item.IsArchived {
		return Product{}, false
	}
	for _, price := range item.Prices {
		if !price.IsArchived && price.isFixed() && price.PriceAmount > 0 {
			return Product{
				ID: item.ID, Name: item.Name, AmountCents: price.PriceAmount, Currency: price.currency(),
			}, true
		}
	}
	for _, price := range item.Prices {
		if !price.IsArchived && price.isCustom() {
			return Product{
				ID: item.ID, Name: item.Name, CustomPrice: true, Currency: price.currency(),
				MinimumAmountCents: price.MinimumAmount, MaximumAmountCents: price.MaximumAmount,
			}, true
		}
	}
	return Product{}, false
}

func (p productJSON) faceValueCents() int64 {
	for _, price := range p.Prices {
		if !price.isFixed() {
			continue
		}
		if price.PriceAmount > 0 {
			return price.PriceAmount
		}
	}
	return 0
}

type priceJSON struct {
	AmountType    string `json:"amount_type"`
	PriceAmount   int64  `json:"price_amount"`
	PriceCurrency string `json:"price_currency"`
	MinimumAmount int64  `json:"minimum_amount"`
	MaximumAmount int64  `json:"maximum_amount"`
	IsArchived    bool   `json:"is_archived"`
}

func (p priceJSON) currency() string {
	currency := strings.ToLower(strings.TrimSpace(p.PriceCurrency))
	if currency == "" {
		return "usd"
	}
	return currency
}

func (p priceJSON) isCustom() bool {
	return strings.EqualFold(strings.TrimSpace(p.AmountType), "custom")
}

func (p priceJSON) isFixed() bool {
	amountType := strings.TrimSpace(p.AmountType)
	return amountType == "" || strings.EqualFold(amountType, "fixed")
}

type orderItemJSON struct {
	Amount       int64        `json:"amount"`
	ProductPrice priceJSON    `json:"product_price"`
	Product      OrderProduct `json:"product"`
}

type customerJSON struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
}

func (c customerJSON) toCustomer() *Customer {
	return &Customer{
		ID:         c.ID,
		ExternalID: c.ExternalID,
		Email:      c.Email,
		Name:       c.Name,
	}
}

type checkoutJSON struct {
	ID         string         `json:"id"`
	URL        string         `json:"url"`
	CustomerID string         `json:"customer_id"`
	ProductID  string         `json:"product_id"`
	Status     string         `json:"status"`
	ExpiresAt  polarTime      `json:"expires_at"`
	Seats      *int           `json:"seats"`
	Metadata   map[string]any `json:"metadata"`
}

type customerSessionJSON struct {
	Token             string `json:"token"`
	CustomerPortalURL string `json:"customer_portal_url"`
	CustomerID        string `json:"customer_id"`
}

type listSubscriptionsJSON struct {
	Items      []SubscriptionData `json:"items"`
	Pagination struct {
		MaxPage int `json:"max_page"`
	} `json:"pagination"`
}

type listOrdersJSON struct {
	Items      []orderJSON `json:"items"`
	Pagination struct {
		MaxPage int `json:"max_page"`
	} `json:"pagination"`
}

type listMembersJSON struct {
	Items []memberJSON `json:"items"`
}

type memberJSON struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type orderJSON struct {
	CheckoutID  string `json:"checkout_id"`
	Currency    string `json:"currency"`
	NetAmount   int64  `json:"net_amount"`
	TaxAmount   int64  `json:"tax_amount"`
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
	TotalAmount int64  `json:"total_amount"`
	Description string `json:"description"`
	Product     *struct {
		Name string `json:"name"`
	} `json:"product"`
}

func (o orderJSON) toOrder() Order {
	name := strings.TrimSpace(o.Description)
	if o.Product != nil {
		if productName := strings.TrimSpace(o.Product.Name); productName != "" {
			name = productName
		}
	}
	return Order{
		CheckoutID:     o.CheckoutID,
		Currency:       o.Currency,
		NetAmountCents: o.NetAmount,
		TaxAmountCents: o.TaxAmount,
		ID:             o.ID,
		CreatedAt:      o.CreatedAt,
		AmountCents:    o.TotalAmount,
		Status:         o.Status,
		ProductName:    name,
	}
}

func isCreditPack(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	value, ok := metadata[creditPackMetadata]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
