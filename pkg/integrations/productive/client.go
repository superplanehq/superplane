package productive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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

	// sortNewestCreated orders tasks by when they appeared, newest first.
	sortNewestCreated = "-created_at"

	// webhooksLimitExceededCode is the JSON:API error code Productive.io
	// answers webhook registration with on plans that do not include
	// webhooks. It is used to turn a 403 into a message a user can act on,
	// rather than a bare HTTP status.
	webhooksLimitExceededCode = "webhooks_limit_exceeded"
)

// ErrWebhooksLimitExceeded reports that Productive.io rejected a webhook
// registration because the organization's plan does not include webhooks.
var ErrWebhooksLimitExceeded = errors.New("productive.io plan does not include webhooks")

// ErrMissingWritePermission reports that the API token can authenticate but
// cannot create webhooks. Error returns the message shown to the user.
var ErrMissingWritePermission error = missingWritePermissionError{}

type missingWritePermissionError struct{}

func (missingWritePermissionError) Error() string {
	return "This API token cannot create webhooks. In Productive, go to Settings > API Integrations and create a token with read and write access."
}

// Client talks to Productive.io's JSON:API v2 API using an API token and an
// organization id, both sent as headers on every request.
type Client struct {
	APIToken       string
	OrganizationID string
	BaseURL        string
	http           core.HTTPContext
}

type responseError struct {
	statusCode int
	body       string
}

func (e *responseError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.statusCode, e.body)
}

func IsNotFoundError(err error) bool {
	code, ok := responseStatusCode(err)
	return ok && code == http.StatusNotFound
}

func responseStatusCode(err error) (int, bool) {
	var responseErr *responseError
	if !errors.As(err, &responseErr) {
		return 0, false
	}
	return responseErr.statusCode, true
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	if strings.TrimSpace(string(apiToken)) == "" {
		return nil, fmt.Errorf("missing Productive API token")
	}

	organizationID, err := ctx.GetConfig("organizationId")
	if err != nil {
		return nil, fmt.Errorf("error getting organizationId: %v", err)
	}

	if strings.TrimSpace(string(organizationID)) == "" {
		return nil, fmt.Errorf("missing Productive organization id")
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
		if res.StatusCode == http.StatusForbidden && hasErrorCode(responseBody, webhooksLimitExceededCode) {
			return nil, fmt.Errorf("%w: %s", ErrWebhooksLimitExceeded, string(responseBody))
		}

		return nil, &responseError{statusCode: res.StatusCode, body: string(responseBody)}
	}

	return responseBody, nil
}

// apiErrorResponse is the JSON:API error shape Productive.io answers a failed
// request with, trimmed to the field that classifies the error.
type apiErrorResponse struct {
	Errors []struct {
		Code string `json:"code"`
	} `json:"errors"`
}

// hasErrorCode reports whether one of a JSON:API error response's errors
// carries the given code. A body that cannot be parsed as such simply
// answers false, since it then carries no code to find.
func hasErrorCode(body []byte, code string) bool {
	response := apiErrorResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return false
	}

	for _, apiErr := range response.Errors {
		if apiErr.Code == code {
			return true
		}
	}

	return false
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

// ValidateWebhookPermission confirms the token can create webhooks.
//
// The probe posts a webhook with empty attributes. Productive rejects that
// body when the token can write, and rejects the request earlier when it
// cannot. A webhook that is created anyway is deleted before this returns.
func (c *Client) ValidateWebhookPermission() error {
	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "webhooks",
			"attributes": map[string]any{},
		},
	})
	if err != nil {
		return fmt.Errorf("error building webhook permission probe: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, fmt.Sprintf("%s/webhooks", c.BaseURL), bytes.NewReader(body))
	if err == nil {
		return c.deleteCreatedWebhook(responseBody)
	}
	if errors.Is(err, ErrWebhooksLimitExceeded) {
		return err
	}

	code, ok := responseStatusCode(err)
	if !ok {
		return err
	}
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %v", ErrMissingWritePermission, err)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return nil
	default:
		return err
	}
}

func (c *Client) deleteCreatedWebhook(responseBody []byte) error {
	response := resourceResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("error parsing webhook: %v", err)
	}
	if response.Data.ID == "" {
		return fmt.Errorf("productive.io created a webhook without an id")
	}
	if err := c.DeleteWebhook(response.Data.ID); err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}
	return nil
}

func webhookCreateError(err error) error {
	if errors.Is(err, ErrWebhooksLimitExceeded) {
		return err
	}
	code, ok := responseStatusCode(err)
	if ok && (code == http.StatusUnauthorized || code == http.StatusForbidden) {
		return fmt.Errorf("%w: %v", ErrMissingWritePermission, err)
	}
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
	TaskListID  string
	Closed      bool
	TypeID      int
}

// TaskList is a Productive.io task list inside one project.
type TaskList struct {
	ID   string
	Name string
}

func (t Task) IsKeyTask() bool {
	return t.TypeID == TaskTypeMilestone
}

func projectFromDocument(doc resourceDocument) Project {
	name, _ := doc.Attributes["name"].(string)
	return Project{ID: doc.ID, Name: name}
}

