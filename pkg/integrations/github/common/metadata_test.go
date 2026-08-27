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
