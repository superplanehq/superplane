package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://api.openai.com/v1"

type Client struct {
	APIKey   string
	AdminKey string
	BaseURL  string
	http     core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	adminKey, _ := ctx.GetConfig("adminKey")

	baseURL := defaultBaseURL
	if customURL, err := ctx.GetConfig("baseURL"); err == nil && len(customURL) > 0 {
		baseURL = string(customURL)
	}

	return &Client{
		APIKey:   string(apiKey),
		AdminKey: string(adminKey),
		BaseURL:  baseURL,
		http:     httpClient,
	}, nil
}

// CreateResponseRequest is the Responses API request. Input is a plain string
// for text-only prompts, or []InputMessage when attaching files/images.
type CreateResponseRequest struct {
	Model string              `json:"model"`
	Input any                 `json:"input"`
	Text  *ResponseTextConfig `json:"text,omitempty"`
}

// InputMessage is a role + content-parts entry in the Responses API input array.
type InputMessage struct {
	Role    string      `json:"role"`
	Content []InputPart `json:"content"`
}

// InputPart is a content part: input_text, input_image, or input_file.
type InputPart struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

// ResponseTextConfig carries the structured-output format for the Responses API.
type ResponseTextConfig struct {
	Format *ResponseFormat `json:"format,omitempty"`
}

// ResponseFormat constrains the response to a JSON schema (Responses API: text.format).
type ResponseFormat struct {
	Type   string `json:"type"`             // "json_schema"
	Name   string `json:"name"`             // required by the Responses API
	Schema any    `json:"schema"`           // the JSON Schema object
	Strict bool   `json:"strict,omitempty"` // enforce exact schema conformance
}

type ResponseContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal,omitempty"`
}

type ResponseOutput struct {
	Type    string            `json:"type"`
	Role    string            `json:"role"`
	Content []ResponseContent `json:"content"`
}

type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type OpenAIResponse struct {
	ID         string           `json:"id"`
	Model      string           `json:"model"`
	OutputText string           `json:"output_text"`
	Output     []ResponseOutput `json:"output"`
	Usage      *ResponseUsage   `json:"usage,omitempty"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID string `json:"id"`
}

// UsagePage is one page of the org Usage/Costs API: buckets plus a pagination cursor.
type UsagePage struct {
	Object   string        `json:"object"`
	Data     []UsageBucket `json:"data"`
	HasMore  bool          `json:"has_more"`
	NextPage string        `json:"next_page"`
}

// UsageBucket is a time bucket of usage results. Results are kept generic
// because each usage category returns different metric fields.
type UsageBucket struct {
	Object    string           `json:"object"`
	StartTime int64            `json:"start_time"`
	EndTime   int64            `json:"end_time"`
	Results   []map[string]any `json:"results"`
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	return err
}

// VerifyAdmin checks the admin API key by requesting a single usage bucket.
func (c *Client) VerifyAdmin() error {
	params := url.Values{}
	params.Set("start_time", fmt.Sprintf("%d", time.Now().AddDate(0, 0, -1).Unix()))
	params.Set("limit", "1")
	_, err := c.execRequestWithKey(http.MethodGet, c.BaseURL+"/organization/usage/completions?"+params.Encode(), nil, c.AdminKey)
	return err
}

// maxUsagePages bounds pagination so a misbehaving cursor cannot loop forever.
const maxUsagePages = 12

// GetUsage fetches all buckets for an org Usage/Costs API path (e.g.
// "/organization/usage/completions"), following the next_page cursor.
// Requires the admin API key.
func (c *Client) GetUsage(path string, params url.Values) ([]UsageBucket, error) {
	if c.AdminKey == "" {
		return nil, fmt.Errorf("admin API key is not configured")
	}

	buckets := []UsageBucket{}
	for range maxUsagePages {
		responseBody, err := c.execRequestWithKey(http.MethodGet, c.BaseURL+path+"?"+params.Encode(), nil, c.AdminKey)
		if err != nil {
			return nil, err
		}

		var page UsagePage
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("failed to unmarshal usage response: %v", err)
		}

		buckets = append(buckets, page.Data...)
		if !page.HasMore || page.NextPage == "" {
			return buckets, nil
		}

		params.Set("page", page.NextPage)
	}

	// Truncating silently would report incomplete totals as a success.
	return nil, fmt.Errorf("usage response exceeded %d pages; narrow the date range", maxUsagePages)
}

func (c *Client) ListModels() ([]Model, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) CreateResponse(req CreateResponseRequest) (*OpenAIResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response OpenAIResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &response, nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// createFormFile is like multipart.Writer.CreateFormFile but lets the caller set
// the part's Content-Type. The stdlib helper hardcodes application/octet-stream,
// which causes the provider to store the file with the wrong media type instead
// of the detected one.
func createFormFile(w *multipart.Writer, fieldname, filename, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		quoteEscaper.Replace(fieldname), quoteEscaper.Replace(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

// UploadFile uploads a file to the OpenAI Files API with the given purpose
// ("vision" for images, "user_data" for documents) and returns its file id.
func (c *Client) UploadFile(content io.Reader, filename, purpose, contentType string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("purpose", purpose); err != nil {
		return "", fmt.Errorf("write purpose field: %w", err)
	}
	part, err := createFormFile(writer, "file", filepath.Base(filename), contentType)
	if err != nil {
		return "", fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, content); err != nil {
		return "", fmt.Errorf("copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/files", &body)
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload file failed (%d): %s", res.StatusCode, string(resBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resBody, &result); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("upload returned empty file ID")
	}
	return result.ID, nil
}

// DeleteFile removes an uploaded file. Best-effort cleanup; no-op for empty IDs.
func (c *Client) DeleteFile(fileID string) error {
	if fileID == "" {
		return nil
	}
	_, err := c.execRequest(http.MethodDelete, c.BaseURL+"/files/"+url.PathEscape(fileID), nil)
	return err
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestWithKey(method, URL, body, c.APIKey)
}

func (c *Client) execRequestWithKey(method, URL string, body io.Reader, key string) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
