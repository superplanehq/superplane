package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__CapabilityMapper__ForOrg(t *testing.T) {
	t.Parallel()
	m := NewCapabilityMapper()

	t.Run("org capabilities are not returned for user accounts", func(t *testing.T) {
		assert.NotContains(t, m.ForUserAccount(), "github.getWorkflowUsage")
	})

	t.Run("org capabilities are returned for organizations", func(t *testing.T) {
		require.Contains(t, m.ForOrg(), "github.getWorkflowUsage")
	})
}

func Test__CapabilityMapper__FindPermissionUpdates(t *testing.T) {
	t.Parallel()

	t.Run("empty requested yields empty diff", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{Repository: map[string]uint8{"Issues": 1}},
			PermissionSet{Repository: map[string]uint8{}},
		)
		assert.True(t, diff.IsEmpty())
	})

	t.Run("new repository permission", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{},
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 0}},
		)
		require.Len(t, diff.Repository, 1)
		assert.Equal(t, uint8(0), diff.Repository[PermissionIssues])
	})

	t.Run("upgrade repository permission", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 0}},
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 1}},
		)
		require.Len(t, diff.Repository, 1)
		assert.Equal(t, uint8(1), diff.Repository[PermissionIssues])
	})

	t.Run("no downgrade", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 1}},
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 0}},
		)
		assert.True(t, diff.IsEmpty())
	})

	t.Run("no change when same level", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 0}},
			PermissionSet{Repository: map[string]uint8{PermissionIssues: 0}},
		)
		assert.True(t, diff.IsEmpty())
	})

	t.Run("organization permissions", func(t *testing.T) {
		t.Parallel()

		diff := FindPermissionUpdates(
			PermissionSet{Organization: map[string]uint8{}},
			PermissionSet{Organization: map[string]uint8{PermissionAdministration: 0}},
		)
		require.Len(t, diff.Organization, 1)
		assert.Equal(t, uint8(0), diff.Organization[PermissionAdministration])
	})
}

func Test__CapabilityMapper__NewPermissionSet(t *testing.T) {
	t.Parallel()

	m := NewCapabilityMapper()

	t.Run("read-only capability", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.getIssue"})
		require.Contains(t, ps.Repository, PermissionIssues)
		assert.Equal(t, uint8(0), ps.Repository[PermissionIssues])
	})

	t.Run("write capability wins over read for same GitHub permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.getIssue", "github.createIssue"})
		require.Contains(t, ps.Repository, PermissionIssues)
		assert.Equal(t, uint8(1), ps.Repository[PermissionIssues])
	})

	t.Run("organization-scoped action", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.getWorkflowUsage"})
		require.Contains(t, ps.Organization, PermissionAdministration)
		assert.Equal(t, uint8(0), ps.Organization[PermissionAdministration])
		assert.NotContains(t, ps.Repository, PermissionAdministration)
	})

	t.Run("unknown capability ignored", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.doesNotExist"})
		assert.True(t, ps.IsEmpty())
	})

	t.Run("deployment actions request deployments write permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.createDeployment", "github.createDeploymentStatus"})
		got := ps.ForAppManifest()
		assert.Equal(t, "write", got["deployments"])
	})

	t.Run("status trigger requests statuses read permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.onCommitStatus"})
		got := ps.ForAppManifest()
		assert.Equal(t, "read", got["statuses"])
	})

	t.Run("combined status action requests statuses read permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.getCombinedCommitStatus"})
		got := ps.ForAppManifest()
		assert.Equal(t, "read", got["statuses"])
	})

	t.Run("check run trigger requests checks read permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.onCheckRun", "github.listCheckRunsForRef"})
		got := ps.ForAppManifest()
		assert.Equal(t, "read", got["checks"])
	})

	t.Run("merge pull request action requests pull request write permission", func(t *testing.T) {
		t.Parallel()

		ps := m.NewPermissionSet([]string{"github.mergePullRequest"})
		got := ps.ForAppManifest()
		assert.Equal(t, "write", got["pull_requests"])
	})
}

func Test__PermissionSet__IsEmpty(t *testing.T) {
	t.Parallel()

	assert.True(t, (&PermissionSet{}).IsEmpty())
	assert.True(t, (&PermissionSet{
		Repository:   map[string]uint8{},
		Organization: map[string]uint8{},
	}).IsEmpty())

	assert.False(t, (&PermissionSet{
		Repository: map[string]uint8{PermissionIssues: 0},
	}).IsEmpty())

	assert.False(t, (&PermissionSet{
		Organization: map[string]uint8{PermissionAdministration: 0},
	}).IsEmpty())
}

func Test__PermissionSet__ForHuman(t *testing.T) {
	t.Parallel()

	ps := PermissionSet{
		Repository:   map[string]uint8{PermissionIssues: 0, PermissionContents: 1},
		Organization: map[string]uint8{PermissionAdministration: 1},
	}
	got := ps.ForHuman()

	permsByKey := map[string]string{}
	for _, p := range got {
		permsByKey[p.Scope+"|"+p.Name] = p.Access
	}

	assert.Equal(t, "Read", permsByKey[PermissionScopeRepository+"|"+PermissionIssues])
	assert.Equal(t, "Read & Write", permsByKey[PermissionScopeRepository+"|"+PermissionContents])
	assert.Equal(t, "Read & Write", permsByKey[PermissionScopeOrganization+"|"+PermissionAdministration])
}

func Test__PermissionSet__ForAppManifest(t *testing.T) {
	t.Parallel()

	ps := PermissionSet{
		Repository: map[string]uint8{
			PermissionIssues:         0,
			PermissionPullRequests:   1,
			PermissionContents:       0,
			PermissionActions:        1,
			PermissionChecks:         0,
			PermissionCommitStatuses: 0,
			PermissionMetadata:       1,
		},
		Organization: map[string]uint8{
			PermissionAdministration: 0,
		},
	}

	got := ps.ForAppManifest()

	assert.Equal(t, "read", got["issues"])
	assert.Equal(t, "write", got["pull_requests"])
	assert.Equal(t, "read", got["contents"])
	assert.Equal(t, "write", got["actions"])
	assert.Equal(t, "read", got["checks"])
	assert.Equal(t, "read", got["statuses"])
	assert.Equal(t, "write", got["metadata"])
	assert.Equal(t, "read", got["organization_administration"])
}
