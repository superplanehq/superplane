package jira

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

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
	case "serviceDesk":
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
				Type: resourceType,
				Name: name,
				ID:   desk.ID,
			})
		}
		return resources, nil

	case "serviceDeskRequestType":
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
				Type: resourceType,
				Name: name,
				ID:   rt.ID,
			})
		}
		return resources, nil

	case "impact":
		return j.listRequestTypeFieldResources(resourceType, "impact", ctx)

	case "urgency":
		return j.listRequestTypeFieldResources(resourceType, "urgency", ctx)

	case "issue":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		projectKey := strings.TrimSpace(ctx.Parameters["project"])

		if projectKey != "" {
			desks, derr := client.ListServiceDesks()
			if derr == nil {
				for _, desk := range desks {
					if strings.EqualFold(strings.TrimSpace(desk.ProjectKey), projectKey) {
						rows, rerr := client.ListCustomerRequestsByServiceDesk(desk.ID, 500)
						if rerr == nil && len(rows) > 0 {
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
									Type: resourceType,
									Name: name,
									ID:   row.IssueKey,
								})
							}
							return resources, nil
						}
						break
					}
				}
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
				Type: resourceType,
				Name: name,
				ID:   hit.Key,
			})
		}
		return resources, nil
	default:
		return []core.IntegrationResource{}, nil
	}
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

	statuses, err := client.GetProjectStatuses(projectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to list issue statuses: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(statuses))
	for _, s := range statuses {
		resources = append(resources, core.IntegrationResource{
			Type: "issueStatus",
			Name: s.Name,
			ID:   s.Name,
		})
	}
	return resources, nil
}

func listAssignees(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

func (j *Jira) listRequestTypeFieldResources(resourceType, fieldLabel string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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
