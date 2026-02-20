package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) execRequest(method, endpoint string, query url.Values, payload any, includeJSONContentType bool) (*http.Response, []byte, error) {
	requestURL, err := buildURLFromBase(c.BaseURL, endpoint, query)
	if err != nil {
		return nil, nil, err
	}

	var body io.Reader
	if payload != nil {
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to encode request payload: %w", err)
		}
		body = bytes.NewReader(encodedPayload)
	}

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.APIToken)
	if includeJSONContentType {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return res, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}

func (c *Client) execRawRequest(
	method, endpoint string,
	query url.Values,
	payload []byte,
	contentType string,
	headers map[string]string,
) (*http.Response, []byte, error) {
	requestURL, err := buildURLFromBase(c.BaseURL, endpoint, query)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(method, requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.APIToken)
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for headerKey, headerValue := range headers {
		if strings.TrimSpace(headerKey) == "" || strings.TrimSpace(headerValue) == "" {
			continue
		}
		req.Header.Set(headerKey, headerValue)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return res, nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return res, responseBody, nil
}

func buildURLFromBase(baseURLValue, endpoint string, query url.Values) (string, error) {
	baseURL, err := url.Parse(baseURLValue)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	basePath := strings.TrimRight(baseURL.Path, "/")
	endpointPath := strings.TrimLeft(endpoint, "/")
	switch {
	case basePath == "" && endpointPath == "":
		baseURL.Path = "/"
	case basePath == "":
		baseURL.Path = "/" + endpointPath
	case endpointPath == "":
		baseURL.Path = basePath
	default:
		baseURL.Path = basePath + "/" + endpointPath
	}

	if query != nil {
		baseURL.RawQuery = query.Encode()
	}

	return baseURL.String(), nil
}
