package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	DefaultBaseURL       = "https://app.harness.io/gateway"
	ResourceTypePipeline = "pipeline"
)

type Client struct {
	APIToken  string
	AccountID string
	OrgID     string
	ProjectID string
	BaseURL   string
	http      core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type RunPipelineRequest struct {
	PipelineIdentifier string
	Ref                string
	InputSetRefs       []string
	InputYAML          string
}

type RunPipelineResponse struct {
	ExecutionID string
}

type ExecutionSummary struct {
	ExecutionID        string
	PipelineIdentifier string
	Status             string
	PlanExecutionURL   string
	StartedAt          string
	EndedAt            string
}

type Pipeline struct {
	Identifier string
	Name       string
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiToken, err := requiredConfig(ctx, "apiToken")
	if err != nil {
		return nil, err
	}

	accountID, err := requiredConfig(ctx, "accountId")
	if err != nil {
		return nil, err
	}

	orgID, err := optionalConfig(ctx, "orgId")
	if err != nil {
		return nil, err
	}

	projectID, err := optionalConfig(ctx, "projectId")
	if err != nil {
		return nil, err
	}

	baseURL, err := optionalConfig(ctx, "baseURL")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		APIToken:  apiToken,
		AccountID: accountID,
		OrgID:     orgID,
		ProjectID: projectID,
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		http:      httpClient,
	}, nil
}

func requiredConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return trimmed, nil
}

