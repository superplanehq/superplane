package github

import (
	"github.com/superplanehq/superplane/pkg/core"
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
			"Issues": {
				{ReadOnly: true, Action: &GetIssue{}},
				{ReadOnly: true, Trigger: &OnIssue{}},
				{ReadOnly: true, Trigger: &OnIssueComment{}},
				{ReadOnly: false, Action: &CreateIssue{}},
				{ReadOnly: false, Action: &UpdateIssue{}},
				{ReadOnly: false, Action: &CreateIssueComment{}},
				{ReadOnly: false, Action: &RemoveIssueLabel{}},
				{ReadOnly: false, Action: &RemoveIssueAssignee{}},
				{ReadOnly: false, Action: &AddIssueLabel{}},
				{ReadOnly: false, Action: &AddIssueAssignee{}},
			},
			"Pull Requests": {
				{ReadOnly: true, Trigger: &OnPullRequest{}},
				{ReadOnly: true, Trigger: &OnPRComment{}},
				{ReadOnly: true, Trigger: &OnPRReviewComment{}},
				{ReadOnly: false, Action: &CreateReview{}},
				{ReadOnly: false, Action: &AddReaction{}},
			},
			"Contents": {
				{ReadOnly: true, Action: &GetRelease{}},
				{ReadOnly: true, Action: &GetRepositoryPermission{}},
				{ReadOnly: true, Trigger: &OnBranchCreated{}},
				{ReadOnly: true, Trigger: &OnPush{}},
				{ReadOnly: true, Trigger: &OnRelease{}},
				{ReadOnly: true, Trigger: &OnTagCreated{}},
				{ReadOnly: false, Action: &CreateRelease{}},
				{ReadOnly: false, Action: &UpdateRelease{}},
				{ReadOnly: false, Action: &DeleteRelease{}},
			},
			"Actions": {
				{ReadOnly: false, Action: &RunWorkflow{}},
				{ReadOnly: true, Trigger: &OnWorkflowRun{}},
				{ReadOnly: true, Action: &GetWorkflowUsage{}},
			},
			"Commit Statuses": {
				{ReadOnly: false, Action: &PublishCommitStatus{}},
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
