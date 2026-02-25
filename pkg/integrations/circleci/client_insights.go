package circleci

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type WorkflowSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkflowSummariesResponse struct {
	Items         []WorkflowSummary `json:"items"`
	NextPageToken string            `json:"next_page_token"`
}

func (c *Client) ListWorkflowSummaries(projectSlug string) (*WorkflowSummariesResponse, error) {
	response := WorkflowSummariesResponse{
		Items: []WorkflowSummary{},
	}
	pageToken := ""
	seenPageTokens := map[string]struct{}{}

	for {
		reqURL := fmt.Sprintf("%s/insights/%s/workflows", baseURL, projectSlug)
		query := url.Values{}
		query.Set("all-branches", "true")
		if pageToken != "" {
			query.Set("page-token", pageToken)
		}
		reqURL = fmt.Sprintf("%s?%s", reqURL, query.Encode())

		responseBody, err := c.execRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		var page WorkflowSummariesResponse
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("error unmarshaling response: %v", err)
		}

		response.Items = append(response.Items, page.Items...)
		if page.NextPageToken == "" {
			break
		}

		if _, ok := seenPageTokens[page.NextPageToken]; ok {
			break
		}
		seenPageTokens[page.NextPageToken] = struct{}{}
		pageToken = page.NextPageToken
	}

	return &response, nil
}

type WorkflowRunsParams struct {
	Branch    string
	StartDate string
	EndDate   string
}

type WorkflowRunsResponse struct {
	Items         []WorkflowRunItem `json:"items"`
	NextPageToken string            `json:"next_page_token"`
}

type WorkflowRunItem struct {
	ID          string `json:"id"`
	Branch      string `json:"branch"`
	Duration    int    `json:"duration"`
	CreatedAt   string `json:"created_at"`
	StoppedAt   string `json:"stopped_at"`
	CreditsUsed int    `json:"credits_used"`
	Status      string `json:"status"`
	IsApproval  bool   `json:"is_approval"`
}

func (c *Client) GetWorkflowRuns(projectSlug, workflowName string, params WorkflowRunsParams) (*WorkflowRunsResponse, error) {
	reqURL := fmt.Sprintf("%s/insights/%s/workflows/%s", baseURL, projectSlug, url.PathEscape(workflowName))

	query := url.Values{}
	if params.Branch != "" {
		query.Set("branch", params.Branch)
	}
	if params.StartDate != "" {
		query.Set("start-date", params.StartDate)
	}
	if params.EndDate != "" {
		query.Set("end-date", params.EndDate)
	}

	if len(query) > 0 {
		reqURL = reqURL + "?" + query.Encode()
	}

	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response WorkflowRunsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type TestMetricsResponse struct {
	MostFailedTests []TestMetric `json:"most_failed_tests"`
	SlowestTests    []TestMetric `json:"slowest_tests"`
	TotalTestRuns   int          `json:"total_test_runs"`
	TestRuns        []TestRun    `json:"test_runs"`
}

type TestMetric struct {
	TestName   string  `json:"test_name"`
	Classname  string  `json:"classname"`
	FailedRuns int     `json:"failed_runs"`
	TotalRuns  int     `json:"total_runs"`
	Flaky      bool    `json:"flaky"`
	P50Secs    float64 `json:"p50_duration_secs,omitempty"`
	Source     string  `json:"source,omitempty"`
	File       string  `json:"file,omitempty"`
}

type TestRun struct {
	PipelineNumber int     `json:"pipeline_number"`
	WorkflowID     string  `json:"workflow_id"`
	SuccessRate    float64 `json:"success_rate"`
	TestCounts     struct {
		Error   int `json:"error"`
		Failure int `json:"failure"`
		Skipped int `json:"skipped"`
		Success int `json:"success"`
		Total   int `json:"total"`
	} `json:"test_counts"`
}

func (c *Client) GetTestMetrics(projectSlug, workflowName string) (*TestMetricsResponse, error) {
	reqURL := fmt.Sprintf("%s/insights/%s/workflows/%s/test-metrics", baseURL, projectSlug, url.PathEscape(workflowName))
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response TestMetricsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type FlakyTestsResponse struct {
	FlakyTests      []FlakyTest `json:"flaky-tests"`
	TotalFlakyTests int         `json:"total-flaky-tests"`
}

type FlakyTest struct {
	TestName     string `json:"test-name"`
	Classname    string `json:"classname"`
	PipelineName string `json:"pipeline-name"`
	WorkflowName string `json:"workflow-name"`
	JobName      string `json:"job-name"`
	TimesFlaky   int    `json:"times-flaked"`
	Source       string `json:"source"`
	File         string `json:"file"`
}

func (c *Client) GetFlakyTests(projectSlug string) (*FlakyTestsResponse, error) {
	reqURL := fmt.Sprintf("%s/insights/%s/flaky-tests", baseURL, projectSlug)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response FlakyTestsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}
