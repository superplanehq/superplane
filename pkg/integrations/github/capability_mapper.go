package github

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/actions"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/admin"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/checks"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/contents"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/deployments"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/issues"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/metadata"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/statuses"
)

const (
	PermissionScopeRepository   = "Repository"
	PermissionScopeOrganization = "Organization"

	//
	// Repository-scoped permissions
	//
	PermissionIssues         = "Issues"
	PermissionContents       = "Contents"
	PermissionPullRequests   = "Pull Requests"
	PermissionActions        = "Actions"
	PermissionChecks         = "Checks"
	PermissionCommitStatuses = "Commit Statuses"
	PermissionDeployments    = "Deployments"
	PermissionMetadata       = "Metadata"

	//
	// Organization-scoped permissions
	//
	PermissionAdministration = "Organization Administration"
)

type CapabilityMapper struct {
	Groups map[string]GroupDef
}

type GroupDef struct {
	PermissionScope string
	Capabilities    []CapabilityDef
}

type CapabilityDef struct {
	ReadOnly bool
	Action   core.Action
	Trigger  core.Trigger
}

func NewCapabilityMapper() *CapabilityMapper {
	return &CapabilityMapper{
		Groups: map[string]GroupDef{
			PermissionActions: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: false, Action: &actions.RunWorkflow{}},
					{ReadOnly: true, Trigger: &actions.OnWorkflowRun{}},
				},
			},
			PermissionChecks: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Action: &checks.ListCheckRunsForRef{}},
					{ReadOnly: true, Trigger: &checks.OnCheckRun{}},
				},
			},
			PermissionCommitStatuses: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Trigger: &statuses.OnCommitStatus{}},
					{ReadOnly: true, Action: &statuses.GetCombinedCommitStatus{}},
					{ReadOnly: false, Action: &statuses.PublishCommitStatus{}},
				},
			},
			PermissionDeployments: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: false, Action: &deployments.CreateDeployment{}},
					{ReadOnly: false, Action: &deployments.CreateDeploymentStatus{}},
				},
			},
			PermissionContents: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
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
			PermissionIssues: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Action: &issues.GetIssue{}},
					{ReadOnly: true, Trigger: &issues.OnIssue{}},
					{ReadOnly: true, Trigger: &issues.OnIssueComment{}},
					{ReadOnly: false, Action: &issues.CreateIssue{}},
					{ReadOnly: false, Action: &issues.UpdateIssue{}},
					{ReadOnly: false, Action: &issues.CreateIssueComment{}},
					{ReadOnly: false, Action: &issues.UpdateIssueComment{}},
					{ReadOnly: false, Action: &issues.RemoveIssueLabel{}},
					{ReadOnly: false, Action: &issues.RemoveIssueAssignee{}},
					{ReadOnly: false, Action: &issues.AddIssueLabel{}},
					{ReadOnly: false, Action: &issues.AddIssueAssignee{}},
				},
			},
			PermissionMetadata: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Action: &metadata.GetRepositoryPermission{}},
				},
			},
			PermissionAdministration: {
				PermissionScope: PermissionScopeOrganization,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Action: &admin.GetWorkflowUsage{}},
				},
			},
			PermissionPullRequests: {
				PermissionScope: PermissionScopeRepository,
				Capabilities: []CapabilityDef{
					{ReadOnly: true, Trigger: &pulls.OnPullRequest{}},
					{ReadOnly: true, Trigger: &pulls.OnPRComment{}},
					{ReadOnly: true, Trigger: &pulls.OnPRReviewComment{}},
					{ReadOnly: false, Action: &pulls.CreateReview{}},
					{ReadOnly: false, Action: &pulls.AddReaction{}},
					{ReadOnly: false, Action: &pulls.CreatePullRequest{}},
					{ReadOnly: false, Action: &pulls.MergePullRequest{}},
					{ReadOnly: false, Action: &pulls.MarkPullRequestReadyForReview{}},
					{ReadOnly: false, Action: &pulls.AddPullRequestReviewers{}},
					{ReadOnly: false, Action: &pulls.UpdatePullRequest{}},
				},
			},
		},
	}
}

func (m *CapabilityMapper) AllNames() []string {
	out := []string{}
	for _, group := range m.Groups {
		for _, capability := range group.Capabilities {
			out = append(out, m.capabilityName(capability))
		}
	}
	return out
}

func (m *CapabilityMapper) ForOwnerType(ownerType string) []string {
	if ownerType == common.OwnerTypeUser {
		return m.ForUserAccount()
	}
	return m.ForOrg()
}

func (m *CapabilityMapper) ForOrg() []string {
	out := []string{}
	for _, group := range m.Groups {
		if group.PermissionScope == PermissionScopeOrganization || group.PermissionScope == PermissionScopeRepository {
			for _, capability := range group.Capabilities {
				out = append(out, m.capabilityName(capability))
			}
		}
	}

	return out
}

