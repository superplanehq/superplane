package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataSetInstallRequests(t *testing.T) {
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

func TestMetadataCurrentInstallRequestsReadsLegacyFields(t *testing.T) {
	requests := (Metadata{
		InstallRequested:        true,
		InstallRequestedAccount: "acme",
	}).CurrentInstallRequests()

	assert.Equal(t, []InstallRequest{{AccountLogin: "acme"}}, requests)
}
