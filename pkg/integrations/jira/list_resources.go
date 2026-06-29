package jira

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const opsAlertLabelMaxRunes = 80

func (j *Jira) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		return listProjects(ctx)
	case "issueType":
		return listIssueTypes(ctx)
	case "issueStatus":
		return listIssueStatuses(ctx)
	case "assignee":
		return listAssignees(ctx)
	case "priority":
		return listPriorities(ctx)
	case "resolution":
		return listResolutions(ctx)
	case "jsmApproval":
		return listJSMApprovals(ctx)
	case "serviceDesk":
		return listServiceDesks(ctx)
	case "serviceDeskRequestType":
		return listServiceDeskRequestTypes(ctx)
	case "impact":
		return listRequestTypeFieldResources(resourceType, "impact", ctx)
	case "urgency":
		return listRequestTypeFieldResources(resourceType, "urgency", ctx)
	case "team":
		return listTeams(ctx)
	case "heartbeat":
		return listHeartbeats(ctx)
	case "issue":
		return listIssues(ctx)
	case "alert":
		return listAlerts(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listServiceDesks(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	desks, err := client.ListServiceDesks()
	if err != nil {
		return nil, fmt.Errorf("list service desks: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(desks))
	for _, desk := range desks {
		name := desk.ProjectName
		if desk.ProjectKey != "" {
			name = fmt.Sprintf("%s (%s)", desk.ProjectName, desk.ProjectKey)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "serviceDesk",
			Name: name,
			ID:   desk.ID,
		})
	}
	return resources, nil
}

func listServiceDeskRequestTypes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	deskID := strings.TrimSpace(ctx.Parameters["serviceDesk"])
	if deskID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	requestTypes, err := client.ListRequestTypes(deskID)
	if err != nil {
		return nil, fmt.Errorf("list request types: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(requestTypes))
	for _, rt := range requestTypes {
		name := rt.Name
		if rt.ID != "" {
			name = fmt.Sprintf("%s (%s)", rt.Name, rt.ID)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "serviceDeskRequestType",
			Name: name,
			ID:   rt.ID,
		})
	}
	return resources, nil
}

func listTeams(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	teams, err := client.ListOpsTeams(cloudID)
	if err != nil {
		return nil, fmt.Errorf("list operations teams: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(teams))
	for _, team := range teams {
		if strings.TrimSpace(team.TeamID) == "" {
			continue
		}
		name := strings.TrimSpace(team.TeamName)
		if name == "" {
			name = team.TeamID
		}
		resources = append(resources, core.IntegrationResource{
			Type: "team",
			Name: name,
			ID:   team.TeamID,
		})
	}
	return resources, nil
}

func listHeartbeats(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	teamID := strings.TrimSpace(ctx.Parameters["team"])
	if teamID == "" {
		return []core.IntegrationResource{}, nil
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	heartbeats, err := client.ListHeartbeats(cloudID, teamID)
	if err != nil {
		return nil, fmt.Errorf("list heartbeats: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(heartbeats))
	for _, hb := range heartbeats {
		if strings.TrimSpace(hb.Name) == "" {
			continue
		}
		name := hb.Name
		if hb.Status != "" {
			name = fmt.Sprintf("%s (%s)", hb.Name, hb.Status)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "heartbeat",
			Name: name,
			ID:   hb.Name,
		})
	}
	return resources, nil
}

func listIssues(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projectKey := strings.TrimSpace(ctx.Parameters["project"])

	if projectKey != "" {
		if resources, ok := listIssuesFromServiceDesk(client, projectKey); ok {
			return resources, nil
		}
	}

	var jql string
	if projectKey != "" {
		jql = fmt.Sprintf(`project = "%s" ORDER BY updated DESC`, jqlQuotedProjectKey(projectKey))
	} else {
		jql = "updated >= -90d ORDER BY updated DESC"
	}

	hits, err := client.SearchIssuesUpTo(jql, 500)
	if err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(hits))
	for _, hit := range hits {
		if strings.TrimSpace(hit.Key) == "" {
			continue
		}
		name := hit.Key
		if hit.Fields != nil {
			if s, ok := hit.Fields["summary"].(string); ok && strings.TrimSpace(s) != "" {
				name = fmt.Sprintf("%s (%s)", strings.TrimSpace(s), hit.Key)
			}
		}
		resources = append(resources, core.IntegrationResource{
			Type: "issue",
			Name: name,
			ID:   hit.Key,
		})
	}
	return resources, nil
}

func listIssuesFromServiceDesk(client *Client, projectKey string) ([]core.IntegrationResource, bool) {
	desks, err := client.ListServiceDesks()
	if err != nil {
		return nil, false
	}

	for _, desk := range desks {
		if !strings.EqualFold(strings.TrimSpace(desk.ProjectKey), projectKey) {
			continue
		}
		rows, err := client.ListCustomerRequestsByServiceDesk(desk.ID, 500)
		if err != nil || len(rows) == 0 {
			break
		}
		resources := make([]core.IntegrationResource, 0, len(rows))
		for _, row := range rows {
			if strings.TrimSpace(row.IssueKey) == "" {
				continue
			}
			name := row.IssueKey
			if s := strings.TrimSpace(row.Summary); s != "" {
				name = fmt.Sprintf("%s (%s)", s, row.IssueKey)
			}
			resources = append(resources, core.IntegrationResource{
				Type: "issue",
				Name: name,
				ID:   row.IssueKey,
			})
		}
		return resources, true
	}
	return nil, false
}

func listProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			projects, err := client.ListProjects()
			if err == nil {
				return projectResources(projects), nil
			}
		}
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return projectResources(metadata.Projects), nil
}

func projectResources(projects []Project) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: "project",
			Name: fmt.Sprintf("%s (%s)", project.Name, project.Key),
			ID:   project.Key,
		})
	}
	return resources
}