func (m *CapabilityMapper) ForUserAccount() []string {
	out := []string{}
	for _, group := range m.Groups {
		if group.PermissionScope == PermissionScopeRepository {
			for _, capability := range group.Capabilities {
				out = append(out, m.capabilityName(capability))
			}
		}
	}
	return out
}

func (m *CapabilityMapper) capabilityName(capability CapabilityDef) string {
	if capability.Action != nil {
		return capability.Action.Name()
	}

	if capability.Trigger != nil {
		return capability.Trigger.Name()
	}

	return ""
}

type LookupEntry struct {
	Permission string

	// Using uint8 here to easily compare if an access level is greater than another.
	// 0. Read
	// 1. Read & Write
	Access uint8
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

			level := uint8(0)
			if !c.ReadOnly {
				level = 1
			}

			out[name] = LookupEntry{Permission: resourceName, Access: level}
		}
	}

	return out
}

func (m *CapabilityMapper) NewPermissionSet(capabilities []string) PermissionSet {
	lookup := m.buildLookup()
	out := PermissionSet{
		Repository:   map[string]uint8{},
		Organization: map[string]uint8{},
	}

	for _, capability := range capabilities {
		c, ok := lookup[capability]
		if !ok {
			continue
		}

		var x map[string]uint8
		if m.Groups[c.Permission].PermissionScope == PermissionScopeRepository {
			x = out.Repository
		} else {
			x = out.Organization
		}

		if _, ok := x[c.Permission]; !ok {
			x[c.Permission] = c.Access
		} else {
			if c.Access > x[c.Permission] {
				x[c.Permission] = c.Access
			}
		}
	}

	return out
}

type PermissionSet struct {
	Repository   map[string]uint8
	Organization map[string]uint8
}

/*
 * Compares one permission set with another and returns a new permission set.
 */
func FindPermissionUpdates(existing PermissionSet, requested PermissionSet) PermissionSet {
	diff := PermissionSet{
		Repository:   map[string]uint8{},
		Organization: map[string]uint8{},
	}

	for resource, requestedAccess := range requested.Repository {
		existingAccess, ok := existing.Repository[resource]
		if !ok {
			diff.Repository[resource] = requestedAccess
			continue
		}

		if requestedAccess > existingAccess {
			diff.Repository[resource] = requestedAccess
		}
	}

	for resource, requestedAccess := range requested.Organization {
		existingAccess, ok := existing.Organization[resource]
		if !ok {
			diff.Organization[resource] = requestedAccess
			continue
		}

		if requestedAccess > existingAccess {
			diff.Organization[resource] = requestedAccess
		}
	}

	return diff
}

func (p *PermissionSet) IsEmpty() bool {
	return len(p.Repository) == 0 && len(p.Organization) == 0
}

type Permission struct {
	Name   string
	Scope  string
	Access string
}

func (p *PermissionSet) ForHuman() []Permission {
	permissions := []Permission{}

	for resource, permission := range p.Repository {
		permissions = append(permissions, Permission{
			Name:   resource,
			Scope:  PermissionScopeRepository,
			Access: p.accessString(permission),
		})
	}

	for resource, permission := range p.Organization {
		permissions = append(permissions, Permission{
			Name:   resource,
			Scope:  PermissionScopeOrganization,
			Access: p.accessString(permission),
		})
	}

	return permissions
}

func (p *PermissionSet) accessString(permission uint8) string {
	if permission == 1 {
		return "Read & Write"
	}
	return "Read"
}

func (p *PermissionSet) ForAppManifest() map[string]string {
	permissions := map[string]string{}

	for resource, permission := range p.Repository {
		permissions[p.permissionForAppManifest(resource)] = p.accessForAppManifest(permission)
	}

	for resource, permission := range p.Organization {
		permissions[p.permissionForAppManifest(resource)] = p.accessForAppManifest(permission)
	}

	return permissions
}

func (p *PermissionSet) accessForAppManifest(level uint8) string {
	if level == 1 {
		return "write"
	}
	return "read"
}

func (p *PermissionSet) permissionForAppManifest(r string) string {
	switch r {
	case PermissionIssues:
		return "issues"
	case PermissionPullRequests:
		return "pull_requests"
	case PermissionContents:
		return "contents"
	case PermissionActions:
		return "actions"
	case PermissionChecks:
		return "checks"
	case PermissionCommitStatuses:
		return "statuses"
	case PermissionDeployments:
		return "deployments"
	case PermissionAdministration:
		return "organization_administration"
	case PermissionMetadata:
		return "metadata"
	default:
		return ""
	}
}
