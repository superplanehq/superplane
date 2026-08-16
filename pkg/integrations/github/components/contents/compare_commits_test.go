package contents

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	githubmocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__CompareCommits__Setup(t *testing.T) {
	component := CompareCommits{}

	for _, field := range []string{"repository", "base", "head"} {
		t.Run("requires "+field, func(t *testing.T) {
			configuration := map[string]any{"repository": "testhq/hello", "base": "main", "head": "feature"}
			delete(configuration, field)

			err := component.Setup(core.SetupContext{
				Configuration: configuration,
				Metadata:      &contexts.MetadataContext{},
				Integration:   githubmocks.IntegrationContextForNewSetupFlow(),
				HTTP:          &contexts.HTTPContext{},
			})

			require.ErrorContains(t, err, field+" is required")
		})
	}

	t.Run("stores repository metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			githubmocks.GitHubResponse(http.StatusOK, `{
				"id": 42,
				"full_name": "testhq/hello",
				"html_url": "https://github.com/testhq/hello"
			}`),
		}}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"repository": "testhq/hello", "base": "main", "head": "feature"},
			Metadata:      metadata,
			Integration:   githubmocks.IntegrationContextForNewSetupFlow(),
			HTTP:          httpContext,
		})

		require.NoError(t, err)
		assert.Equal(t, common.NodeMetadata{Repository: &common.Repository{
			ID: 42, Name: "testhq/hello", URL: "https://github.com/testhq/hello",
		}}, metadata.Get())
	})

	t.Run("rejects an invalid literal path pattern", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"repository": "testhq/hello", "base": "main", "head": "feature", "paths": []string{"[invalid"},
		}})
		require.ErrorContains(t, err, `invalid path pattern "[invalid"`)
	})

	t.Run("allows a path expression until execution", func(t *testing.T) {
		metadata := &contexts.MetadataContext{Metadata: common.NodeMetadata{Repository: &common.Repository{Name: "testhq/hello"}}}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"repository": "testhq/hello", "base": "main", "head": "feature", "paths": []string{`{{$.paths}}`}},
			Metadata:      metadata,
		})
		require.NoError(t, err)
	})

	t.Run("rejects an unsupported status filter", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"repository": "testhq/hello", "base": "main", "head": "feature", "statuses": []string{"deleted"},
		}})
		require.ErrorContains(t, err, `unsupported status filter "deleted"`)
	})

	t.Run("reports missing fields in configuration order", func(t *testing.T) {
		for _, testCase := range []struct {
			configuration map[string]any
			expectedError string
		}{
			{configuration: map[string]any{"repository": " ", "base": " ", "head": " "}, expectedError: "repository is required"},
			{configuration: map[string]any{"repository": "testhq/hello", "base": " ", "head": " "}, expectedError: "base is required"},
		} {
			err := component.Setup(core.SetupContext{Configuration: testCase.configuration})
			require.EqualError(t, err, testCase.expectedError)
		}
	})
}

func Test__CompareCommits__Execute__routesEmptyComparison(t *testing.T) {
	state := executeComparison(t, map[string]any{"repository": "testhq/hello", "base": "main", "head": "main"}, comparisonResponse(`[]`))
	assert.Equal(t, "unmatched", state.Channel)
	payload := comparisonPayload(t, state)
	assert.Empty(t, payload.Files)
	assert.Equal(t, "head-sha", payload.Comparison.HeadSHA)
}

func Test__CompareCommits__Execute__normalizesStatuses(t *testing.T) {
	state := executeComparison(t, baseConfiguration(), comparisonResponse(`[
		{"filename":"added","status":"added"},
		{"filename":"changed","status":"changed"},
		{"filename":"copied","previous_filename":"source","status":"copied"},
		{"filename":"modified","status":"modified"},
		{"filename":"removed","status":"removed"},
		{"filename":"renamed","previous_filename":"old","status":"renamed"},
		{"filename":"unchanged","status":"unchanged"}
	]`))

	assert.Equal(t, []ComparedFile{
		{Path: "added", Status: "added"},
		{Path: "changed", Status: "modified"},
		{Path: "copied", Status: "added"},
		{Path: "modified", Status: "modified"},
		{Path: "removed", Status: "removed"},
		{Path: "renamed", Status: "renamed", PreviousPath: "old"},
	}, comparisonPayload(t, state).Files)
	assert.Equal(t, 6, comparisonPayload(t, state).Comparison.ChangedFileCount)
}

func Test__CompareCommits__Execute__omitsUnchangedFilesFromChangedCount(t *testing.T) {
	state := executeComparison(t, baseConfiguration(), comparisonResponse(`[
		{"filename":"unchanged","status":"unchanged"}
	]`))

	assert.Equal(t, "unmatched", state.Channel)
	payload := comparisonPayload(t, state)
	assert.Zero(t, payload.Comparison.ChangedFileCount)
	assert.Empty(t, payload.Files)
}

func Test__CompareCommits__Execute__failsForUnsupportedStatus(t *testing.T) {
	state, err := runComparison(baseConfiguration(), comparisonResponse(`[{"filename":"file","status":"mystery"}]`))
	require.ErrorContains(t, err, `unsupported GitHub file status "mystery"`)
	assert.False(t, state.Finished)
}

func Test__CompareCommits__Execute__filtersFiles(t *testing.T) {
	files := comparisonResponse(`[
		{"filename":"docs/readme.md","status":"modified"},
		{"filename":"src/new.go","previous_filename":"legacy/new.go","status":"renamed"},
		{"filename":"src/copied.go","status":"copied"},
		{"filename":"src/skip.md","status":"added"}
	]`)

	t.Run("status after normalization", func(t *testing.T) {
		config := baseConfiguration()
		config["statuses"] = []string{"added", "added"}
		state := executeComparison(t, config, files)
		assert.Equal(t, "matched", state.Channel)
		assert.Equal(t, []ComparedFile{{Path: "src/copied.go", Status: "added"}, {Path: "src/skip.md", Status: "added"}}, comparisonPayload(t, state).Files)
		assert.Equal(t, &AppliedFilters{Statuses: stringSlicePointer([]string{"added"})}, comparisonPayload(t, state).AppliedFilters)
	})

	t.Run("path includes and exclusions", func(t *testing.T) {
		config := baseConfiguration()
		config["paths"] = []string{" src/** ", "!src/**/*.md"}
		state := executeComparison(t, config, files)
		assert.Equal(t, []ComparedFile{{Path: "src/copied.go", Status: "added"}, {Path: "src/new.go", Status: "renamed", PreviousPath: "legacy/new.go"}}, comparisonPayload(t, state).Files)
		assert.Equal(t, stringSlicePointer([]string{"src/**", "!src/**/*.md"}), comparisonPayload(t, state).AppliedFilters.Paths)
	})

	t.Run("rename matches previous path", func(t *testing.T) {
		config := baseConfiguration()
		config["paths"] = []string{"legacy/**"}
		state := executeComparison(t, config, files)
		assert.Equal(t, []ComparedFile{{Path: "src/new.go", Status: "renamed", PreviousPath: "legacy/new.go"}}, comparisonPayload(t, state).Files)
	})

	t.Run("rename into included path matches when previous path is excluded", func(t *testing.T) {
		config := baseConfiguration()
		config["paths"] = []string{"src/**", "!legacy/**"}
		state := executeComparison(t, config, files)
		assert.Equal(t, []ComparedFile{
			{Path: "src/copied.go", Status: "added"},
			{Path: "src/new.go", Status: "renamed", PreviousPath: "legacy/new.go"},
			{Path: "src/skip.md", Status: "added"},
		}, comparisonPayload(t, state).Files)
	})

	t.Run("exclusion-only patterns use an implicit include", func(t *testing.T) {
		config := baseConfiguration()
		config["paths"] = []string{"!docs/**"}
		state := executeComparison(t, config, files)
		assert.Len(t, comparisonPayload(t, state).Files, 3)
		assert.NotContains(t, comparisonPayload(t, state).Files, ComparedFile{Path: "docs/readme.md", Status: "modified"})
	})

	t.Run("status and path use AND logic", func(t *testing.T) {
		config := baseConfiguration()
		config["statuses"] = []string{"renamed"}
		config["paths"] = []string{"docs/**"}
		state := executeComparison(t, config, files)
		assert.Equal(t, "unmatched", state.Channel)
		assert.Empty(t, comparisonPayload(t, state).Files)
	})

	t.Run("empty enabled filters match every changed file", func(t *testing.T) {
		config := baseConfiguration()
		config["statuses"] = []string{}
		config["paths"] = []string{" ", ""}
		state := executeComparison(t, config, files)
		assert.Equal(t, "matched", state.Channel)
		assert.Len(t, comparisonPayload(t, state).Files, 4)
		assert.Empty(t, *comparisonPayload(t, state).AppliedFilters.Statuses)
		assert.Empty(t, *comparisonPayload(t, state).AppliedFilters.Paths)
	})
}

func Test__CompareCommits__Execute__failsForInvalidGlob(t *testing.T) {
	config := baseConfiguration()
	config["paths"] = []string{"src/**", "[invalid"}
	state, err := runComparison(config, comparisonResponse(`[]`))
	require.ErrorContains(t, err, `invalid path pattern "[invalid"`)
	assert.False(t, state.Finished)
}

func Test__CompareCommits__Execute__preservesGitHubError(t *testing.T) {
	state, err := runComparison(baseConfiguration(), `{"message":"No common ancestor"}`, http.StatusNotFound)
	require.ErrorContains(t, err, "No common ancestor")
	assert.False(t, state.Finished)
}

func Test__CompareCommits__Execute__rejectsThreeHundredFiles(t *testing.T) {
	files := make([]string, 300)
	for index := range files {
		files[index] = fmt.Sprintf(`{"filename":"file-%03d","status":"modified"}`, index)
	}
	state, err := runComparison(baseConfiguration(), comparisonResponse("["+strings.Join(files, ",")+"]"))
	require.ErrorContains(t, err, "GitHub returned 300 files")
	assert.False(t, state.Finished)
}

func Test__CompareCommits__OutputChannels(t *testing.T) {
	component := CompareCommits{}
	assert.Equal(t, []string{"matched", "unmatched"}, channelNames(component.OutputChannels(baseConfiguration())))
	config := baseConfiguration()
	config["statuses"] = []string{}
	assert.Equal(t, []string{"matched", "unmatched"}, channelNames(component.OutputChannels(config)))
	assert.Equal(t, []string{"matched", "unmatched"}, channelNames(component.OutputChannels("invalid")))
}

func Test__CompareCommits__Execute__emitsMatchedComparison(t *testing.T) {
	component := CompareCommits{}
	executionState := &contexts.ExecutionStateContext{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		githubmocks.GitHubResponse(http.StatusOK, comparisonResponse(`[
			{"filename":"z.go","status":"modified","additions":2,"deletions":1,"changes":3},
			{"filename":"a.go","status":"added","additions":4,"deletions":0,"changes":4}
		]`)),
		githubmocks.GitHubResponse(http.StatusOK, `{"sha":"head-sha"}`),
	}}

	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"repository": "testhq/hello", "base": "main", "head": "feature"},
		Integration:    githubmocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpContext,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.Equal(t, "matched", executionState.Channel)
	assert.Equal(t, "github.commitComparison", executionState.Type)
	require.Len(t, executionState.Payloads, 1)
	payload := executionState.Payloads[0].(map[string]any)["data"].(CommitComparisonPayload)
	assert.Equal(t, ComparisonMetadata{
		Repository:       "testhq/hello",
		BaseSHA:          "base-sha",
		HeadSHA:          "head-sha",
		MergeBaseSHA:     "merge-base-sha",
		URL:              "https://github.com/testhq/hello/compare/main...feature",
		ChangedFileCount: 2,
	}, payload.Comparison)
	assert.Equal(t, []ComparedFile{
		{Path: "a.go", Status: "added", Additions: 4, Changes: 4},
		{Path: "z.go", Status: "modified", Additions: 2, Deletions: 1, Changes: 3},
	}, payload.Files)
	assert.Equal(t, "/repos/testhq/hello/compare/main...feature", httpContext.Requests[0].URL.Path)
}

func Test__CompareCommits__Execute__normalizesRequiredConfiguration(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		githubmocks.GitHubResponse(http.StatusOK, comparisonResponse(`[]`)),
		githubmocks.GitHubResponse(http.StatusOK, `{"sha":"head-sha"}`),
	}}
	executionState := &contexts.ExecutionStateContext{}

	err := (&CompareCommits{}).Execute(core.ExecutionContext{
		Configuration:  map[string]any{"repository": " hello ", "base": " main ", "head": " feature "},
		Integration:    githubmocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpContext,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.Equal(t, "testhq/hello", comparisonPayload(t, executionState).Comparison.Repository)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, "/repos/testhq/hello/compare/main...feature", httpContext.Requests[0].URL.Path)
	assert.Equal(t, "/repos/testhq/hello/commits/feature", httpContext.Requests[1].URL.Path)
}

func Test__CompareCommits__Execute__resolvesConfiguredHead(t *testing.T) {
	firstPage := githubmocks.GitHubResponse(http.StatusOK, `{
		"html_url":"https://github.com/testhq/hello/compare/main...feature",
		"base_commit":{"sha":"base-sha"},
		"merge_base_commit":{"sha":"merge-base-sha"},
		"commits":[{"sha":"comparison-head"}],
		"files":[{"filename":"file.go","status":"modified"}]
	}`)
	firstPage.Header.Set("Link", `<https://api.github.com/repos/testhq/hello/compare/main...feature?page=2>; rel="next"`)
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		firstPage,
		githubmocks.GitHubResponse(http.StatusOK, `{"sha":"resolved-head"}`),
	}}
	executionState := &contexts.ExecutionStateContext{}

	err := (&CompareCommits{}).Execute(core.ExecutionContext{
		Configuration:  baseConfiguration(),
		Integration:    githubmocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpContext,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	payload := comparisonPayload(t, executionState)
	assert.Equal(t, "resolved-head", payload.Comparison.HeadSHA)
	assert.Equal(t, []ComparedFile{{Path: "file.go", Status: "modified"}}, payload.Files)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, "/repos/testhq/hello/commits/feature", httpContext.Requests[1].URL.Path)
}

