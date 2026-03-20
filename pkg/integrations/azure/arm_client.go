package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/sirupsen/logrus"
)

const (
	armBaseURL                    = "https://management.azure.com"
	armAPIVersionCompute          = "2024-03-01"
	armAPIVersionNetwork          = "2023-11-01"
	armAPIVersionResources        = "2024-03-01"
	armAPIVersionEventGrid        = "2024-06-01-preview"
	armAPIVersionResourceProvider = "2021-04-01"

	lroDefaultPollInterval = 5 * time.Second
	lroMaxPollDuration     = 30 * time.Minute
)

// armClient wraps HTTP calls to the Azure Resource Manager REST API.
type armClient struct {
	credential     azcore.TokenCredential
	subscriptionID string
	httpClient     *http.Client
	logger         *logrus.Entry

	// baseURL overrides armBaseURL when set (used in tests).
	baseURL string

	// tokenFunc overrides credential-based token fetching when set (used in tests).
	tokenFunc func(context.Context) (string, error)
}

func newARMClient(credential azcore.TokenCredential, subscriptionID string, logger *logrus.Entry) *armClient {
	return &armClient{
		credential:     credential,
		subscriptionID: subscriptionID,
		httpClient:     &http.Client{Timeout: 120 * time.Second},
		logger:         logger,
	}
}

// bearerToken obtains an OAuth2 token for the ARM management plane.
func (c *armClient) bearerToken(ctx context.Context) (string, error) {
	if c.tokenFunc != nil {
		return c.tokenFunc(ctx)
	}
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get ARM token: %w", err)
	}
	return token.Token, nil
}

func (c *armClient) getBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return armBaseURL
}

// doRequest executes an authenticated ARM request and returns the response.
func (c *armClient) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		c.logger.WithError(err).Errorf("ARM %s %s failed after %s", method, url, elapsed)
		return nil, err
	}

	c.logger.Debugf("ARM %s %s -> %d (%s)", method, url, resp.StatusCode, elapsed)
	return resp, nil
}

// get performs a GET request and unmarshals the response JSON into dest.
func (c *armClient) get(ctx context.Context, url string, dest any) error {
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readARMError(resp)
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// post performs a POST request and returns the raw response.
func (c *armClient) post(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, readARMError(resp)
	}

	return resp, nil
}

// ensureResourceProviderRegistered checks whether an Azure resource provider
// (e.g. "Microsoft.EventGrid") is registered, and if not, attempts to register
// it. The method fails fast if the caller lacks permissions or if registration
// does not complete within a short window.
func (c *armClient) ensureResourceProviderRegistered(ctx context.Context, providerNamespace string) error {
	checkURL := fmt.Sprintf("%s/subscriptions/%s/providers/%s?api-version=%s",
		armBaseURL, c.subscriptionID, providerNamespace, armAPIVersionResourceProvider)

	var provider struct {
		RegistrationState string `json:"registrationState"`
	}
	if err := c.get(ctx, checkURL, &provider); err != nil {
		return fmt.Errorf("failed to check %s registration state: %w", providerNamespace, err)
	}

	if provider.RegistrationState == "Registered" {
		return nil
	}

	// Attempt to register the provider.
	registerURL := fmt.Sprintf("%s/subscriptions/%s/providers/%s/register?api-version=%s",
		armBaseURL, c.subscriptionID, providerNamespace, armAPIVersionResourceProvider)

	resp, err := c.post(ctx, registerURL, nil)
	if err != nil {
		return fmt.Errorf(
			"%s resource provider is not registered and auto-registration failed: %w. "+
				"Please register it manually: az provider register --namespace %s",
			providerNamespace, err, providerNamespace,
		)
	}
	resp.Body.Close()

	// Poll with a short timeout — registration typically takes 10-30s.
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"timed out waiting for %s to register. "+
					"Please register it manually and retry: az provider register --namespace %s",
				providerNamespace, providerNamespace,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}

		if err := c.get(ctx, checkURL, &provider); err != nil {
			return fmt.Errorf("failed to check registration state: %w", err)
		}

		if provider.RegistrationState == "Registered" {
			return nil
		}
	}
}

// put performs a PUT request with a JSON body and returns the raw response.
func (c *armClient) put(ctx context.Context, url string, body any) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, readARMError(resp)
	}

	return resp, nil
}

// patch performs a PATCH request with a JSON body and returns the raw response.
func (c *armClient) patch(ctx context.Context, url string, body any) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, readARMError(resp)
	}

	return resp, nil
}

// deleteAndPoll performs a DELETE, then polls the LRO until terminal state.
func (c *armClient) deleteAndPoll(ctx context.Context, url string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readARMError(resp)
	}

	// 200 or 204 with no async header means synchronous completion
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		asyncURL := resp.Header.Get("Azure-AsyncOperation")
		locationURL := resp.Header.Get("Location")
		if asyncURL == "" && locationURL == "" {
			return nil
		}

		pollURL := asyncURL
		if pollURL == "" {
			pollURL = locationURL
		}
		_, err = c.pollLRO(ctx, pollURL, "")
		return err
	}

	// 202 Accepted — poll the LRO
	asyncURL := resp.Header.Get("Azure-AsyncOperation")
	locationURL := resp.Header.Get("Location")

	pollURL := asyncURL
	if pollURL == "" {
		pollURL = locationURL
	}

	if pollURL == "" {
		// No poll URL but 202 — treat as success
		return nil
	}

	_, err = c.pollLRO(ctx, pollURL, "")
	return err
}

