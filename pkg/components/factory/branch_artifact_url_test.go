package factory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestBranchTreeURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		provider   string
		repository string
		branch     string
		want       string
	}{
		{
			name:       "empty provider uses GitHub",
			repository: "example/repo",
			branch:     "feature/refund-retry",
			want:       "https://github.com/example/repo/tree/feature/refund-retry",
		},
		{
			name:       "GitHub owner/repo",
			provider:   "github",
			repository: "example/repo",
			branch:     "feature/refund-retry",
			want:       "https://github.com/example/repo/tree/feature/refund-retry",
		},
		{
			name:       "Bitbucket workspace/repo",
			provider:   "bitbucket",
			repository: "acme/app",
			branch:     "feature/refund-retry",
			want:       "https://bitbucket.org/acme/app/src/feature/refund-retry",
		},
		{
			name:       "unknown provider keeps GitHub owner/repo",
			provider:   "gitlab",
			repository: "example/repo",
			branch:     "feature/refund-retry",
			want:       "https://github.com/example/repo/tree/feature/refund-retry",
		},
		{
			name:       "repository https URL",
			repository: "https://github.com/example/repo",
			branch:     "hotfix",
			want:       "https://github.com/example/repo/tree/hotfix",
		},
		{
			name:       "full bitbucket.org URL uses src",
			repository: "https://bitbucket.org/acme/app",
			branch:     "feature/refund-retry",
			want:       "https://bitbucket.org/acme/app/src/feature/refund-retry",
		},
		{
			name:       "full github.com URL keeps tree when the provider is Bitbucket",
			provider:   "bitbucket",
			repository: "https://github.com/example/repo",
			branch:     "hotfix",
			want:       "https://github.com/example/repo/tree/hotfix",
		},
		{
			name:       "GitHub Enterprise URL",
			repository: "https://git.example.com/acme/storefront/",
			branch:     "feat/#42-fix",
			want:       "https://git.example.com/acme/storefront/tree/feat/%2342-fix",
		},
		{
			name:       "strips query and fragment from a repository URL",
			repository: "https://git.example.com/acme/storefront?tab=readme#readme",
			branch:     "hotfix",
			want:       "https://git.example.com/acme/storefront/tree/hotfix",
		},
		{
			name:       "strips embedded credentials from a repository URL",
			repository: "https://oauth2:token@git.example.com/acme/storefront",
			branch:     "hotfix",
			want:       "https://git.example.com/acme/storefront/tree/hotfix",
		},
		{
			name:       "strips credentials from a mixed-case scheme",
			repository: "HTTPS://oauth2:token@git.example.com/acme/storefront",
			branch:     "hotfix",
			want:       "https://git.example.com/acme/storefront/tree/hotfix",
		},
		{
			name:       "blank repository",
			repository: "",
			branch:     "feature/foo",
			want:       "",
		},
		{
			name:       "blank branch",
			repository: "example/repo",
			branch:     "",
			want:       "",
		},
		{
			name:       "rejects extra path segments in owner/repo",
			repository: "group/sub/repo",
			branch:     "main",
			want:       "",
		},
		{
			name:       "rejects non-http schemes",
			repository: "javascript:alert(1)",
			branch:     "main",
			want:       "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, branchTreeURL(tc.provider, tc.repository, tc.branch))
		})
	}
}

func TestAddWorkOrderArtifact_Execute_WritesBitbucketBranchURL(t *testing.T) {
	component := &AddWorkOrderArtifact{}
	factoryCtx := &fakeFactoryContext{vcsProvider: "bitbucket"}
	stateCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"orderId":      "wo-1",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
			"repository":   "acme/app",
		},
		ExecutionState: stateCtx,
		Factory:        factoryCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://bitbucket.org/acme/app/src/feature/refund-retry", factoryCtx.addArtifactParams.Data["url"])
}

func TestBuildArtifactData_WritesBranchTreeURLFromRepository(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
		Repository:   "example/repo",
	})

	assert.Equal(t, "https://github.com/example/repo/tree/feature/refund-retry", data["url"])
	assert.Equal(t, "example/repo", data["repository"])
	assert.Equal(t, "feature/refund-retry", data["name"])
}

