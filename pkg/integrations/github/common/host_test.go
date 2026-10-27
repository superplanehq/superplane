package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind HostKind
		wantHost string
		wantErr  string
	}{
		{name: "empty is github.com", input: "", wantKind: HostKindDotCom, wantHost: ""},
		{name: "github.com hostname", input: "github.com", wantKind: HostKindDotCom, wantHost: ""},
		{name: "www.github.com", input: "www.github.com", wantKind: HostKindDotCom, wantHost: ""},
		{name: "https github.com", input: "https://github.com", wantKind: HostKindDotCom, wantHost: ""},
		{name: "https github.com with slash", input: "https://github.com/", wantKind: HostKindDotCom, wantHost: ""},
		{name: "ghe.com tenant hostname", input: "acme.ghe.com", wantKind: HostKindGHECom, wantHost: "acme.ghe.com"},
		{name: "ghe.com tenant URL", input: "https://acme.ghe.com/", wantKind: HostKindGHECom, wantHost: "acme.ghe.com"},
		{name: "ghes hostname", input: "ghes.corp", wantKind: HostKindGHES, wantHost: "ghes.corp"},
		{name: "ghes URL", input: "https://ghes.corp", wantKind: HostKindGHES, wantHost: "ghes.corp"},
		{name: "rejects http", input: "http://acme.ghe.com", wantErr: "must use https"},
		{name: "rejects path", input: "https://acme.ghe.com/orgs/acme", wantErr: "must not include a path"},
		{name: "rejects query", input: "https://acme.ghe.com/?x=1", wantErr: "must not include a query"},
		{name: "rejects userinfo", input: "https://user:pass@acme.ghe.com", wantErr: "must not include userinfo"},
		{name: "rejects ghe.com apex", input: "ghe.com", wantErr: "not a tenant hostname"},
		{name: "rejects slash in hostname", input: "acme.ghe.com/org", wantErr: "must be a hostname"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host, err := ParseHost(test.input)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantKind, host.Kind)
			assert.Equal(t, test.wantHost, host.Hostname)
		})
	}
}