func listIssueTypes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectKey := ctx.Parameters["project"]
	if projectKey == "" || strings.Contains(projectKey, "{{") {
		return []core.IntegrationResource{}, nil
	}

	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	issueTypes, err := client.GetProjectIssueTypes(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list issue types: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(issueTypes))
	for _, it := range issueTypes {
		resources = append(resources, core.IntegrationResource{
			Type: "issueType",
			Name: it.Name,
			ID:   it.Name,
		})
	}
	return resources, nil
}

func listIssueStatuses(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projectKey := strings.TrimSpace(ctx.Parameters["project"])
	if projectKey != "" && !strings.Contains(projectKey, "{{") {
		statuses, err := client.GetProjectStatuses(projectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list issue statuses: %w", err)
		}
		return issueStatusResources(statuses), nil
	}

	statuses, err := client.ListGlobalStatuses()
	if err != nil {
		return nil, fmt.Errorf("failed to list issue statuses: %w", err)
	}
	return issueStatusResources(statuses), nil
}

func issueStatusResources(statuses []Status) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(statuses))
	for _, s := range statuses {
		resources = append(resources, core.IntegrationResource{
			Type: "issueStatus",
			Name: s.Name,
			ID:   s.Name,
		})
	}
	return resources
}

func listAssignees(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	projectKey := assigneeProjectKey(ctx)
	if projectKey == "" {
		return []core.IntegrationResource{}, nil
	}

	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	users, err := client.ListAssignableUsers(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignable users: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(users))
	for _, u := range users {
		name := u.DisplayName
		if u.EmailAddr != "" {
			name = fmt.Sprintf("%s (%s)", u.DisplayName, u.EmailAddr)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "assignee",
			Name: name,
			ID:   u.AccountID,
		})
	}
	return resources, nil
}