func TestBuildArtifactData_KeepsExplicitBranchURL(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
		Repository:   "example/repo",
		URL:          "https://git.example.com/acme/storefront/tree/feature/refund-retry",
	})

	assert.Equal(t, "https://git.example.com/acme/storefront/tree/feature/refund-retry", data["url"])
	assert.Equal(t, "example/repo", data["repository"])
}

func TestBuildArtifactData_WritesBranchTreeURLFromFreeFormRepo(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Data:         []ArtifactDataEntry{{Name: "repo", Value: "acme/storefront"}},
	})

	assert.Equal(t, "https://github.com/acme/storefront/tree/hotfix", data["url"])
}

func TestBuildArtifactData_StripsCredentialsFromStoredRepository(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Repository:   "https://oauth2:token@git.example.com/acme/storefront",
	})

	assert.Equal(t, "https://git.example.com/acme/storefront", data["repository"])
	assert.Equal(t, "https://git.example.com/acme/storefront/tree/hotfix", data["url"])
	assert.NotContains(t, fmt.Sprintf("%v", data), "token")
}

func TestBuildArtifactData_StripsCredentialsFromMixedCaseRepositoryScheme(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Repository:   "HTTPS://oauth2:token@git.example.com/acme/storefront",
	})

	assert.Equal(t, "https://git.example.com/acme/storefront", data["repository"])
	assert.Equal(t, "https://git.example.com/acme/storefront/tree/hotfix", data["url"])
	assert.NotContains(t, fmt.Sprintf("%v", data), "token")
}

func TestBuildArtifactData_StripsCredentialsFromStoredRepositoryWhenURLIsSet(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Repository:   "https://oauth2:token@git.example.com/acme/storefront",
		URL:          "https://git.example.com/acme/storefront/tree/hotfix",
	})

	assert.Equal(t, "https://git.example.com/acme/storefront", data["repository"])
	assert.NotContains(t, fmt.Sprintf("%v", data), "token")
}

func TestBuildArtifactData_DropsUnparseableRepositoryURLWithCredentials(t *testing.T) {
	data, err := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Repository:   "https://oauth2:token@",
	}, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch artifact requires a url or a repository")
	assert.NotContains(t, err.Error(), "token")
	assert.Nil(t, data)
}

func TestBuildArtifactData_StripsCredentialsFromFreeFormRepository(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "hotfix",
		Data: []ArtifactDataEntry{
			{Name: "repository", Value: "https://oauth2:token@git.example.com/acme/storefront"},
		},
	})

	assert.Equal(t, "https://git.example.com/acme/storefront", data["repository"])
	assert.NotContains(t, fmt.Sprintf("%v", data), "token")
}

func TestBuildArtifactData_RejectsBranchWithoutReachableURL(t *testing.T) {
	_, err := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
	}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch artifact requires a url or a repository")
}

func TestValidateBranchArtifactConfiguration_RejectsNameOnly(t *testing.T) {
	err := validateBranchArtifactConfiguration(AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch artifact requires a url or a repository")
}

func TestValidateBranchArtifactConfiguration_AcceptsURLOnly(t *testing.T) {
	err := validateBranchArtifactConfiguration(AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
		URL:          "https://github.com/example/repo/tree/feature/refund-retry",
	})
	require.NoError(t, err)
}

func TestValidateBranchArtifactConfiguration_AcceptsRepositoryExpression(t *testing.T) {
	err := validateBranchArtifactConfiguration(AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "{{ previous().result.branch }}",
		Repository:   "{{ install_params.appRepository }}",
	})
	require.NoError(t, err)
}

func TestBuildArtifactData_AcceptsExplicitBranchURLWithoutRepository(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
		URL:          "https://github.com/example/repo/tree/feature/refund-retry",
	})

	assert.Equal(t, "https://github.com/example/repo/tree/feature/refund-retry", data["url"])
	assert.Equal(t, "feature/refund-retry", data["name"])
}