func optionalConfig(ctx core.IntegrationContext, name string) (string, error) {
	value, err := ctx.GetConfig(name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func (c *Client) Verify() error {
	_, _, err := c.execRequest(http.MethodGet, "/ng/api/user/currentUser", nil, nil, false)
	return err
}

func (c *Client) RunPipeline(request RunPipelineRequest) (*RunPipelineResponse, error) {
	pipelineIdentifier := strings.TrimSpace(request.PipelineIdentifier)
	if pipelineIdentifier == "" {
		return nil, fmt.Errorf("pipeline identifier is required")
	}

	query := c.scopeQuery()
	payload := map[string]any{}

	if len(request.InputSetRefs) > 0 {
		payload["inputSetReferences"] = request.InputSetRefs
	}

	if strings.TrimSpace(request.InputYAML) != "" {
		// Harness expects this field name when executing with input sets.
		payload["lastYamlToMerge"] = request.InputYAML
	}

	if branch, tag := parseRef(request.Ref); branch != "" || tag != "" {
		runPipelineInputs := map[string]any{}
		if branch != "" {
			runPipelineInputs["branch"] = branch
		}
		if tag != "" {
			runPipelineInputs["tag"] = tag
		}
		payload["runPipelineInputs"] = runPipelineInputs
	}

	_, body, err := c.execRequest(
		http.MethodPost,
		fmt.Sprintf("/pipeline/api/pipeline/execute/%s/inputSetList", url.PathEscape(pipelineIdentifier)),
		query,
		payload,
		true,
	)
	if err != nil {
		return nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse run pipeline response: %w", err)
	}

	if status := strings.ToLower(readString(data["status"])); status == "error" || status == "failure" {
		message := strings.TrimSpace(readString(data["message"]))
		if message == "" {
			message = string(body)
		}
		return nil, fmt.Errorf("run pipeline failed: %s", message)
	}

	executionID := firstNonEmpty(
		readString(data["planExecutionId"]),
		readString(data["executionId"]),
		readStringPath(data, "data", "planExecutionId"),
		readStringPath(data, "data", "executionId"),
		readStringPath(data, "data", "id"),
		readStringPath(data, "data", "planExecution", "uuid"),
		readStringPath(data, "data", "planExecution", "executionUuid"),
		readStringPath(data, "resource", "planExecutionId"),
		readStringPath(data, "resource", "executionId"),
		readStringPath(data, "resource", "id"),
		readStringPath(data, "execution", "planExecutionId"),
		readStringPath(data, "execution", "executionId"),
		readStringPath(data, "execution", "id"),
	)
	if executionID == "" {
		return nil, fmt.Errorf("run pipeline response missing execution id: %s", string(body))
	}

	return &RunPipelineResponse{ExecutionID: executionID}, nil
}

func (c *Client) GetExecutionSummary(executionID string) (*ExecutionSummary, error) {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return nil, fmt.Errorf("execution id is required")
	}

	query := c.scopeQuery()
	payload := map[string]any{
		"filterType":       "PipelineExecution",
		"planExecutionIds": []string{executionID},
	}

	_, body, err := c.execRequest(
		http.MethodPost,
		"/pipeline/api/pipelines/execution/summary",
		query,
		payload,
		true,
	)
	if err != nil {
		return nil, err
	}

	response := map[string]any{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse execution summary response: %w", err)
	}

	item := firstExecutionSummaryItem(response)
	if item == nil {
		return nil, fmt.Errorf("execution summary not found")
	}

	summary := &ExecutionSummary{
		ExecutionID: firstNonEmpty(
			readString(item["planExecutionId"]),
			readString(item["executionId"]),
			readString(item["id"]),
			readStringPath(item, "planExecution", "uuid"),
			readStringPath(item, "planExecution", "executionUuid"),
		),
		PipelineIdentifier: firstNonEmpty(
			readString(item["pipelineIdentifier"]),
			readString(item["pipelineId"]),
			readStringPath(item, "planExecution", "metadata", "pipelineIdentifier"),
		),
		Status: firstNonEmpty(
			readString(item["status"]),
			readString(item["executionStatus"]),
			readStringPath(item, "planExecution", "status"),
			readStringPath(item, "planExecution", "metadata", "status"),
		),
		PlanExecutionURL: firstNonEmpty(
			readString(item["planExecutionUrl"]),
			readString(item["url"]),
			readStringPath(item, "planExecution", "planExecutionUrl"),
		),
		StartedAt: firstNonEmpty(
			readString(item["startTs"]),
			readString(item["startedAt"]),
			readStringPath(item, "planExecution", "startTs"),
		),
		EndedAt: firstNonEmpty(
			readString(item["endTs"]),
			readString(item["endedAt"]),
			readStringPath(item, "planExecution", "endTs"),
		),
	}

	if summary.ExecutionID == "" {
		summary.ExecutionID = executionID
	}

	return summary, nil
}

func (c *Client) ListPipelines() ([]Pipeline, error) {
	query := c.scopeQuery()
	query.Set("page", "0")
	query.Set("size", "100")

	_, body, err := c.execRequest(
		http.MethodPost,
		"/pipeline/api/pipelines/list",
		query,
		map[string]any{
			"filterType": "PipelineSetup",
		},
		true,
	)
	if err != nil {
		return nil, err
	}

	response := map[string]any{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse list pipelines response: %w", err)
	}

	items := firstContentArray(response)
	pipelines := make([]Pipeline, 0, len(items))
	for _, item := range items {
		identifier := firstNonEmpty(
			readString(item["identifier"]),
			readString(item["id"]),
		)

		name := firstNonEmpty(
			readString(item["name"]),
			identifier,
		)

		if identifier == "" {
			continue
		}

		pipelines = append(pipelines, Pipeline{Identifier: identifier, Name: name})
	}

	return pipelines, nil
}

func (c *Client) execRequest(method, endpoint string, query url.Values, payload any, includeJSONContentType bool) (*http.Response, []byte, error) {
	requestURL, err := c.buildURL(endpoint, query)
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

func (c *Client) buildURL(endpoint string, query url.Values) (string, error) {
	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	baseURL.Path = path.Join(baseURL.Path, endpoint)
	if query != nil {
		baseURL.RawQuery = query.Encode()
	}

	return baseURL.String(), nil
}

func (c *Client) scopeQuery() url.Values {
	query := url.Values{}
	query.Set("accountIdentifier", c.AccountID)

	if c.OrgID != "" {
		query.Set("orgIdentifier", c.OrgID)
	}

	if c.ProjectID != "" {
		query.Set("projectIdentifier", c.ProjectID)
	}

	return query
}

func parseRef(ref string) (string, string) {
	trimmed := strings.TrimSpace(ref)
	switch {
	case strings.HasPrefix(trimmed, "refs/heads/"):
		return strings.TrimPrefix(trimmed, "refs/heads/"), ""
	case strings.HasPrefix(trimmed, "refs/tags/"):
		return "", strings.TrimPrefix(trimmed, "refs/tags/")
	default:
		return "", ""
	}
}

func firstExecutionSummaryItem(response map[string]any) map[string]any {
	if item := firstMapFromArray(readAnyPath(response, "data", "content")); item != nil {
		return item
	}
	if item := firstMapFromArray(readAnyPath(response, "data", "items")); item != nil {
		return item
	}
	if item := firstMapFromArray(readAnyPath(response, "content")); item != nil {
		return item
	}
	return firstMapFromArray(readAnyPath(response, "data"))
}

func firstContentArray(response map[string]any) []map[string]any {
	if items := arrayOfMaps(readAnyPath(response, "data", "content")); len(items) > 0 {
		return items
	}
	if items := arrayOfMaps(readAnyPath(response, "data", "items")); len(items) > 0 {
		return items
	}
	if items := arrayOfMaps(readAnyPath(response, "content")); len(items) > 0 {
		return items
	}
	if items := arrayOfMaps(readAnyPath(response, "data")); len(items) > 0 {
		return items
	}
	return []map[string]any{}
}

func readAnyPath(input map[string]any, path ...string) any {
	current := any(input)
	for _, key := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		next, ok := currentMap[key]
		if !ok {
			return nil
		}

		current = next
	}

	return current
}

func readStringPath(input map[string]any, path ...string) string {
	value := readAnyPath(input, path...)
	return readString(value)
}

func readString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case float32:
		f := float64(typed)
		if math.Mod(f, 1) == 0 {
			return strconv.FormatInt(int64(f), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(f, 'f', -1, 32))
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	}

	if text, ok := value.(fmt.Stringer); ok {
		return strings.TrimSpace(text.String())
	}

	return ""
}

func arrayOfMaps(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		mapItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, mapItem)
	}

	return result
}

func firstMapFromArray(value any) map[string]any {
	items := arrayOfMaps(value)
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
