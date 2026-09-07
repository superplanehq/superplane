package productive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// projectsPageSize is how many projects each page requests.
	projectsPageSize = 100

	// maxProjectPages bounds how many pages ListProjects walks, so a
	// misbehaving pagination cursor cannot loop forever.
	maxProjectPages = 20
)

// Client talks to Productive.io's JSON:API v2 API using an API token and an
// organization id, both sent as headers on every request.
type Client struct {
	APIToken       string
	OrganizationID string
	BaseURL        string
	http           core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	if strings.TrimSpace(string(apiToken)) == "" {
		return nil, fmt.Errorf("missing Productive.io API token")
	}

	organizationID, err := ctx.GetConfig("organizationId")
	if err != nil {
		return nil, fmt.Errorf("error getting organizationId: %v", err)
	}

	if strings.TrimSpace(string(organizationID)) == "" {
		return nil, fmt.Errorf("missing Productive.io organization id")
	}

	baseURL := BaseURL
	if region, err := ctx.GetConfig("region"); err == nil && strings.TrimSpace(string(region)) != "" {
		baseURL = strings.TrimRight(strings.TrimSpace(string(region)), "/")
	}

	if httpCtx == nil {
		return nil, fmt.Errorf("missing HTTP context")
	}

	return &Client{
		APIToken:       strings.TrimSpace(string(apiToken)),
		OrganizationID: strings.TrimSpace(string(organizationID)),
		BaseURL:        baseURL,
		http:           httpCtx,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set(AuthTokenHeader, c.APIToken)
	req.Header.Set(OrganizationIDHeader, c.OrganizationID)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// ValidateCredentials confirms the token and organization id are accepted by
// Productive.io. Organization memberships is the cheapest authenticated
// endpoint that also proves the organization id header is correct: a token
// valid for a different organization is rejected before any data is read.
func (c *Client) ValidateCredentials() error {
	url := fmt.Sprintf("%s/organization_memberships?page[size]=1", c.BaseURL)
	_, err := c.execRequest(http.MethodGet, url, nil)
	return err
}

// resourceDocument is a single JSON:API resource, trimmed to the fields this
// client reads.
type resourceDocument struct {
	ID            string                          `json:"id"`
	Type          string                          `json:"type"`
	Attributes    map[string]any                  `json:"attributes"`
	Relationships map[string]resourceRelationship `json:"relationships"`
}

type resourceRelationship struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type resourceListResponse struct {
	Data []resourceDocument `json:"data"`
}

type resourceResponse struct {
	Data resourceDocument `json:"data"`
}

// Project is a Productive.io project, used by the onTask trigger's project
// picker and to scope a webhook to one project.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Task is a Productive.io task that can seed a factory intake.
type Task struct {
	ID          string
	Number      string
	Title       string
	Description string
	ProjectID   string
}

func projectFromDocument(doc resourceDocument) Project {
	name, _ := doc.Attributes["name"].(string)
	return Project{ID: doc.ID, Name: name}
}

func taskFromDocument(doc resourceDocument) Task {
	title, _ := doc.Attributes["title"].(string)
	description, _ := doc.Attributes["description"].(string)
	projectID := doc.Relationships["project"].Data.ID
	return Task{
		ID:          doc.ID,
		Number:      stringAttribute(doc.Attributes["task_number"]),
		Title:       title,
		Description: description,
		ProjectID:   projectID,
	}
}

func stringAttribute(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

// ListProjects returns every project of the organization. Productive.io
// paginates responses, so pages are walked until a short page ends them.
func (c *Client) ListProjects() ([]Project, error) {
	projects := []Project{}

	for page := 1; page <= maxProjectPages; page++ {
		url := fmt.Sprintf("%s/projects?page[number]=%d&page[size]=%d", c.BaseURL, page, projectsPageSize)
		body, err := c.execRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		response := resourceListResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing projects: %v", err)
		}

		for _, doc := range response.Data {
			projects = append(projects, projectFromDocument(doc))
		}

		if len(response.Data) < projectsPageSize {
			return projects, nil
		}
	}

	return projects, nil
}

// GetProject fetches a single project by id, for resolving the project a
// trigger was configured with into the name shown on its canvas card.
func (c *Client) GetProject(id string) (*Project, error) {
	url := fmt.Sprintf("%s/projects/%s", c.BaseURL, id)
	body, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response := resourceResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing project: %v", err)
	}

	if response.Data.ID == "" {
		return nil, fmt.Errorf("project %s not found", id)
	}

	project := projectFromDocument(response.Data)
	return &project, nil
}

// ListTasks returns open tasks from one project, optionally filtered by text.
func (c *Client) ListTasks(projectID, query string, limit int) ([]Task, error) {
	body, err := c.execRequest(http.MethodGet, c.taskListURL(projectID, query, limit), nil)
	if err != nil {
		return nil, err
	}

	response := resourceListResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing tasks: %v", err)
	}

	tasks := make([]Task, 0, len(response.Data))
	for _, doc := range response.Data {
		tasks = append(tasks, taskFromDocument(doc))
	}
	return tasks, nil
}

