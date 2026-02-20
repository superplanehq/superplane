package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) RunPipeline(request RunPipelineRequest) (*RunPipelineResponse, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

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
		readStringPath(data, "data", "planExecutionId"),
		readStringPath(data, "data", "planExecution", "uuid"),
		readStringPath(data, "data", "planExecution", "executionUuid"),
		readStringPath(data, "data", "planExecution", "metadata", "executionUuid"),
		readString(data["planExecutionId"]),
		readString(data["executionId"]),
	)
	if executionID == "" {
		return nil, fmt.Errorf("run pipeline response missing execution id: %s", string(body))
	}

	return &RunPipelineResponse{ExecutionID: executionID}, nil
}

func (c *Client) GetExecutionSummary(executionID string) (*ExecutionSummary, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

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

	summary := mapExecutionSummaryItem(item)
	if summary.ExecutionID == "" {
		summary.ExecutionID = executionID
	}

	return &summary, nil
}

func mapExecutionSummaryItem(item map[string]any) ExecutionSummary {
	return ExecutionSummary{
		ExecutionID: firstNonEmpty(
			readString(item["planExecutionId"]),
			readString(item["executionId"]),
			readString(item["id"]),
			readStringPath(item, "planExecution", "uuid"),
			readStringPath(item, "planExecution", "executionUuid"),
			readStringPath(item, "planExecution", "metadata", "executionUuid"),
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
		),
		PlanExecutionURL: firstNonEmpty(
			readString(item["planExecutionUrl"]),
			readString(item["executionUrl"]),
			readStringPath(item, "planExecution", "planExecutionUrl"),
		),
		StartedAt: firstNonEmpty(
			readString(item["startTs"]),
			readString(item["startedAt"]),
			readString(item["startTime"]),
			readStringPath(item, "planExecution", "startTs"),
		),
		EndedAt: firstNonEmpty(
			readString(item["endTs"]),
			readString(item["endedAt"]),
			readString(item["endTime"]),
			readStringPath(item, "planExecution", "endTs"),
		),
	}
}

