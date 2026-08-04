package linear

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ResourceTypeTeam          = "team"
	ResourceTypeWorkflowState = "workflowState"
	ResourceTypeMember        = "member"
	ResourceTypeLabel         = "label"
	ResourceTypeProject       = "project"
	ResourceTypeAttachment    = "attachment"
)

func (l *Linear) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeTeam:
		return ListTeams(ctx)
	case ResourceTypeWorkflowState:
		return ListWorkflowStates(ctx)
	case ResourceTypeMember:
		return ListMembers(ctx)
	case ResourceTypeLabel:
		return ListLabels(ctx)
	case ResourceTypeProject:
		return ListProjects(ctx)
	case ResourceTypeAttachment:
		return ListAttachments(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

// ListTeams reads the teams captured during sync, so the picker stays responsive
// and does not spend Linear's per-user request budget on every render.
func ListTeams(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Teams))
	for _, team := range metadata.Teams {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeTeam,
			Name: fmt.Sprintf("%s (%s)", team.Name, team.Key),
			ID:   team.ID,
		})
	}

	return resources, nil
}

func ListWorkflowStates(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	teamID := ctx.Parameters["team"]
	if teamID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	states, err := client.ListWorkflowStates(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow states: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(states))
	for _, state := range states {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeWorkflowState,
			Name: state.Name,
			ID:   state.ID,
		})
	}

	return resources, nil
}

func ListMembers(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	teamID := ctx.Parameters["team"]
	if teamID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	members, err := client.ListTeamMembers(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(members))
	for _, member := range members {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeMember,
			Name: memberLabel(member),
			ID:   member.ID,
		})
	}

	return resources, nil
}

func ListLabels(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	teamID := ctx.Parameters["team"]
	if teamID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	labels, err := client.ListLabels(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(labels))
	for _, label := range labels {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeLabel,
			Name: label.Name,
			ID:   label.ID,
		})
	}

	return resources, nil
}

func ListProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	teamID := ctx.Parameters["team"]
	if teamID == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	projects, err := client.ListTeamProjects(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProject,
			Name: project.Name,
			ID:   project.ID,
		})
	}

	return resources, nil
}

// ListAttachments returns the attachments on the issue selected in the sibling
// field. It stays empty until an issue is chosen, and an issue set from an
// expression cannot be resolved here, so the picker offers nothing in that case.
func ListAttachments(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	issueID := ctx.Parameters["issue"]
	if issueID == "" || strings.Contains(issueID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	attachments, err := client.ListIssueAttachments(issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %v", err)
	}

	resources := make([]core.IntegrationResource, 0, len(attachments))
	for _, attachment := range attachments {
		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeAttachment,
			Name: attachmentLabel(attachment),
			ID:   attachment.ID,
		})
	}

	return resources, nil
}

func attachmentLabel(attachment Attachment) string {
	if attachment.Subtitle != "" {
		return fmt.Sprintf("%s (%s)", attachment.Title, attachment.Subtitle)
	}

	return attachment.Title
}

func memberLabel(member User) string {
	if member.DisplayName != "" {
		return fmt.Sprintf("%s (@%s)", member.Name, member.DisplayName)
	}

	return member.Name
}
