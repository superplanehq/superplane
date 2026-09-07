package contents

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/pathfilter"
)

const (
	commitComparisonPayloadType = "github.commitComparison"

	fileStatusAdded    = "added"
	fileStatusModified = "modified"
	fileStatusRemoved  = "removed"
	fileStatusRenamed  = "renamed"
)

var supportedFileStatuses = []string{
	fileStatusAdded,
	fileStatusModified,
	fileStatusRemoved,
	fileStatusRenamed,
}

type CompareCommits struct{}

type CompareCommitsConfiguration struct {
	Repository string    `json:"repository" mapstructure:"repository"`
	Base       string    `json:"base" mapstructure:"base"`
	Head       string    `json:"head" mapstructure:"head"`
	Statuses   *[]string `json:"statuses,omitempty" mapstructure:"statuses,omitempty"`
	Paths      *[]string `json:"paths,omitempty" mapstructure:"paths,omitempty"`
}

type CommitComparisonPayload struct {
	Comparison     ComparisonMetadata `json:"comparison"`
	AppliedFilters *AppliedFilters    `json:"appliedFilters,omitempty"`
	Files          []ComparedFile     `json:"files"`
}

type ComparisonMetadata struct {
	Repository       string `json:"repository"`
	BaseSHA          string `json:"baseSha"`
	HeadSHA          string `json:"headSha"`
	MergeBaseSHA     string `json:"mergeBaseSha"`
	URL              string `json:"url"`
	ChangedFileCount int    `json:"changedFileCount"`
}

type AppliedFilters struct {
	Statuses *[]string `json:"statuses,omitempty"`
	Paths    *[]string `json:"paths,omitempty"`
}

type ComparedFile struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	Changes      int    `json:"changes"`
	PreviousPath string `json:"previousPath,omitempty"`
}

func (c *CompareCommits) Name() string  { return "github.compareCommits" }
func (c *CompareCommits) Label() string { return "Compare Commits" }
func (c *CompareCommits) Description() string {
	return "Find files changed between two GitHub branches, tags, or commits"
}
func (c *CompareCommits) Icon() string  { return "github" }
func (c *CompareCommits) Color() string { return "gray" }

func (c *CompareCommits) Documentation() string {
	return `Compare Commits returns files that changed from the base ref to the head ref. Refs can be branch names, tag names, or commit SHAs. GitHub does not require the base ref to be an ancestor of the head ref.

GitHub statuses are normalized to added, modified, removed, and renamed. Copied files become added files, changed files become modified files, and unchanged files are omitted.

Status and path filters are optional. Status values use OR logic. Path patterns support ** and ! exclusions. Exclusions take precedence, and exclusion-only lists include ** implicitly. For renamed files, the component evaluates the current and previous paths separately. The file matches when either path passes all include and exclude patterns. Status and path filters use AND logic together.

The component always emits matched or unmatched. Without filters, all changed files match. The payload contains resolved comparison SHAs, the GitHub URL, the unfiltered file count, applied filters, and sorted file metadata.

GitHub returns the changed-file list on the first comparison page and limits the list to 300 files. GitHub paginates only the commit list. The component resolves the configured head separately and does not load the paginated commit list. The component fails when GitHub returns 300 files because it cannot determine whether the file list is complete.

Example filtered payload:

    {"comparison":{"repository":"owner/repo","baseSha":"abc","headSha":"def","mergeBaseSha":"aaa","url":"https://github.com/owner/repo/compare/main...feature","changedFileCount":1},"appliedFilters":{"statuses":["renamed"],"paths":["src/**"]},"files":[{"path":"src/new.ts","status":"renamed","additions":1,"deletions":1,"changes":2,"previousPath":"lib/old.ts"}]}`
}