func TestHostURLs(t *testing.T) {
	tests := []struct {
		name     string
		host     Host
		web      string
		api      string
		graphql  string
		upload   string
		git      string
		clone    string
		repo     string
		create   string
		install  string
		manifest string
		perms    string
		settings string
	}{
		{
			name:     "empty host is github.com",
			host:     Host{},
			web:      "https://github.com",
			api:      "https://api.github.com/",
			graphql:  "https://api.github.com/graphql",
			upload:   "https://uploads.github.com/",
			git:      "github.com",
			clone:    "https://github.com/acme/app.git",
			repo:     "https://github.com/acme/app",
			create:   "https://github.com/organizations/acme/settings/apps/new",
			install:  "https://github.com/apps/superplane/installations/new?state=abc%2Fdef",
			manifest: "https://api.github.com/app-manifests/code1/conversions",
			perms:    "https://github.com/organizations/acme/settings/apps/superplane/permissions",
			settings: "https://github.com/organizations/acme/settings/installations/101",
		},
		{
			name:     "dotcom helper",
			host:     DotCom(),
			web:      "https://github.com",
			api:      "https://api.github.com/",
			graphql:  "https://api.github.com/graphql",
			upload:   "https://uploads.github.com/",
			git:      "github.com",
			clone:    "https://github.com/acme/app.git",
			repo:     "https://github.com/acme/app",
			create:   "https://github.com/organizations/acme/settings/apps/new",
			install:  "https://github.com/apps/superplane/installations/new?state=abc%2Fdef",
			manifest: "https://api.github.com/app-manifests/code1/conversions",
			perms:    "https://github.com/organizations/acme/settings/apps/superplane/permissions",
			settings: "https://github.com/organizations/acme/settings/installations/101",
		},
		{
			name:     "ghe.com tenant",
			host:     Host{Kind: HostKindGHECom, Hostname: "acme.ghe.com"},
			web:      "https://acme.ghe.com",
			api:      "https://api.acme.ghe.com/",
			graphql:  "https://api.acme.ghe.com/graphql",
			upload:   "https://uploads.acme.ghe.com/",
			git:      "acme.ghe.com",
			clone:    "https://acme.ghe.com/acme/app.git",
			repo:     "https://acme.ghe.com/acme/app",
			create:   "https://acme.ghe.com/organizations/acme/settings/apps/new",
			install:  "https://acme.ghe.com/apps/superplane/installations/new?state=abc%2Fdef",
			manifest: "https://api.acme.ghe.com/app-manifests/code1/conversions",
			perms:    "https://acme.ghe.com/organizations/acme/settings/apps/superplane/permissions",
			settings: "https://acme.ghe.com/organizations/acme/settings/installations/101",
		},
		{
			name:     "ghes",
			host:     Host{Kind: HostKindGHES, Hostname: "ghes.corp"},
			web:      "https://ghes.corp",
			api:      "https://ghes.corp/api/v3/",
			graphql:  "https://ghes.corp/api/graphql",
			upload:   "https://ghes.corp/api/uploads/",
			git:      "ghes.corp",
			clone:    "https://ghes.corp/acme/app.git",
			repo:     "https://ghes.corp/acme/app",
			create:   "https://ghes.corp/organizations/acme/settings/apps/new",
			install:  "https://ghes.corp/apps/superplane/installations/new?state=abc%2Fdef",
			manifest: "https://ghes.corp/api/v3/app-manifests/code1/conversions",
			perms:    "https://ghes.corp/organizations/acme/settings/apps/superplane/permissions",
			settings: "https://ghes.corp/organizations/acme/settings/installations/101",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.web, test.host.WebURL())
			assert.Equal(t, test.api, test.host.APIURL())
			assert.Equal(t, test.graphql, test.host.GraphQLURL())
			assert.Equal(t, test.upload, test.host.UploadURL())
			assert.Equal(t, test.git, test.host.GitHost())
			assert.Equal(t, test.clone, test.host.CloneURL("acme/app"))
			assert.Equal(t, test.repo, test.host.RepoURL("acme/app"))
			assert.Equal(t, test.create, test.host.CreateAppURL(OwnerTypeOrganization, "acme"))
			assert.Equal(t, test.install, test.host.InstallAppURL("superplane", "abc/def"))
			assert.Equal(t, test.manifest, test.host.ManifestConversionURL("code1"))
			assert.Equal(t, test.perms, test.host.AppPermissionsURL(OwnerTypeOrganization, "acme", "superplane"))
			assert.Equal(t, test.settings, test.host.InstallationSettingsURL(OwnerTypeOrganization, "acme", "101"))
			assert.Equal(t, test.host.WebURL()+"/settings/apps/new", test.host.CreateAppURL(OwnerTypeUser, "octocat"))
			assert.Equal(t, test.host.WebURL()+"/settings/apps/superplane/permissions", test.host.AppPermissionsURL(OwnerTypeUser, "octocat", "superplane"))
			assert.Equal(t, test.host.WebURL()+"/settings/installations/55", test.host.InstallationSettingsURL(OwnerTypeUser, "octocat", "55"))
		})
	}
}

func TestHostEmptyPreservesDotComSecretsAndAllowlist(t *testing.T) {
	host := Host{}
	assert.Contains(t, host.SecretUsage(), "https://github.com/<owner>/<repo>.git")
	assert.Contains(t, host.SetupGitScript(), "gh auth setup-git --hostname github.com --force")
	assert.Equal(t, []string{
		"github.com",
		"api.github.com",
		"codeload.github.com",
		"*.githubusercontent.com",
		"objects.githubusercontent.com",
	}, host.AllowedHosts())
	assert.Equal(t, "https://github.com", host.ServerURLEnv())
	assert.Equal(t, "https://api.github.com", host.APIURLEnv())
	assert.Equal(t, "https://api.github.com/graphql", host.GraphQLURLEnv())
}

