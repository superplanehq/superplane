package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestInstallationSettingsURL(t *testing.T) {
	stored := "https://github.com/organizations/acme/settings/installations/101"

	tests := []struct {
		name         string
		installation models.VCSProviderInstallation
		want         string
	}{
		{
			name: "uses the URL stored from GitHub",
			installation: models.VCSProviderInstallation{
				InstallationID: 101,
				AccountLogin:   "acme",
				AccountType:    "Organization",
				HTMLURL:        stored,
			},
			want: stored,
		},
		{
			name: "builds an organization installation URL",
			installation: models.VCSProviderInstallation{
				InstallationID: 101,
				AccountLogin:   "acme",
				AccountType:    "Organization",
			},
			want: stored,
		},
		{
			name: "builds a user installation URL",
			installation: models.VCSProviderInstallation{
				InstallationID: 55,
				AccountLogin:   "octocat",
				AccountType:    "User",
			},
			want: "https://github.com/settings/installations/55",
		},
		{
			name: "omits an organization URL without a login",
			installation: models.VCSProviderInstallation{
				InstallationID: 101,
				AccountType:    "Organization",
			},
		},
		{
			name: "omits a URL when the account type is unknown",
			installation: models.VCSProviderInstallation{
				InstallationID: 101,
				AccountLogin:   "acme",
				AccountType:    "Enterprise",
			},
		},
		{
			name: "omits a URL without an installation id",
			installation: models.VCSProviderInstallation{
				AccountLogin: "acme",
				AccountType:  "Organization",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, InstallationSettingsURL(test.installation))
		})
	}
}