func (c *CompareCommits) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "repository", Label: "Repository", Type: configuration.FieldTypeIntegrationResource, Required: true, TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "repository", UseNameAsValue: true}}},
		{Name: "base", Label: "Base", Type: configuration.FieldTypeString, Required: true, Description: "Branch name, tag name, or commit SHA to compare from"},
		{Name: "head", Label: "Head", Type: configuration.FieldTypeString, Required: true, Description: "Branch name, tag name, or commit SHA to compare to"},
		{Name: "statuses", Label: "Statuses", Type: configuration.FieldTypeMultiSelect, Togglable: true, Default: []string{fileStatusAdded}, TypeOptions: &configuration.TypeOptions{MultiSelect: &configuration.MultiSelectTypeOptions{Options: []configuration.FieldOption{{Label: "Added", Value: fileStatusAdded}, {Label: "Modified", Value: fileStatusModified}, {Label: "Removed", Value: fileStatusRemoved}, {Label: "Renamed", Value: fileStatusRenamed}}}}},
		{Name: "paths", Label: "Paths", Type: configuration.FieldTypeList, Togglable: true, Default: []string{"**"}, Description: "Path patterns. Use ** for directories and prefix exclusions with !.", TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{ItemLabel: "Pattern", ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeExpression}}}},
	}
}

func (c *CompareCommits) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{{Name: "matched", Description: "One or more files matched"}, {Name: "unmatched", Description: "No files matched"}}
}

func (c *CompareCommits) Setup(ctx core.SetupContext) error {
	config, err := decodeCompareCommitsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	config = normalizeRequiredCompareCommitsConfiguration(config)
	if err := validateRequiredCompareCommitsConfiguration(config); err != nil {
		return err
	}
	if config.Paths != nil {
		literalPaths := slices.DeleteFunc(normalizedPaths(config.Paths), common.IsExpression)
		if err := validatePathPatterns(literalPaths); err != nil {
			return err
		}
	}
	if err := validateStatusFilters(uniqueStrings(config.Statuses)); err != nil {
		return err
	}
	return common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration)
}

func (c *CompareCommits) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCompareCommitsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	config = normalizeRequiredCompareCommitsConfiguration(config)
	if err := validateRequiredCompareCommitsConfiguration(config); err != nil {
		return err
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}
	comparison, err := client.CompareCommits(context.Background(), config.Repository, config.Base, config.Head)
	if err != nil {
		return fmt.Errorf("failed to compare %q with %q: %w", config.Base, config.Head, err)
	}
	if len(comparison.Files) >= 300 {
		return fmt.Errorf("GitHub returned 300 files; narrow the comparison because the result can be incomplete")
	}

	files, err := normalizeComparedFiles(comparison.Files)
	if err != nil {
		return err
	}
	headCommit, err := client.GetCommit(context.Background(), config.Repository, config.Head)
	if err != nil {
		return fmt.Errorf("failed to resolve head %q: %w", config.Head, err)
	}
	payload := CommitComparisonPayload{Comparison: ComparisonMetadata{Repository: client.CanonicalRepository(config.Repository), BaseSHA: comparison.GetBaseCommit().GetSHA(), HeadSHA: headCommit.GetSHA(), MergeBaseSHA: comparison.GetMergeBaseCommit().GetSHA(), URL: comparison.GetHTMLURL(), ChangedFileCount: len(files)}, Files: files}

	filterMode := config.Statuses != nil || config.Paths != nil
	if filterMode {
		statuses := uniqueStrings(config.Statuses)
		paths := normalizedPaths(config.Paths)
		if err := validateStatusFilters(statuses); err != nil {
			return err
		}
		if err := validatePathPatterns(paths); err != nil {
			return err
		}
		files = filterComparedFiles(files, statuses, paths)
		payload.Files = files
		payload.AppliedFilters = &AppliedFilters{}
		if config.Statuses != nil {
			payload.AppliedFilters.Statuses = &statuses
		}
		if config.Paths != nil {
			payload.AppliedFilters.Paths = &paths
		}
	}

	return ctx.ExecutionState.Emit(compareCommitsOutputChannel(len(files)), commitComparisonPayloadType, []any{payload})
}

