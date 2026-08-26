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
)

const (
	sandboxAPIBaseURL    = "https://sandbox-api.polar.sh/v1"
	productionAPIBaseURL = "https://api.polar.sh/v1"
	creditPackMetadata   = "superplane_credit_pack"
	httpTimeout          = 15 * time.Second
)

var (
	errNotFound      = fmt.Errorf("polar resource not found")
	ErrNotCreditPack = errors.New("product is not a hosted credit pack")
)

type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

type Product struct {
	ID          string
	Name        string
	AmountCents int64
}

type Customer struct {
	ID         string
	ExternalID string
	Email      string
}

type CheckoutSession struct {
	URL        string
	CustomerID string
}

type CustomerSession struct {
	PortalURL  string
	CustomerID string
}

func Configured() bool {
	return strings.TrimSpace(os.Getenv("POLAR_ACCESS_TOKEN")) != ""
}

func NewClientFromEnv() *Client {
	return NewClient(APIBaseURL(), os.Getenv("POLAR_ACCESS_TOKEN"), nil)
}

func NewClient(baseURL, accessToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: httpTimeout}
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		accessToken: strings.TrimSpace(accessToken),
		httpClient:  httpClient,
	}
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

func (c *Client) ListCreditPacks(ctx context.Context) ([]Product, error) {
	var items []productJSON
	page := 1
	for {
		query := url.Values{}
		query.Set("is_archived", "false")
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

	packs := make([]Product, 0, len(items))
	for _, item := range items {
		if !isCreditPack(item.Metadata) {
			continue
		}
		amount := item.faceValueCents()
		if amount <= 0 {
			continue
		}
		packs = append(packs, Product{
			ID:          item.ID,
			Name:        item.Name,
			AmountCents: amount,
		})
	}
	slices.SortFunc(packs, func(left, right Product) int {
		return cmp.Compare(left.AmountCents, right.AmountCents)
	})
	return packs, nil
}

func (c *Client) GetCreditPack(ctx context.Context, productID string) (*Product, error) {
	id := strings.TrimSpace(productID)
	if id == "" {
		return nil, fmt.Errorf("product id is required")
	}

	var payload productJSON
	if err := c.get(ctx, "/products/"+url.PathEscape(id), &payload); err != nil {
		return nil, err
	}
	if !isCreditPack(payload.Metadata) {
		return nil, ErrNotCreditPack
	}
	amount := payload.faceValueCents()
	if amount <= 0 {
		return nil, fmt.Errorf("credit pack face value is missing")
	}
	return &Product{
		ID:          payload.ID,
		Name:        payload.Name,
		AmountCents: amount,
	}, nil
}

func (c *Client) GetCustomerByExternalID(ctx context.Context, externalID string) (*Customer, error) {
	var payload customerJSON
	err := c.get(ctx, "/customers/external/"+url.PathEscape(externalID), &payload)
	if err != nil {
		return nil, err
	}
	return payload.toCustomer(), nil
}

func (c *Client) CreateCustomer(ctx context.Context, externalID, email string) (*Customer, error) {
	var payload customerJSON
	err := c.post(ctx, "/customers", map[string]any{
		"external_id": externalID,
		"email":       email,
	}, &payload)
	if err != nil {
		return nil, err
	}
	return payload.toCustomer(), nil
}

func (c *Client) EnsureCustomer(ctx context.Context, externalID, email string) (*Customer, error) {
	customer, err := c.GetCustomerByExternalID(ctx, externalID)
	if err == nil {
		return customer, nil
	}
	if !IsNotFound(err) {
		return nil, err
	}
	return c.CreateCustomer(ctx, externalID, email)
}

func (c *Client) CreateCheckout(ctx context.Context, productID, externalCustomerID, email, successURL, customerIP string) (*CheckoutSession, error) {
	body := map[string]any{
		"products":             []string{productID},
		"external_customer_id": externalCustomerID,
		"success_url":          successURL,
	}
	if strings.TrimSpace(email) != "" {
		body["customer_email"] = email
	}
	if strings.TrimSpace(customerIP) != "" {
		body["customer_ip_address"] = customerIP
	}

	var payload checkoutJSON
	if err := c.post(ctx, "/checkouts/", body, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.URL) == "" {
		return nil, fmt.Errorf("polar checkout did not return a url")
	}
	return &CheckoutSession{
		URL:        payload.URL,
		CustomerID: payload.CustomerID,
	}, nil
}

func (c *Client) CreateCustomerSession(ctx context.Context, customerID string) (*CustomerSession, error) {
	var payload customerSessionJSON
	if err := c.post(ctx, "/customer-sessions/", map[string]any{
		"customer_id": customerID,
	}, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.CustomerPortalURL) == "" {
		return nil, fmt.Errorf("polar customer session did not return a portal url")
	}
	return &CustomerSession{
		PortalURL:  payload.CustomerPortalURL,
		CustomerID: payload.CustomerID,
	}, nil
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	return c.do(ctx, http.MethodGet, path, nil, dest)
}

func (c *Client) post(ctx context.Context, path string, body any, dest any) error {
	return c.do(ctx, http.MethodPost, path, body, dest)
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
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", errNotFound, strings.TrimSpace(string(payload)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("polar api %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if dest == nil || len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, dest)
}

func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, errNotFound)
}

type listProductsJSON struct {
	Items      []productJSON `json:"items"`
	Pagination struct {
		MaxPage int `json:"max_page"`
	} `json:"pagination"`
}

type productJSON struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata"`
	Prices   []priceJSON    `json:"prices"`
}

func (p productJSON) faceValueCents() int64 {
	for _, price := range p.Prices {
		if price.AmountType != "" && price.AmountType != "fixed" {
			continue
		}
		if price.PriceAmount > 0 {
			return price.PriceAmount
		}
	}
	return 0
}

type priceJSON struct {
	AmountType  string `json:"amount_type"`
	PriceAmount int64  `json:"price_amount"`
}

type customerJSON struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	Email      string `json:"email"`
}

func (c customerJSON) toCustomer() *Customer {
	return &Customer{
		ID:         c.ID,
		ExternalID: c.ExternalID,
		Email:      c.Email,
	}
}

type checkoutJSON struct {
	URL        string `json:"url"`
	CustomerID string `json:"customer_id"`
}

type customerSessionJSON struct {
	CustomerPortalURL string `json:"customer_portal_url"`
	CustomerID        string `json:"customer_id"`
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