// ListNewestOpenTaskDocuments returns the untrimmed JSON:API resource of each
// open task in the project, newest first. Seeding an intake replays these
// through the graph the webhook feeds, and that graph reads attributes Task
// does not keep.
func (c *Client) ListNewestOpenTaskDocuments(projectID string, limit int) ([]map[string]any, error) {
	body, err := c.execRequest(http.MethodGet, c.taskListURL(projectID, "", limit), nil)
	if err != nil {
		return nil, err
	}

	response := struct {
		Data []map[string]any `json:"data"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing tasks: %v", err)
	}

	return response.Data, nil
}

// taskListURL asks for the open tasks of one project, newest first.
func (c *Client) taskListURL(projectID, query string, limit int) string {
	params := url.Values{}
	params.Set("filter[project_id]", projectID)
	params.Set("filter[status]", "1")
	params.Set("page[size]", strconv.Itoa(limit))
	params.Set("sort", "-created_at")
	if strings.TrimSpace(query) != "" {
		params.Set("filter[query]", strings.TrimSpace(query))
	}

	return fmt.Sprintf("%s/tasks?%s", c.BaseURL, params.Encode())
}

// GetTask returns one task by its Productive.io resource id.
func (c *Client) GetTask(id string) (*Task, error) {
	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/tasks/%s", c.BaseURL, url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	response := resourceResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing task: %v", err)
	}
	if response.Data.ID == "" {
		return nil, fmt.Errorf("task %s not found", id)
	}

	task := taskFromDocument(response.Data)
	return &task, nil
}

// Webhook is a Productive.io webhook subscription.
type Webhook struct {
	ID string `json:"id"`
}

// CreateWebhook registers a webhook subscribed to the given events, scoped to
// one project when projectID is non-empty. Productive.io answers with the
// created resource, whose id is stored so a later Cleanup can remove it.
func (c *Client) CreateWebhook(url, secret string, events []string, projectID string) (*Webhook, error) {
	attributes := map[string]any{
		"target_url": url,
		"secret":     secret,
		"events":     events,
	}

	data := map[string]any{
		"type":       "webhooks",
		"attributes": attributes,
	}

	if projectID != "" {
		data["relationships"] = map[string]any{
			"project": map[string]any{
				"data": map[string]any{"type": "projects", "id": projectID},
			},
		}
	}

	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	requestURL := fmt.Sprintf("%s/webhooks", c.BaseURL)
	responseBody, err := c.execRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := resourceResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing webhook response: %v", err)
	}

	if response.Data.ID == "" {
		return nil, fmt.Errorf("productive did not return a webhook id")
	}

	return &Webhook{ID: response.Data.ID}, nil
}

// DeleteWebhook removes a webhook subscription by id.
func (c *Client) DeleteWebhook(id string) error {
	url := fmt.Sprintf("%s/webhooks/%s", c.BaseURL, id)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}