func normalizeComparedFiles(githubFiles []*github.CommitFile) ([]ComparedFile, error) {
	files := make([]ComparedFile, 0, len(githubFiles))
	for _, file := range githubFiles {
		status, include, err := normalizeFileStatus(file.GetStatus())
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		comparedFile := ComparedFile{Path: file.GetFilename(), Status: status, Additions: file.GetAdditions(), Deletions: file.GetDeletions(), Changes: file.GetChanges()}
		if status == fileStatusRenamed {
			comparedFile.PreviousPath = file.GetPreviousFilename()
		}
		files = append(files, comparedFile)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func compareCommitsOutputChannel(fileCount int) string {
	if fileCount > 0 {
		return "matched"
	}
	return "unmatched"
}

func decodeCompareCommitsConfiguration(raw any) (CompareCommitsConfiguration, error) {
	var config CompareCommitsConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return config, nil
}

func validateRequiredCompareCommitsConfiguration(config CompareCommitsConfiguration) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "repository", value: config.Repository},
		{name: "base", value: config.Base},
		{name: "head", value: config.Head},
	} {
		if field.value == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}

func normalizeRequiredCompareCommitsConfiguration(config CompareCommitsConfiguration) CompareCommitsConfiguration {
	config.Repository = strings.TrimSpace(config.Repository)
	config.Base = strings.TrimSpace(config.Base)
	config.Head = strings.TrimSpace(config.Head)
	return config
}

func normalizeFileStatus(status string) (string, bool, error) {
	switch status {
	case fileStatusAdded, fileStatusModified, fileStatusRemoved, fileStatusRenamed:
		return status, true, nil
	case "copied":
		return fileStatusAdded, true, nil
	case "changed":
		return fileStatusModified, true, nil
	case "unchanged":
		return "", false, nil
	default:
		return "", false, fmt.Errorf("unsupported GitHub file status %q", status)
	}
}

func uniqueStrings(values *[]string) []string {
	if values == nil {
		return nil
	}
	unique := make([]string, 0, len(*values))
	for _, value := range *values {
		if !slices.Contains(unique, value) {
			unique = append(unique, value)
		}
	}
	return unique
}

func normalizedPaths(values *[]string) []string {
	if values == nil {
		return nil
	}
	return pathfilter.TrimNonEmptyStrings(*values)
}

func validatePathPatterns(patterns []string) error {
	for _, pattern := range patterns {
		glob := strings.TrimSpace(strings.TrimPrefix(pattern, "!"))
		if glob == "" || !doublestar.ValidatePattern(glob) {
			return fmt.Errorf("invalid path pattern %q", pattern)
		}
	}
	return nil
}

func validateStatusFilters(statuses []string) error {
	for _, status := range statuses {
		if !slices.Contains(supportedFileStatuses, status) {
			return fmt.Errorf("unsupported status filter %q", status)
		}
	}
	return nil
}

func filterComparedFiles(files []ComparedFile, statuses, patterns []string) []ComparedFile {
	matched := make([]ComparedFile, 0, len(files))
	for _, file := range files {
		if len(statuses) > 0 && !slices.Contains(statuses, file.Status) {
			continue
		}
		if len(patterns) > 0 && !comparedFileMatchesPaths(file, patterns) {
			continue
		}
		matched = append(matched, file)
	}
	return matched
}

func comparedFileMatchesPaths(file ComparedFile, patterns []string) bool {
	paths := []string{file.Path}
	if file.PreviousPath != "" {
		paths = append(paths, file.PreviousPath)
	}

	return slices.ContainsFunc(paths, func(path string) bool {
		return pathfilter.EvaluatePushPathGlobFilter(patterns, []string{path}, nil, nil, nil)
	})
}

func (c *CompareCommits) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *CompareCommits) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *CompareCommits) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *CompareCommits) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *CompareCommits) Hooks() []core.Hook                          { return nil }
func (c *CompareCommits) HandleHook(ctx core.ActionHookContext) error { return nil }