func TestHostGHEComSecretsAndAllowlist(t *testing.T) {
	host := Host{Kind: HostKindGHECom, Hostname: "acme.ghe.com"}
	assert.Contains(t, host.SecretUsage(), "GH_HOST")
	assert.Contains(t, host.SecretUsage(), "https://acme.ghe.com/<owner>/<repo>.git")
	assert.Contains(t, host.SetupGitScript(), "gh auth setup-git --hostname acme.ghe.com --force")
	assert.Equal(t, []string{"acme.ghe.com", "*.acme.ghe.com"}, host.AllowedHosts())
	assert.NotContains(t, host.AllowedHosts(), "*.ghe.com")
}

func TestHostMatches(t *testing.T) {
	dotcom := DotCom()
	ghe := Host{Kind: HostKindGHECom, Hostname: "acme.ghe.com"}
	ghes := Host{Kind: HostKindGHES, Hostname: "ghes.corp"}

	assert.True(t, dotcom.Matches("github.com"))
	assert.True(t, dotcom.Matches("api.github.com"))
	assert.True(t, dotcom.Matches("uploads.github.com"))
	assert.True(t, dotcom.Matches("codeload.github.com"))
	assert.True(t, dotcom.Matches("private-user-images.githubusercontent.com"))
	assert.False(t, dotcom.Matches("acme.ghe.com"))
	assert.False(t, dotcom.IsWebHostname("api.github.com"))
	assert.True(t, dotcom.IsWebHostname("github.com"))

	assert.True(t, ghe.Matches("acme.ghe.com"))
	assert.True(t, ghe.Matches("api.acme.ghe.com"))
	assert.True(t, ghe.Matches("uploads.acme.ghe.com"))
	assert.True(t, ghe.Matches("objects.acme.ghe.com"))
	assert.False(t, ghe.Matches("other.ghe.com"))
	assert.False(t, ghe.Matches("github.com"))
	assert.True(t, ghe.IsWebHostname("acme.ghe.com"))
	assert.False(t, ghe.IsWebHostname("api.acme.ghe.com"))

	assert.True(t, ghes.Matches("ghes.corp"))
	assert.False(t, ghes.Matches("api.ghes.corp"))
	assert.False(t, ghes.Matches("github.com"))
}

func TestHostParseRepoURL(t *testing.T) {
	host := DotCom()
	owner, repo, ok := host.ParseRepoURL("https://github.com/acme/app")
	assert.True(t, ok)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "app", repo)

	owner, repo, ok = host.ParseRepoURL("https://github.com/acme/app.git")
	assert.True(t, ok)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "app", repo)

	_, _, ok = host.ParseRepoURL("https://gitlab.com/acme/app")
	assert.False(t, ok)

	ghe := Host{Kind: HostKindGHECom, Hostname: "acme.ghe.com"}
	owner, repo, ok = ghe.ParseRepoURL("https://acme.ghe.com/acme/app/issues/1")
	assert.True(t, ok)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "app", repo)
}

func TestIsKnownGitHubHostname(t *testing.T) {
	assert.True(t, IsKnownGitHubHostname("github.com"))
	assert.True(t, IsKnownGitHubHostname("api.github.com"))
	assert.True(t, IsKnownGitHubHostname("acme.ghe.com"))
	assert.True(t, IsKnownGitHubHostname("api.acme.ghe.com"))
	assert.False(t, IsKnownGitHubHostname("gitlab.com"))
	assert.False(t, IsKnownGitHubWebHostname("api.github.com"))
	assert.True(t, IsKnownGitHubWebHostname("github.com"))
	assert.True(t, IsKnownGitHubWebHostname("acme.ghe.com"))
}