func comparisonResponse(files string) string {
	return `{
		"html_url":"https://github.com/testhq/hello/compare/main...feature",
		"base_commit":{"sha":"base-sha"},
		"merge_base_commit":{"sha":"merge-base-sha"},
		"commits":[{"sha":"head-sha"}],
		"files":` + files + `
	}`
}

func baseConfiguration() map[string]any {
	return map[string]any{"repository": "testhq/hello", "base": "main", "head": "feature"}
}

func executeComparison(t *testing.T, configuration map[string]any, body string) *contexts.ExecutionStateContext {
	t.Helper()
	state, err := runComparison(configuration, body)
	require.NoError(t, err)
	return state
}

func runComparison(configuration map[string]any, body string, statusCodes ...int) (*contexts.ExecutionStateContext, error) {
	statusCode := http.StatusOK
	if len(statusCodes) > 0 {
		statusCode = statusCodes[0]
	}
	state := &contexts.ExecutionStateContext{}
	err := (&CompareCommits{}).Execute(core.ExecutionContext{
		Configuration: configuration,
		Integration:   githubmocks.IntegrationContextForNewSetupFlow(),
		HTTP: &contexts.HTTPContext{Responses: []*http.Response{
			githubmocks.GitHubResponse(statusCode, body),
			githubmocks.GitHubResponse(http.StatusOK, `{"sha":"head-sha"}`),
		}},
		ExecutionState: state,
	})
	return state, err
}

func comparisonPayload(t *testing.T, state *contexts.ExecutionStateContext) CommitComparisonPayload {
	t.Helper()
	require.Len(t, state.Payloads, 1)
	return state.Payloads[0].(map[string]any)["data"].(CommitComparisonPayload)
}

func stringSlicePointer(values []string) *[]string { return &values }

func channelNames(channels []core.OutputChannel) []string {
	names := make([]string, len(channels))
	for index, channel := range channels {
		names[index] = channel.Name
	}
	return names
}