// postAndPoll performs a POST (with an optional JSON body), then polls the
// LRO until terminal state. Used for VM actions (start, stop, deallocate,
// restart) that return 202 Accepted.
func (c *armClient) postAndPoll(ctx context.Context, url string, body any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return readARMError(resp)
	}

	// 200 or 204 with no async header means synchronous completion
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		asyncURL := resp.Header.Get("Azure-AsyncOperation")
		locationURL := resp.Header.Get("Location")
		if asyncURL == "" && locationURL == "" {
			return nil
		}

		pollURL := asyncURL
		if pollURL == "" {
			pollURL = locationURL
		}
		_, err = c.pollLRO(ctx, pollURL, "")
		return err
	}

	// 202 Accepted — poll the LRO
	asyncURL := resp.Header.Get("Azure-AsyncOperation")
	locationURL := resp.Header.Get("Location")

	pollURL := asyncURL
	if pollURL == "" {
		pollURL = locationURL
	}

	if pollURL == "" {
		return nil
	}

	_, err = c.pollLRO(ctx, pollURL, "")
	return err
}

// putAndPoll performs a PUT, then polls the LRO until terminal state.
// Returns the final response body.
func (c *armClient) putAndPoll(ctx context.Context, url string, body any) (json.RawMessage, error) {
	resp, err := c.put(ctx, url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If 200/201 with no async header, the operation completed synchronously
	asyncURL := resp.Header.Get("Azure-AsyncOperation")
	locationURL := resp.Header.Get("Location")

	// Read initial response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if asyncURL == "" && locationURL == "" {
		// Synchronous completion
		return respBody, nil
	}

	// Poll the LRO
	pollURL := asyncURL
	if pollURL == "" {
		pollURL = locationURL
	}

	return c.pollLRO(ctx, pollURL, url)
}

// pollLRO polls a long-running operation URL until it reaches a terminal state.
// resourceURL is the original resource URL, used to fetch the final result.
func (c *armClient) pollLRO(ctx context.Context, pollURL, resourceURL string) (json.RawMessage, error) {
	deadline := time.Now().Add(lroMaxPollDuration)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("LRO timed out after %v", lroMaxPollDuration)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lroDefaultPollInterval):
		}

		var status struct {
			Status     string          `json:"status"`
			Properties json.RawMessage `json:"properties"`
			Error      *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := c.get(ctx, pollURL, &status); err != nil {
			return nil, fmt.Errorf("failed to poll LRO: %w", err)
		}

		switch status.Status {
		case "Succeeded":
			// For delete operations (empty resourceURL), no final GET is needed.
			if resourceURL == "" {
				return nil, nil
			}

			// Fetch the final resource state
			var result json.RawMessage
			if err := c.get(ctx, resourceURL, &result); err != nil {
				return nil, fmt.Errorf("LRO succeeded but failed to fetch resource: %w", err)
			}
			return result, nil

		case "Failed":
			msg := "unknown error"
			if status.Error != nil {
				msg = fmt.Sprintf("%s: %s", status.Error.Code, status.Error.Message)
			}
			return nil, fmt.Errorf("LRO failed: %s", msg)

		case "Canceled":
			return nil, fmt.Errorf("LRO was canceled")
		}
		// Otherwise (InProgress, Creating, etc.) keep polling
	}
}

// listAll paginates through all pages of a list endpoint.
// It follows `nextLink` values and collects all `value` arrays.
func (c *armClient) listAll(ctx context.Context, url string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	currentURL := url
	page := 0

	c.logger.Debugf("ARM listAll: %s", url)

	for currentURL != "" {
		page++

		var pageResp struct {
			Value    []json.RawMessage `json:"value"`
			NextLink string            `json:"nextLink"`
		}

		if err := c.get(ctx, currentURL, &pageResp); err != nil {
			c.logger.WithError(err).Errorf("ARM listAll failed on page %d", page)
			return nil, err
		}

		c.logger.Debugf("ARM listAll page %d: %d items, hasNextLink=%v", page, len(pageResp.Value), pageResp.NextLink != "")
		all = append(all, pageResp.Value...)
		currentURL = pageResp.NextLink
	}

	c.logger.Debugf("ARM listAll complete: %d total items across %d page(s)", len(all), page)
	return all, nil
}

// armError represents an Azure Resource Manager error response.
type armError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *armError) Error() string {
	return fmt.Sprintf("ARM error %d: %s - %s", e.StatusCode, e.Code, e.Message)
}

func readARMError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Code != "" {
		return &armError{
			StatusCode: resp.StatusCode,
			Code:       errResp.Error.Code,
			Message:    errResp.Error.Message,
		}
	}

	return &armError{
		StatusCode: resp.StatusCode,
		Code:       "Unknown",
		Message:    string(body),
	}
}

func isARMNotFound(err error) bool {
	var armErr *armError
	return errors.As(err, &armErr) && armErr.StatusCode == 404
}