func taskFromDocument(doc resourceDocument) Task {
	title, _ := doc.Attributes["title"].(string)
	description, _ := doc.Attributes["description"].(string)
	closed, _ := doc.Attributes["closed"].(bool)
	projectID := doc.Relationships["project"].Data.ID
	return Task{
		ID:          doc.ID,
		Number:      stringAttribute(doc.Attributes["task_number"]),
		Title:       title,
		Description: description,
		ProjectID:   projectID,
		TaskListID:  doc.Relationships["task_list"].Data.ID,
		Closed:      closed,
		TypeID:      numberAttribute(doc.Attributes["type_id"]),
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
	projectID   string
	query       string
	openOnly    bool
	regularOnly bool
	taskListIDs []string
	sort        string
	pageSize    int
}

// ListTasks returns open tasks from one project, optionally filtered by text.
// When regularOnly is set, key tasks (milestones) are omitted. A non-empty
// taskListIDs keeps tasks that belong to those task lists.
func (c *Client) ListTasks(projectID, query string, limit int, regularOnly bool, taskListIDs []string) ([]Task, error) {
	url := c.taskListURL(taskListOptions{
		projectID:   projectID,
		query:       query,
		openOnly:    true,
		regularOnly: regularOnly,
		taskListIDs: taskListIDs,
		sort:        sortNewestCreated,
		pageSize:    limit,
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
func (c *Client) ListNewestOpenTaskDocuments(projectID string, limit int, regularOnly bool, taskListIDs []string) ([]map[string]any, error) {
	return c.listTaskDocuments(taskListOptions{
		projectID:   projectID,
		openOnly:    true,
		regularOnly: regularOnly,
		taskListIDs: taskListIDs,
		sort:        sortNewestCreated,
		pageSize:    limit,
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

	for i := range response.Data {
		if taskListID(response.Data[i]) != "" {
			continue
		}
		if len(options.taskListIDs) == 1 {
			// A list filtered to one task list can omit the relationship.
			// The intake filter reads that id, so write the only possible value.
			setTaskListID(response.Data[i], options.taskListIDs[0])
			continue
		}
		if len(options.taskListIDs) > 1 {
			id, _ := response.Data[i]["id"].(string)
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			task, err := c.GetTask(id)
			if err != nil || task.TaskListID == "" {
				continue
			}
			setTaskListID(response.Data[i], task.TaskListID)
		}
	}

	return response.Data, nil
}

// taskListURL asks for one page of a project's tasks.
func (c *Client) taskListURL(options taskListOptions) string {
	params := url.Values{}
	params.Set("filter[project_id]", options.projectID)
	params.Set("page[size]", strconv.Itoa(options.pageSize))
	params.Set("sort", options.sort)

	if options.openOnly {
		params.Set("filter[status]", "1")
	}

	if options.regularOnly {
		params.Set("filter[type_id]", strconv.Itoa(TaskTypeRegular))
	}

	if query := strings.TrimSpace(options.query); query != "" {
		params.Set("filter[query]", query)
	}

	if len(options.taskListIDs) > 0 {
		params.Set("filter[task_list_id]", strings.Join(options.taskListIDs, ","))
		params.Set("include", "task_list")
	}

	return fmt.Sprintf("%s/tasks?%s", c.BaseURL, params.Encode())
}

// ListTaskLists returns the active task lists of one project. Productive.io
// paginates responses, so pages are walked until a short page ends them.
func (c *Client) ListTaskLists(projectID string) ([]TaskList, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project is required")
	}

	lists := []TaskList{}
	for page := 1; page <= maxProjectPages; page++ {
		params := url.Values{}
		params.Set("filter[project_id]", projectID)
		params.Set("filter[status]", "1")
		params.Set("page[number]", strconv.Itoa(page))
		params.Set("page[size]", strconv.Itoa(projectsPageSize))

		body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/task_lists?%s", c.BaseURL, params.Encode()), nil)
		if err != nil {
			return nil, err
		}

		response := resourceListResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing task lists: %v", err)
		}

		for _, doc := range response.Data {
			name, _ := doc.Attributes["name"].(string)
			lists = append(lists, TaskList{ID: doc.ID, Name: name})
		}

		if len(response.Data) < projectsPageSize {
			break
		}
	}

	sort.Slice(lists, func(i, j int) bool {
		return strings.ToLower(lists[i].Name) < strings.ToLower(lists[j].Name)
	})
	return lists, nil
}

// GetTask returns one task by its Productive.io resource id.
func (c *Client) GetTask(id string) (*Task, error) {
	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/tasks/%s?include=task_list", c.BaseURL, url.PathEscape(id)), nil)
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

// Webhook is a Productive.io webhook subscription. Productive.io webhooks
// are organization-wide and fire for one event_id each.
type Webhook struct {
	ID             string
	SignatureToken string
}

// CreateWebhook registers an organization webhook that POSTs eventID
// deliveries to webhookURL. eventName is sent back on each delivery as
// EventHeader. Productive.io answers 403 webhooks_limit_exceeded (surfaced
// as ErrWebhooksLimitExceeded) on plans that do not include webhooks.
func (c *Client) CreateWebhook(webhookURL string, eventID int, eventName string) (*Webhook, error) {
	attributes := map[string]any{
		"name":       "SuperPlane",
		"event_id":   eventID,
		"target_url": webhookURL,
		"type_id":    WebhookTypeStandard,
	}
	if eventName != "" {
		attributes["custom_headers"] = map[string]string{
			EventHeader: eventName,
		}
	}

	payload := map[string]any{
		"data": map[string]any{
			"type":       "webhooks",
			"attributes": attributes,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error building webhook payload: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, fmt.Sprintf("%s/webhooks", c.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, webhookCreateError(err)
	}

	response := resourceResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing webhook: %v", err)
	}

	if response.Data.ID == "" {
		return nil, fmt.Errorf("productive.io did not return a webhook id")
	}

	token, _ := response.Data.Attributes["signature_token"].(string)
	return &Webhook{ID: response.Data.ID, SignatureToken: token}, nil
}

// DeleteWebhook removes a webhook by its Productive.io resource id.
func (c *Client) DeleteWebhook(id string) error {
	_, err := c.execRequest(http.MethodDelete, fmt.Sprintf("%s/webhooks/%s", c.BaseURL, url.PathEscape(id)), nil)
	return err
}
