package jenkins

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL  string
	Username string
	APIToken string
	http     core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	rawURL, err := ctx.GetConfig("url")
	if err != nil {
		return nil, fmt.Errorf("error getting url: %v", err)
	}

	username, err := ctx.GetConfig("username")
	if err != nil {
		return nil, fmt.Errorf("error getting username: %v", err)
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	return &Client{
		BaseURL:  strings.TrimRight(string(rawURL), "/"),
		Username: string(username),
		APIToken: string(apiToken),
		http:     httpCtx,
	}, nil
}

func (c *Client) authHeader() string {
	credentials := fmt.Sprintf("%s:%s", c.Username, c.APIToken)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials)))
}

// execRequest performs an HTTP request and returns the response and status code.
// The caller can specify additional allowed status codes that should not be
// treated as errors (e.g. 201 for build triggers).
func (c *Client) execRequest(method, requestURL string, body io.Reader, allowedStatuses ...int) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res, nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, responseBody, nil
	}

	for _, status := range allowedStatuses {
		if res.StatusCode == status {
			return res, responseBody, nil
		}
	}

	return res, nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
}

func (c *Client) apiURL(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

// encodeJobPath converts a job path like "folder/jobname" into the URL segment
// "job/folder/job/jobname" that Jenkins expects.
func encodeJobPath(jobPath string) string {
	parts := strings.Split(jobPath, "/")
	encoded := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		encoded = append(encoded, "job", url.PathEscape(part))
	}
	return strings.Join(encoded, "/")
}

// ServerInfo represents the top-level Jenkins API response.
type ServerInfo struct {
	Mode string `json:"mode"`
	URL  string `json:"url"`
}

// GetServerInfo verifies credentials by fetching the Jenkins server info.
func (c *Client) GetServerInfo() (*ServerInfo, error) {
	_, responseBody, err := c.execRequest(http.MethodGet, c.apiURL("/api/json"), nil)
	if err != nil {
		return nil, err
	}

	var info ServerInfo
	if err := json.Unmarshal(responseBody, &info); err != nil {
		return nil, fmt.Errorf("error parsing server info response: %v", err)
	}

	return &info, nil
}

// Job represents a Jenkins job.
type Job struct {
	Name     string `json:"name"`
	FullName string `json:"fullName"`
	URL      string `json:"url"`
	Color    string `json:"color"`
	Class    string `json:"_class"`
	Jobs     []Job  `json:"jobs,omitempty"`
}

type jobListResponse struct {
	Jobs []Job `json:"jobs"`
}

// ListJobs returns all jobs from the Jenkins instance, flattening folder structures.
func (c *Client) ListJobs() ([]Job, error) {
	requestURL := c.apiURL("/api/json?tree=jobs[name,fullName,url,color,_class,jobs[name,fullName,url,color,_class]]")
	_, responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var response jobListResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing jobs response: %v", err)
	}

	return flattenJobs(response.Jobs), nil
}

// flattenJobs recursively collects leaf jobs from nested folder structures.
func flattenJobs(jobs []Job) []Job {
	var result []Job
	for _, job := range jobs {
		if len(job.Jobs) > 0 {
			result = append(result, flattenJobs(job.Jobs)...)
		} else {
			result = append(result, job)
		}
	}
	return result
}

// GetJob fetches details for a specific job by path.
func (c *Client) GetJob(jobPath string) (*Job, error) {
	requestURL := c.apiURL(fmt.Sprintf("/%s/api/json", encodeJobPath(jobPath)))
	_, responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var job Job
	if err := json.Unmarshal(responseBody, &job); err != nil {
		return nil, fmt.Errorf("error parsing job response: %v", err)
	}

	return &job, nil
}

// TriggerBuild triggers a build for the given job. If parameters are provided,
// it uses the buildWithParameters endpoint. Returns the queue item ID.
func (c *Client) TriggerBuild(jobPath string, parameters map[string]string) (int64, error) {
	var requestURL string
	var body io.Reader

	if len(parameters) > 0 {
		params := url.Values{}
		for k, v := range parameters {
			params.Set(k, v)
		}
		requestURL = c.apiURL(fmt.Sprintf("/%s/buildWithParameters?%s", encodeJobPath(jobPath), params.Encode()))
	} else {
		requestURL = c.apiURL(fmt.Sprintf("/%s/build", encodeJobPath(jobPath)))
	}

	res, _, err := c.execRequest(http.MethodPost, requestURL, body, http.StatusCreated)
	if err != nil {
		return 0, fmt.Errorf("error triggering build: %v", err)
	}

	location := res.Header.Get("Location")
	if location == "" {
		return 0, fmt.Errorf("no Location header in trigger response")
	}

	return parseQueueID(location)
}

// parseQueueID extracts the queue item ID from a Jenkins Location header.
// The format is: {baseURL}/queue/item/{id}/
func parseQueueID(location string) (int64, error) {
	location = strings.TrimRight(location, "/")
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid queue location: %s", location)
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing queue item ID from %s: %v", location, err)
	}

	return id, nil
}

// QueueItem represents an item in the Jenkins build queue.
type QueueItem struct {
	ID         int64     `json:"id"`
	Blocked    bool      `json:"blocked"`
	Executable *BuildRef `json:"executable"`
}

// BuildRef is a reference to a build from a queue item.
type BuildRef struct {
	Number int64  `json:"number"`
	URL    string `json:"url"`
}

// GetQueueItem fetches a queue item by ID. When the build has started,
// the Executable field will be populated with the build number.
func (c *Client) GetQueueItem(id int64) (*QueueItem, error) {
	requestURL := c.apiURL(fmt.Sprintf("/queue/item/%d/api/json", id))
	_, responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var item QueueItem
	if err := json.Unmarshal(responseBody, &item); err != nil {
		return nil, fmt.Errorf("error parsing queue item response: %v", err)
	}

	return &item, nil
}

// Build represents a Jenkins build.
type Build struct {
	Number    int64  `json:"number"`
	URL       string `json:"url"`
	Result    string `json:"result"`
	Building  bool   `json:"building"`
	Duration  int64  `json:"duration"`
	Timestamp int64  `json:"timestamp"`
}

// GetBuild fetches build details for a specific job and build number.
func (c *Client) GetBuild(jobPath string, buildNumber int64) (*Build, error) {
	requestURL := c.apiURL(fmt.Sprintf("/%s/%d/api/json", encodeJobPath(jobPath), buildNumber))
	_, responseBody, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	var build Build
	if err := json.Unmarshal(responseBody, &build); err != nil {
		return nil, fmt.Errorf("error parsing build response: %v", err)
	}

	return &build, nil
}
