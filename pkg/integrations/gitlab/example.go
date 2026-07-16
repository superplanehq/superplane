package gitlab

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

//go:embed example_data_on_merge_comment.json
var exampleDataOnMergeCommentBytes []byte

//go:embed example_data_on_merge_request.json
var exampleDataOnMergeRequestBytes []byte

//go:embed example_data_on_milestone.json
var exampleDataOnMilestoneBytes []byte

//go:embed example_data_on_pipeline.json
var exampleDataOnPipelineBytes []byte

//go:embed example_data_on_release.json
var exampleDataOnReleaseBytes []byte

//go:embed example_data_on_tag.json
var exampleDataOnTagBytes []byte

//go:embed example_data_on_vulnerability.json
var exampleDataOnVulnerabilityBytes []byte
var exampleDataOnIssue = utils.NewEmbeddedJSON(exampleDataOnIssueBytes)
var exampleDataOnMergeComment = utils.NewEmbeddedJSON(exampleDataOnMergeCommentBytes)
var exampleDataOnMergeRequest = utils.NewEmbeddedJSON(exampleDataOnMergeRequestBytes)
var exampleDataOnMilestone = utils.NewEmbeddedJSON(exampleDataOnMilestoneBytes)
var exampleDataOnPipeline = utils.NewEmbeddedJSON(exampleDataOnPipelineBytes)
var exampleDataOnRelease = utils.NewEmbeddedJSON(exampleDataOnReleaseBytes)
var exampleDataOnTag = utils.NewEmbeddedJSON(exampleDataOnTagBytes)
var exampleDataOnVulnerability = utils.NewEmbeddedJSON(exampleDataOnVulnerabilityBytes)

func (i *OnIssue) ExampleData() map[string]any {
	return exampleDataOnIssue.Value()
}

func (m *OnMergeComment) ExampleData() map[string]any {
	return exampleDataOnMergeComment.Value()
}

func (m *OnMergeRequest) ExampleData() map[string]any {
	return exampleDataOnMergeRequest.Value()
}

func (m *OnMilestone) ExampleData() map[string]any {
	return exampleDataOnMilestone.Value()
}

func (p *OnPipeline) ExampleData() map[string]any {
	return exampleDataOnPipeline.Value()
}

func (r *OnRelease) ExampleData() map[string]any {
	return exampleDataOnRelease.Value()
}

func (t *OnTag) ExampleData() map[string]any {
	return exampleDataOnTag.Value()
}

func (v *OnVulnerability) ExampleData() map[string]any {
	return exampleDataOnVulnerability.Value()
}
