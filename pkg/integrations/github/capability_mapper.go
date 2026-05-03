package github

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/actions"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/admin"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/contents"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/issues"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/metadata"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/statuses"
)

const (
	PermissionScopeRepository   = "repository"
	PermissionScopeOrganization = "organization"

	//
	// Repository-scoped permissions
	//
	ResourceIssues         = "Issues"
	ResourceContents       = "Contents"
	ResourcePullRequests   = "Pull Requests"
	ResourceActions        = "Actions"
	ResourceCommitStatuses = "Commit Statuses"
	ResourceMetadata       = "Metadata"

	//
	// Organization-scoped permissions
	//
	ResourceAdministration = "Administration"
)

type CapabilityMapper struct {
	Groups map[string]X
}

type X struct {
	PermissionScope string
	Capabilities    []C
}

type C struct {
	ReadOnly bool
	Action   core.Action
	Trigger  core.Trigger
}

func NewCapabilityMapper() *CapabilityMapper {
	return &CapabilityMapper{
		Groups: map[string]X{
			ResourceActions: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: false, Action: &actions.RunWorkflow{}},
					{ReadOnly: true, Trigger: &actions.OnWorkflowRun{}},
				},
			},
			ResourceCommitStatuses: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: false, Action: &statuses.PublishCommitStatus{}},
				},
			},
			ResourceContents: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: true, Action: &contents.GetRelease{}},
					{ReadOnly: true, Trigger: &contents.OnBranchCreated{}},
					{ReadOnly: true, Trigger: &contents.OnPush{}},
					{ReadOnly: true, Trigger: &contents.OnRelease{}},
					{ReadOnly: true, Trigger: &contents.OnTagCreated{}},
					{ReadOnly: false, Action: &contents.CreateRelease{}},
					{ReadOnly: false, Action: &contents.UpdateRelease{}},
					{ReadOnly: false, Action: &contents.DeleteRelease{}},
				},
			},
			ResourceIssues: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: true, Action: &issues.GetIssue{}},
					{ReadOnly: true, Trigger: &issues.OnIssue{}},
					{ReadOnly: true, Trigger: &issues.OnIssueComment{}},
					{ReadOnly: false, Action: &issues.CreateIssue{}},
					{ReadOnly: false, Action: &issues.UpdateIssue{}},
					{ReadOnly: false, Action: &issues.CreateIssueComment{}},
					{ReadOnly: false, Action: &issues.RemoveIssueLabel{}},
					{ReadOnly: false, Action: &issues.RemoveIssueAssignee{}},
					{ReadOnly: false, Action: &issues.AddIssueLabel{}},
					{ReadOnly: false, Action: &issues.AddIssueAssignee{}},
				},
			},
			ResourceMetadata: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: true, Action: &metadata.GetRepositoryPermission{}},
				},
			},
			ResourceAdministration: X{
				PermissionScope: PermissionScopeOrganization,
				Capabilities: []C{
					{ReadOnly: true, Action: &admin.GetWorkflowUsage{}},
				},
			},
			ResourcePullRequests: X{
				PermissionScope: PermissionScopeRepository,
				Capabilities: []C{
					{ReadOnly: true, Trigger: &pulls.OnPullRequest{}},
					{ReadOnly: true, Trigger: &pulls.OnPRComment{}},
					{ReadOnly: true, Trigger: &pulls.OnPRReviewComment{}},
					{ReadOnly: false, Action: &pulls.CreateReview{}},
					{ReadOnly: false, Action: &pulls.AddReaction{}},
				},
			},
		},
	}
}

/*
 * Returns two sets of permissions: permissions for the repository and permissions for the organization.
 */
func (m *CapabilityMapper) PermissionsForPAT(capabilities []string) (map[string]string, map[string]string) {
	lookup := m.buildLookup()
	repoPermissions := map[string]string{}
	orgPermissions := map[string]string{}

	for _, capability := range capabilities {
		if c, ok := lookup[capability]; ok {
			var x map[string]string
			if m.Groups[c.Resource].PermissionScope == PermissionScopeRepository {
				x = repoPermissions
			} else {
				x = orgPermissions
			}

			if !c.ReadOnly {
				x[c.Resource] = "Read & Write"
				continue
			}

			//
			// Do not override the permission if it already exists and is "Read & Write"
			//
			if c.ReadOnly && x[c.Resource] != "Read & Write" {
				x[c.Resource] = "Read"
			}
		}
	}

	return repoPermissions, orgPermissions
}

func (m *CapabilityMapper) PermissionsForApp(capabilities []string) map[string]string {
	lookup := m.buildLookup()
	out := map[string]string{}
	for _, capability := range capabilities {
		if c, ok := lookup[capability]; ok {
			if !c.ReadOnly {
				out[m.lookupResourceForApp(c.Resource)] = "write"
				continue
			}

			//
			// Do not override the permission if it already exists and is "Read & Write"
			//
			if c.ReadOnly && out[c.Resource] != "Read & Write" {
				out[m.lookupResourceForApp(c.Resource)] = "read"
			}
		}
	}

	return out
}

func (m *CapabilityMapper) lookupResourceForApp(r string) string {
	switch r {
	case ResourceIssues:
		return "issues"
	case ResourcePullRequests:
		return "pull_requests"
	case ResourceContents:
		return "contents"
	case ResourceActions:
		return "actions"
	case ResourceCommitStatuses:
		return "statuses"
	case ResourceAdministration:
		return "organization_administration"
	case ResourceMetadata:
		return "metadata"
	default:
		return ""
	}
}

type LookupEntry struct {
	Resource string
	ReadOnly bool
}

func (m CapabilityMapper) buildLookup() map[string]LookupEntry {
	out := map[string]LookupEntry{}
	for resourceName, group := range m.Groups {
		for _, c := range group.Capabilities {
			var name string
			if c.Action != nil {
				name = c.Action.Name()
			} else if c.Trigger != nil {
				name = c.Trigger.Name()
			}

			out[name] = LookupEntry{Resource: resourceName, ReadOnly: c.ReadOnly}
		}
	}

	return out
}
