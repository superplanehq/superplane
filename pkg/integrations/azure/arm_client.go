package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const (
	armBaseURL                 = "https://management.azure.com"
	armAPIVersionCompute       = "2024-03-01"
	armAPIVersionNetwork       = "2023-11-01"
	armAPIVersionResources     = "2024-03-01"
	armAPIVersionEventGrid     = "2024-06-01-preview"

	lroDefaultPollInterval = 5 * time.Second
	lroMaxPollDuration     = 30 * time.Minute
)

// armClient wraps HTTP calls to the Azure Resource Manager REST API.
type armClient struct {
	credential     azcore.TokenCredential
	subscriptionID string
	httpClient     *http.Client
}

func newARMClient(credential azcore.TokenCredential, subscriptionID string) *armClient {
	return &armClient{
		credential:     credential,
		subscriptionID: subscriptionID,
		httpClient:     &http.Client{Timeout: 60 * time.Second},
	}
}

// bearerToken obtains an OAuth2 token for the ARM management plane.
func (c *armClient) bearerToken(ctx context.Context) (string, error) {
	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get ARM token: %w", err)
	}
	return token.Token, nil
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

	return c.httpClient.Do(req)
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

	for currentURL != "" {
		var page struct {
			Value    []json.RawMessage `json:"value"`
			NextLink string            `json:"nextLink"`
		}

		if err := c.get(ctx, currentURL, &page); err != nil {
			return nil, err
		}

		all = append(all, page.Value...)
		currentURL = page.NextLink
	}

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
	if armErr, ok := err.(*armError); ok {
		return armErr.StatusCode == 404
	}
	return false
}
