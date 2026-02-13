package gitlab

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

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

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

var exampleDataOnMergeRequestOnce sync.Once
var exampleDataOnMergeRequest map[string]any

var exampleDataOnMilestoneOnce sync.Once
var exampleDataOnMilestone map[string]any

var exampleDataOnPipelineOnce sync.Once
var exampleDataOnPipeline map[string]any

var exampleDataOnReleaseOnce sync.Once
var exampleDataOnRelease map[string]any

var exampleDataOnTagOnce sync.Once
var exampleDataOnTag map[string]any

var exampleDataOnVulnerabilityOnce sync.Once
var exampleDataOnVulnerability map[string]any

func (i *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

func (m *OnMergeRequest) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMergeRequestOnce, exampleDataOnMergeRequestBytes, &exampleDataOnMergeRequest)
}

func (m *OnMilestone) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMilestoneOnce, exampleDataOnMilestoneBytes, &exampleDataOnMilestone)
}

func (p *OnPipeline) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPipelineOnce, exampleDataOnPipelineBytes, &exampleDataOnPipeline)
}

func (r *OnRelease) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnReleaseOnce, exampleDataOnReleaseBytes, &exampleDataOnRelease)
}

func (t *OnTag) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTagOnce, exampleDataOnTagBytes, &exampleDataOnTag)
}

func (v *OnVulnerability) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnVulnerabilityOnce, exampleDataOnVulnerabilityBytes, &exampleDataOnVulnerability)
}