func (c *Client) ListExecutionSummariesPage(
	page int,
	size int,
	pipelineIdentifier string,
) ([]ExecutionSummary, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = DefaultExecutionsLimit
	}
	if size > 100 {
		size = 100
	}

	query := c.scopeQuery()
	query.Set("page", strconv.Itoa(page))
	query.Set("size", strconv.Itoa(size))

	payload := map[string]any{
		"filterType": "PipelineExecution",
	}

	pipelineIdentifier = strings.TrimSpace(pipelineIdentifier)
	if pipelineIdentifier != "" {
		payload["pipelineIdentifier"] = pipelineIdentifier
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
		return nil, fmt.Errorf("failed to parse execution summaries response: %w", err)
	}

	items := firstContentArray(response)
	summaries := make([]ExecutionSummary, 0, len(items))
	for _, item := range items {
		summary := mapExecutionSummaryItem(item)
		if strings.TrimSpace(summary.ExecutionID) == "" {
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func IsExecutionSummaryPipelineIdentifierFilterUnsupported(err error) bool {
	apiError := &APIError{}
	if !errors.As(err, &apiError) {
		return false
	}
	if apiError.StatusCode != http.StatusBadRequest && apiError.StatusCode != http.StatusUnprocessableEntity {
		return false
	}

	body := strings.ToLower(strings.TrimSpace(apiError.Body))
	if body == "" {
		return false
	}
	if !strings.Contains(body, "pipelineidentifier") {
		return false
	}

	unsupportedMarkers := []string{
		"unknown field",
		"unrecognized field",
		"unsupported",
		"invalid field",
		"field does not exist",
		"cannot deserialize",
	}

	for _, marker := range unsupportedMarkers {
		if strings.Contains(body, marker) {
			return true
		}
	}

	return false
}

func (c *Client) ListOrganizations() ([]Organization, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

	ngQuery := c.accountQuery()
	ngQuery.Set("page", "0")
	ngQuery.Set("size", "100")

	v1Query := c.accountQuery()
	v1Query.Set("page", "0")
	v1Query.Set("size", "100")

	attempts := []struct {
		method            string
		endpoint          string
		query             url.Values
		payload           any
		includeJSONHeader bool
	}{
		{
			method:            http.MethodGet,
			endpoint:          "/ng/api/organizations",
			query:             ngQuery,
			includeJSONHeader: false,
		},
		{
			method:            http.MethodGet,
			endpoint:          "/v1/orgs",
			query:             v1Query,
			includeJSONHeader: false,
		},
	}

	var (
		lastErr               error
		lastParseErr          error
		hasSuccessfulResponse bool
	)

	for _, attempt := range attempts {
		_, body, err := c.execRequest(
			attempt.method,
			attempt.endpoint,
			attempt.query,
			attempt.payload,
			attempt.includeJSONHeader,
		)
		if err != nil {
			lastErr = err
			continue
		}
		hasSuccessfulResponse = true

		organizations, parseErr := organizationsFromBody(body)
		if parseErr != nil {
			lastParseErr = parseErr
			continue
		}
		lastParseErr = nil
		if len(organizations) > 0 {
			return organizations, nil
		}
	}

	if !hasSuccessfulResponse && lastErr != nil {
		return nil, lastErr
	}
	if hasSuccessfulResponse && lastParseErr != nil {
		return nil, lastParseErr
	}

	return []Organization{}, nil
}

func (c *Client) ListProjects(orgID string) ([]Project, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return []Project{}, nil
	}

	accountQuery := c.accountQuery()
	accountQuery.Set("orgIdentifier", orgID)
	accountQuery.Set("page", "0")
	accountQuery.Set("size", "100")

	v1Query := c.accountQuery()
	v1Query.Set("page", "0")
	v1Query.Set("size", "100")

	attempts := []struct {
		method            string
		endpoint          string
		query             url.Values
		payload           any
		includeJSONHeader bool
	}{
		{
			method:            http.MethodGet,
			endpoint:          "/ng/api/aggregate/projects",
			query:             accountQuery,
			includeJSONHeader: false,
		},
		{
			method:            http.MethodGet,
			endpoint:          fmt.Sprintf("/v1/orgs/%s/projects", url.PathEscape(orgID)),
			query:             v1Query,
			includeJSONHeader: false,
		},
	}

	var (
		lastErr               error
		lastParseErr          error
		hasSuccessfulResponse bool
	)

	for _, attempt := range attempts {
		_, body, err := c.execRequest(
			attempt.method,
			attempt.endpoint,
			attempt.query,
			attempt.payload,
			attempt.includeJSONHeader,
		)
		if err != nil {
			lastErr = err
			continue
		}
		hasSuccessfulResponse = true

		projects, parseErr := projectsFromBody(body)
		if parseErr != nil {
			lastParseErr = parseErr
			continue
		}
		lastParseErr = nil
		if len(projects) > 0 {
			return projects, nil
		}
	}

	if !hasSuccessfulResponse && lastErr != nil {
		return nil, lastErr
	}
	if hasSuccessfulResponse && lastParseErr != nil {
		return nil, lastParseErr
	}

	return []Project{}, nil
}

func (c *Client) ListPipelines() ([]Pipeline, error) {
	if err := c.ensureAccountID(); err != nil {
		return nil, err
	}

	query := c.scopeQuery()
	query.Set("page", "0")
	query.Set("size", "100")

	_, body, err := c.execRequest(
		http.MethodPost,
		"/pipeline/api/pipelines/list",
		query,
		map[string]any{"filterType": "PipelineSetup"},
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
		identifier := readString(item["identifier"])
		name := firstNonEmpty(readString(item["name"]), identifier)
		if identifier == "" {
			continue
		}
		pipelines = append(pipelines, Pipeline{Identifier: identifier, Name: name})
	}

	return pipelines, nil
}

func parseRef(ref string) (string, string) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", ""
	}

	switch {
	case strings.HasPrefix(trimmed, "refs/heads/"):
		return strings.TrimPrefix(trimmed, "refs/heads/"), ""
	case strings.HasPrefix(trimmed, "refs/tags/"):
		return "", strings.TrimPrefix(trimmed, "refs/tags/")
	default:
		// Treat plain ref values (e.g. "main") as branch names so ref input is
		// never silently dropped.
		return trimmed, ""
	}
}