func listPriorities(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	priorities, err := client.ListPriorities()
	if err != nil {
		return nil, fmt.Errorf("failed to list priorities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(priorities))
	for _, p := range priorities {
		resources = append(resources, core.IntegrationResource{
			Type: "priority",
			Name: p.Name,
			ID:   p.Name,
		})
	}
	return resources, nil
}

// listJSMApprovals returns the pending approvals for a JSM customer request,
// so the Approve Workflow component can offer a picker instead of asking the
// user to copy an approval id by hand. Non-pending approvals are filtered out
// because they are not actionable.
func listJSMApprovals(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	issueKey := strings.TrimSpace(ctx.Parameters["issueKey"])
	if issueKey == "" || strings.Contains(issueKey, "{{") {
		return []core.IntegrationResource{}, nil
	}

	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	approvals, err := client.ListApprovals(issueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list approvals: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(approvals))
	for _, approval := range approvals {
		if !isPendingApproval(approval) {
			continue
		}
		id := approval.ID.String()
		if id == "" {
			continue
		}
		name := strings.TrimSpace(approval.Name)
		if name == "" {
			name = fmt.Sprintf("Approval %s", id)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "jsmApproval",
			Name: name,
			ID:   id,
		})
	}
	return resources, nil
}

func listResolutions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.HTTP == nil {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	resolutions, err := client.ListResolutions()
	if err != nil {
		return nil, fmt.Errorf("failed to list resolutions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(resolutions))
	for _, r := range resolutions {
		resources = append(resources, core.IntegrationResource{
			Type: "resolution",
			Name: r.Name,
			ID:   r.Name,
		})
	}
	return resources, nil
}

func listRequestTypeFieldResources(resourceType, fieldLabel string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	deskID := strings.TrimSpace(ctx.Parameters["serviceDesk"])
	reqID := strings.TrimSpace(ctx.Parameters["serviceDeskRequestType"])
	if deskID == "" || reqID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projectKey := resolveServiceDeskProjectKey(client, deskID)

	rtFields, err := client.ListRequestTypeFields(deskID, reqID)
	if err != nil {
		return nil, fmt.Errorf("list request type fields: %w", err)
	}

	field := findRequestTypeField(rtFields, fieldLabel)
	if field == nil {
		if allFields, listErr := client.ListFields(); listErr == nil {
			if gf := FindGlobalFieldByLabel(allFields, fieldLabel); gf != nil {
				field = &RequestTypeField{FieldID: gf.ID, Name: gf.Name}
			}
		}
	}

	return requestTypeFieldResources(client, field, resourceType, projectKey, fieldLabel)
}

func listAlerts(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("cloud id required for Ops alerts: %w", err)
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	rows, err := client.ListOpsAlerts(cloudID, 100)
	if err != nil {
		return nil, fmt.Errorf("list Ops alerts: %w", err)
	}
	out := make([]core.IntegrationResource, 0, len(rows))
	for _, row := range rows {
		rawID := strings.TrimSpace(opsAlertStringField(row, "id"))
		if rawID == "" {
			continue
		}
		name := opsAlertIntegrationResourceLabel(row, rawID)
		out = append(out, core.IntegrationResource{
			Type: "alert",
			Name: name,
			ID:   rawID,
		})
	}
	return out, nil
}

func findRequestTypeFieldID(fields []RequestTypeField, fieldLabel string) string {
	if f := findRequestTypeField(fields, fieldLabel); f != nil {
		return strings.TrimSpace(f.FieldID)
	}
	return ""
}

func findRequestTypeField(fields []RequestTypeField, fieldLabel string) *RequestTypeField {
	want := strings.ToLower(strings.TrimSpace(fieldLabel))

	var exactNoValues *RequestTypeField
	var contains []RequestTypeField

	for i := range fields {
		f := fields[i]
		nameLower := strings.ToLower(strings.TrimSpace(f.Name))
		if nameLower == want {
			if len(f.ValidValues) > 0 {
				return &fields[i]
			}
			if exactNoValues == nil {
				exactNoValues = &fields[i]
			}
			continue
		}
		if strings.Contains(nameLower, want) {
			contains = append(contains, f)
		}
	}

	if exactNoValues != nil {
		return exactNoValues
	}

	return bestRequestTypeFieldMatch(contains, want)
}

func bestRequestTypeFieldMatch(candidates []RequestTypeField, want string) *RequestTypeField {
	if len(candidates) == 0 {
		return nil
	}

	var best *RequestTypeField
	bestScore := -1
	for i := range candidates {
		f := &candidates[i]
		score := requestTypeFieldMatchScore(f, want)
		if len(f.ValidValues) > 0 {
			score += 100
		}
		if score > bestScore {
			bestScore = score
			best = f
		}
	}
	return best
}

func requestTypeFieldMatchScore(f *RequestTypeField, want string) int {
	nameLower := strings.ToLower(strings.TrimSpace(f.Name))
	switch {
	case nameLower == want:
		return 50
	case strings.HasPrefix(nameLower, want+" "), strings.HasPrefix(nameLower, want+"-"):
		return 40
	case strings.HasPrefix(nameLower, want):
		return 30
	case strings.Contains(nameLower, want):
		return 10
	default:
		return 0
	}
}

func requestTypeFieldResources(
	client *Client,
	field *RequestTypeField,
	resourceType, projectKey, fieldLabel string,
) ([]core.IntegrationResource, error) {
	options := loadRequestTypeFieldOptions(client, field, projectKey, fieldLabel)
	return fieldOptionsToResources(options, resourceType), nil
}

func loadRequestTypeFieldOptions(
	client *Client,
	field *RequestTypeField,
	projectKey, fieldLabel string,
) []RequestTypeFieldValue {
	if field == nil {
		if client == nil || projectKey == "" || fieldLabel == "" {
			return nil
		}
		opts, _ := client.listFieldAllowedValuesFromCreateMeta(projectKey, fieldLabel)
		return opts
	}

	options := field.ValidValues
	if len(options) == 0 && client != nil {
		options = client.ListCustomFieldOptions(field.FieldID, projectKey, fieldLabel)
	}
	return options
}

func resolveIncidentFieldID(client *Client, rtFields []RequestTypeField, projectKey, fieldLabel string) string {
	if id := findRequestTypeFieldID(rtFields, fieldLabel); id != "" {
		return id
	}
	if client == nil {
		return ""
	}
	allFields, err := client.ListFields()
	if err == nil {
		if gf := FindGlobalFieldByLabel(allFields, fieldLabel); gf != nil {
			return strings.TrimSpace(gf.ID)
		}
	}
	if projectKey != "" {
		if _, fieldID := client.listFieldAllowedValuesFromCreateMeta(projectKey, fieldLabel); fieldID != "" {
			return fieldID
		}
	}
	return ""
}

func fieldOptionsToResources(options []RequestTypeFieldValue, resourceType string) []core.IntegrationResource {
	if len(options) == 0 {
		return nil
	}
	out := make([]core.IntegrationResource, 0, len(options))
	for _, opt := range options {
		id := strings.TrimSpace(opt.Value)
		label := strings.TrimSpace(opt.Label)
		if id == "" && label == "" {
			continue
		}
		if id == "" {
			id = label
		}
		if label == "" {
			label = id
		}
		out = append(out, core.IntegrationResource{
			Type: resourceType,
			Name: label,
			ID:   id,
		})
	}
	return out
}

func resolveServiceDeskProjectKey(client *Client, serviceDeskID string) string {
	desks, err := client.ListServiceDesks()
	if err != nil {
		return ""
	}
	for _, desk := range desks {
		if desk.ID == serviceDeskID {
			return strings.TrimSpace(desk.ProjectKey)
		}
	}
	return ""
}

func assigneeProjectKey(ctx core.ListResourcesContext) string {
	projectKey := strings.TrimSpace(ctx.Parameters["project"])
	if projectKey != "" && !strings.Contains(projectKey, "{{") {
		return projectKey
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return ""
	}
	for _, p := range metadata.Projects {
		if k := strings.TrimSpace(p.Key); k != "" {
			return k
		}
	}
	return ""
}

func truncateOpsAlertLabelMessage(msg string) string {
	if utf8.RuneCountInString(msg) <= opsAlertLabelMaxRunes {
		return msg
	}
	runes := []rune(msg)
	keep := opsAlertLabelMaxRunes - utf8.RuneCountInString("...")
	if keep < 0 {
		keep = 0
	}
	return string(runes[:keep]) + "..."
}

func opsAlertIntegrationResourceLabel(row map[string]any, alertID string) string {
	msg := strings.TrimSpace(opsAlertStringField(row, "message"))
	if msg != "" {
		msg = truncateOpsAlertLabelMessage(msg)
	}
	tiny := strings.TrimSpace(opsAlertStringField(row, "tinyId"))

	switch {
	case msg != "" && tiny != "":
		return fmt.Sprintf("%s #%s", msg, tiny)
	case msg != "":
		return msg
	case tiny != "":
		return fmt.Sprintf("#%s", tiny)
	default:
		return alertID
	}
}
