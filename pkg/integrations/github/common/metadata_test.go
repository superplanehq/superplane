package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Metadata_AllowsPendingInstallation(t *testing.T) {
	metadata := Metadata{
		PendingInstallations: []PendingInstallation{
			{ID: "11", AccountLogin: "acme"},
			{ID: "22", AccountLogin: "octo"},
		},
	}

	assert.True(t, metadata.AllowsPendingInstallation("11"))
	assert.True(t, metadata.AllowsPendingInstallation("22"))
	assert.False(t, metadata.AllowsPendingInstallation("99"))
	assert.False(t, metadata.AllowsPendingInstallation(""))
	assert.False(t, Metadata{}.AllowsPendingInstallation("11"))
}

func Test__Metadata_AllowsStartedBy(t *testing.T) {
	assert.True(t, Metadata{}.AllowsStartedBy("any"))
	assert.True(t, Metadata{StartedByUserID: "user-1"}.AllowsStartedBy("user-1"))
	assert.False(t, Metadata{StartedByUserID: "user-1"}.AllowsStartedBy("user-2"))
	assert.False(t, Metadata{StartedByUserID: "user-1"}.AllowsStartedBy(""))
}

func Test__Metadata_SetInstallRequests(t *testing.T) {
	metadata := Metadata{}
	metadata.SetInstallRequests([]InstallRequest{
		{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
		{ID: "1", AccountLogin: "acme", RequesterLogin: "member"},
		{ID: "1", AccountLogin: "acme", RequesterLogin: "member"},
	})

	assert.True(t, metadata.InstallRequested)
	assert.Empty(t, metadata.InstallRequestedAccount)
	assert.Len(t, metadata.InstallRequests, 2)
}

func Test__Metadata_CurrentInstallRequestsReadsLegacyFields(t *testing.T) {
	requests := (Metadata{
		InstallRequested:        true,
		InstallRequestedAccount: "acme",
		StartedByGitHubLogin:    "member",
	}).CurrentInstallRequests()

	assert.Equal(t, []InstallRequest{{AccountLogin: "acme", RequesterLogin: "member"}}, requests)
}

func Test__Metadata_SetPendingInstallationsDeduplicatesByID(t *testing.T) {
	metadata := Metadata{}
	metadata.SetPendingInstallations([]PendingInstallation{
		{ID: "11", AccountLogin: "acme"},
		{ID: "11", AccountLogin: "acme-copy"},
		{ID: "22", AccountLogin: "octo"},
	})

	assert.Equal(t, []PendingInstallation{
		{ID: "11", AccountLogin: "acme"},
		{ID: "22", AccountLogin: "octo"},
	}, metadata.PendingInstallations)
}