func firstExecutionSummaryItem(response map[string]any) map[string]any {
	if item := firstMapFromArray(readAnyPath(response, "data", "content")); item != nil {
		return item
	}
	return firstMapFromArray(readAnyPath(response, "data", "items"))
}

func firstContentArray(response map[string]any) []map[string]any {
	if items := arrayOfMaps(readAnyPath(response, "data", "content")); len(items) > 0 {
		return items
	}
	return arrayOfMaps(readAnyPath(response, "data", "items"))
}

func organizationsFromResponse(response map[string]any) []Organization {
	collections := []any{
		readAnyPath(response, "data", "content"),
		readAnyPath(response, "data", "items"),
		readAnyPath(response, "data", "organizations"),
		readAnyPath(response, "content"),
		readAnyPath(response, "items"),
	}

	for _, collection := range collections {
		items := arrayOfMaps(collection)
		if len(items) == 0 {
			continue
		}

		organizations := make([]Organization, 0, len(items))
		for _, item := range items {
			organization, ok := organizationFromItem(item)
			if !ok {
				continue
			}
			organizations = append(organizations, organization)
		}

		if len(organizations) > 0 {
			return organizations
		}
	}

	return []Organization{}
}

func organizationsFromBody(body []byte) ([]Organization, error) {
	parsed := any(nil)
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse organizations response: %w", err)
	}

	switch typed := parsed.(type) {
	case map[string]any:
		return organizationsFromResponse(typed), nil
	case []any:
		return organizationsFromArray(typed), nil
	default:
		return []Organization{}, nil
	}
}

func organizationsFromArray(items []any) []Organization {
	organizations := make([]Organization, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		organization, ok := organizationFromItem(item)
		if !ok {
			continue
		}
		organizations = append(organizations, organization)
	}
	return organizations
}

func projectsFromResponse(response map[string]any) []Project {
	collections := []any{
		readAnyPath(response, "data", "content"),
		readAnyPath(response, "data", "items"),
		readAnyPath(response, "data", "projects"),
		readAnyPath(response, "content"),
		readAnyPath(response, "items"),
	}

	for _, collection := range collections {
		items := arrayOfMaps(collection)
		if len(items) == 0 {
			continue
		}

		projects := make([]Project, 0, len(items))
		for _, item := range items {
			project, ok := projectFromItem(item)
			if !ok {
				continue
			}
			projects = append(projects, project)
		}

		if len(projects) > 0 {
			return projects
		}
	}

	return []Project{}
}

func projectsFromBody(body []byte) ([]Project, error) {
	parsed := any(nil)
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse projects response: %w", err)
	}

	switch typed := parsed.(type) {
	case map[string]any:
		return projectsFromResponse(typed), nil
	case []any:
		return projectsFromArray(typed), nil
	default:
		return []Project{}, nil
	}
}

func projectsFromArray(items []any) []Project {
	projects := make([]Project, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		project, ok := projectFromItem(item)
		if !ok {
			continue
		}
		projects = append(projects, project)
	}
	return projects
}

func organizationFromItem(item map[string]any) (Organization, bool) {
	identifier := firstNonEmpty(
		readString(item["identifier"]),
		readString(item["id"]),
		readString(item["orgIdentifier"]),
		readStringPath(item, "organization", "identifier"),
		readStringPath(item, "org", "identifier"),
	)
	if strings.TrimSpace(identifier) == "" {
		return Organization{}, false
	}

	name := firstNonEmpty(
		readString(item["name"]),
		readStringPath(item, "organization", "name"),
		readStringPath(item, "org", "name"),
		identifier,
	)

	return Organization{
		Identifier: identifier,
		Name:       name,
	}, true
}

func projectFromItem(item map[string]any) (Project, bool) {
	identifier := firstNonEmpty(
		readString(item["identifier"]),
		readString(item["id"]),
		readString(item["projectIdentifier"]),
		readStringPath(item, "project", "identifier"),
		readStringPath(item, "projectResponse", "project", "identifier"),
	)
	if strings.TrimSpace(identifier) == "" {
		return Project{}, false
	}

	name := firstNonEmpty(
		readString(item["name"]),
		readStringPath(item, "project", "name"),
		readStringPath(item, "projectResponse", "project", "name"),
		identifier,
	)

	return Project{
		Identifier: identifier,
		Name:       name,
	}, true
}
