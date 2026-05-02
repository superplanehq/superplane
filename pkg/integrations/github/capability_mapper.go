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
	ResourceIssues         = "Issues"
	ResourceContents       = "Contents"
	ResourcePullRequests   = "Pull Requests"
	ResourceActions        = "Actions"
	ResourceCommitStatuses = "Commit Statuses"
)

type CapabilityMapper struct {
	Groups map[string][]C
}

type C struct {
	ReadOnly bool
	Action   core.Action
	Trigger  core.Trigger
}

func NewCapabilityMapper() *CapabilityMapper {
	return &CapabilityMapper{
		Groups: map[string][]C{
			"Actions": {
				{ReadOnly: false, Action: &actions.RunWorkflow{}},
				{ReadOnly: true, Trigger: &actions.OnWorkflowRun{}},
			},
			"Admin": {
				{ReadOnly: true, Action: &admin.GetWorkflowUsage{}},
			},
			"Commit Statuses": {
				{ReadOnly: false, Action: &statuses.PublishCommitStatus{}},
			},
			"Contents": {
				{ReadOnly: true, Action: &contents.GetRelease{}},
				{ReadOnly: true, Trigger: &contents.OnBranchCreated{}},
				{ReadOnly: true, Trigger: &contents.OnPush{}},
				{ReadOnly: true, Trigger: &contents.OnRelease{}},
				{ReadOnly: true, Trigger: &contents.OnTagCreated{}},
				{ReadOnly: false, Action: &contents.CreateRelease{}},
				{ReadOnly: false, Action: &contents.UpdateRelease{}},
				{ReadOnly: false, Action: &contents.DeleteRelease{}},
			},
			"Issues": {
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
			"Metadata": {
				{ReadOnly: true, Action: &metadata.GetRepositoryPermission{}},
			},
			"Pull Requests": {
				{ReadOnly: true, Trigger: &pulls.OnPullRequest{}},
				{ReadOnly: true, Trigger: &pulls.OnPRComment{}},
				{ReadOnly: true, Trigger: &pulls.OnPRReviewComment{}},
				{ReadOnly: false, Action: &pulls.CreateReview{}},
				{ReadOnly: false, Action: &pulls.AddReaction{}},
			},
		},
	}
}

func (m *CapabilityMapper) PermissionsForPAT(requestedCapabilities []string) map[string]string {
	lookup := m.buildLookup()
	out := map[string]string{}
	for _, capability := range requestedCapabilities {
		if c, ok := lookup[capability]; ok {
			if !c.ReadOnly {
				out[c.Resource] = "Read & Write"
				continue
			}

			//
			// Do not override the permission if it already exists and is "Read & Write"
			//
			if c.ReadOnly && out[c.Resource] != "Read & Write" {
				out[c.Resource] = "Read"
			}
		}
	}

	//
	// We always include the webhooks permission,
	// because SuperPlane needs that for creating webhooks for triggers.
	//
	out["Webhooks"] = "Read & Write"

	return out
}

func (m *CapabilityMapper) PermissionsForApp(requestedCapabilities []string) map[string]string {
	lookup := m.buildLookup()
	out := map[string]string{}
	for _, capability := range requestedCapabilities {
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

	//
	// We always include the repository_hooks permission,
	// because SuperPlane needs that for creating webhooks for triggers.
	//
	out["repository_hooks"] = "write"

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
		for _, c := range group {
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
