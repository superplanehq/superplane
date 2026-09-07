package productive

import (
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

	// sortNewestCreated and sortNewestUpdated order tasks by the timestamp a
	// caller reads them for: when they appeared, or when they last changed.
	sortNewestCreated = "-created_at"
	sortNewestUpdated = "-updated_at"
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

// taskListOptions describes one page of a project's tasks.
type taskListOptions struct {
	projectID string
	query     string
	openOnly  bool
	sort      string
	page      int
	pageSize  int
}

// ListTasks returns open tasks from one project, optionally filtered by text.
func (c *Client) ListTasks(projectID, query string, limit int) ([]Task, error) {
	url := c.taskListURL(taskListOptions{
		projectID: projectID,
		query:     query,
		openOnly:  true,
		sort:      sortNewestCreated,
		pageSize:  limit,
	})

	body, err := c.execRequest(http.MethodGet, url, nil)
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
// through the graph the trigger feeds, and that graph reads attributes Task
// does not keep.
func (c *Client) ListNewestOpenTaskDocuments(projectID string, limit int) ([]map[string]any, error) {
	return c.listTaskDocuments(taskListOptions{
		projectID: projectID,
		openOnly:  true,
		sort:      sortNewestCreated,
		pageSize:  limit,
	})
}

// ListChangedTaskDocuments returns one page of the project's tasks, most
// recently changed first. The onTask trigger reads these to find what changed
// since its last poll, so closed tasks are included: closing a task is a
// change the trigger can be configured to report.
func (c *Client) ListChangedTaskDocuments(projectID string, page, pageSize int) ([]map[string]any, error) {
	return c.listTaskDocuments(taskListOptions{
		projectID: projectID,
		sort:      sortNewestUpdated,
		page:      page,
		pageSize:  pageSize,
	})
}

func (c *Client) listTaskDocuments(options taskListOptions) ([]map[string]any, error) {
	body, err := c.execRequest(http.MethodGet, c.taskListURL(options), nil)
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

// taskListURL asks for one page of a project's tasks.
func (c *Client) taskListURL(options taskListOptions) string {
	params := url.Values{}
	params.Set("filter[project_id]", options.projectID)
	params.Set("page[size]", strconv.Itoa(options.pageSize))
	params.Set("sort", options.sort)

	if options.page > 0 {
		params.Set("page[number]", strconv.Itoa(options.page))
	}

	if options.openOnly {
		params.Set("filter[status]", "1")
	}

	if query := strings.TrimSpace(options.query); query != "" {
		params.Set("filter[query]", query)
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
