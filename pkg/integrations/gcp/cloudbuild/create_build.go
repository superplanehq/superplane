package cloudbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	createBuildPayloadType         = "gcp.cloudbuild.build"
	createBuildPassedOutputChannel = "passed"
	createBuildFailedOutputChannel = "failed"
	createBuildPollAction          = "poll"
	createBuildPollInterval        = 5 * time.Minute
	createBuildExecutionKV         = "build_id"

	createBuildConnectedRevisionBranch = "branch"
	createBuildConnectedRevisionTag    = "tag"
	createBuildConnectedRevisionCommit = "commit"
)

type CreateBuild struct{}

type CreateBuildConfiguration struct {
	ProjectID              string   `json:"projectId" mapstructure:"projectId"`
	Source                 string   `json:"source" mapstructure:"source"`
	UseConnectedRepository bool     `json:"useConnectedRepository" mapstructure:"useConnectedRepository"`
	ConnectionLocation     string   `json:"connectionLocation" mapstructure:"connectionLocation"`
	Connection             string   `json:"connection" mapstructure:"connection"`
	ConnectedRepository    string   `json:"connectedRepository" mapstructure:"connectedRepository"`
	ConnectedRevisionType  string   `json:"connectedRevisionType" mapstructure:"connectedRevisionType"`
	ConnectedBranch        string   `json:"connectedBranch" mapstructure:"connectedBranch"`
	ConnectedTag           string   `json:"connectedTag" mapstructure:"connectedTag"`
	ConnectedCommitSHA     string   `json:"connectedCommitSha" mapstructure:"connectedCommitSha"`
	RepoName               string   `json:"repoName" mapstructure:"repoName"`
	BranchName             string   `json:"branchName" mapstructure:"branchName"`
	TagName                string   `json:"tagName" mapstructure:"tagName"`
	CommitSHA              string   `json:"commitSha" mapstructure:"commitSha"`
	Steps                  string   `json:"steps" mapstructure:"steps"`
	Images                 []string `json:"images" mapstructure:"images"`
	Substitutions          string   `json:"substitutions" mapstructure:"substitutions"`
	Timeout                string   `json:"timeout" mapstructure:"timeout"`
}

type CreateBuildNodeMetadata struct {
	SubscriptionID string `json:"subscriptionId,omitempty" mapstructure:"subscriptionId,omitempty"`
}

type CreateBuildExecutionMetadata struct {
	Build map[string]any `json:"build,omitempty" mapstructure:"build,omitempty"`
}

func (c *CreateBuild) Name() string {
	return "gcp.cloudbuild.createBuild"
}

func (c *CreateBuild) Label() string {
	return "Cloud Build • Create Build"
}

func (c *CreateBuild) Description() string {
	return "Create a Cloud Build build and wait for it to finish"
}

func (c *CreateBuild) Documentation() string {
	return `Creates and starts a Google Cloud Build build, then waits for the build to reach a terminal status.

## Configuration

- **Steps** (required): JSON array of build steps. Each step needs at minimum a ` + "`name`" + ` (builder image) and optional ` + "`args`" + `. Example: ` + "`[{\"name\":\"gcr.io/cloud-builders/docker\",\"args\":[\"build\",\"-t\",\"gcr.io/$PROJECT_ID/myapp\",\".\"]}]`" + `
- **Source**: Optional JSON object for the build source. This is the most flexible option and supports ` + "`gitSource`" + `, ` + "`repoSource`" + `, or ` + "`storageSource`" + `. Example: ` + "`{\"gitSource\":{\"url\":\"https://github.com/org/repo.git\",\"revision\":\"main\"}}`" + `
- **Connected Repository**: Optional Cloud Build 2nd-gen repository path. Select a location, connection, repository, and branch/tag/commit directly from GCP. SuperPlane sends ` + "`source.connectedRepository`" + ` and creates the build in the repository's region.
- **Repository / Branch / Tag / Commit SHA**: Convenience shortcut for repository-backed builds. If the repository value looks like a Git URL (` + "`https://...`" + `, ` + "`ssh://...`" + `, or ` + "`git@...`" + `), SuperPlane creates ` + "`source.gitSource`" + `. Otherwise it treats the value as a Cloud Source Repository name and creates ` + "`source.repoSource`" + `. Choose exactly one revision field.
- **Images**: Optional list of Docker image names to push after the build.
- **Substitutions**: JSON object of substitution key-value pairs (e.g. ` + "`{\"_ENV\":\"production\"}`" + `).
- **Timeout**: Build timeout (e.g. ` + "`600s`" + `). Defaults to Cloud Build default (10 minutes).
- **Project ID Override**: Optionally run the build in a different project than the connected integration.

## Output

The terminal Build resource, including ` + "`id`" + `, ` + "`status`" + `, ` + "`logUrl`" + `, ` + "`createTime`" + `, ` + "`finishTime`" + `, and more.

## Output Channels

- **Passed**: Emitted when Cloud Build finishes with ` + "`SUCCESS`" + `.
- **Failed**: Emitted when Cloud Build finishes with any other terminal status, including ` + "`FAILURE`" + `, ` + "`INTERNAL_ERROR`" + `, ` + "`TIMEOUT`" + `, ` + "`CANCELLED`" + `, or ` + "`EXPIRED`" + `.

## Notes

- SuperPlane listens for Cloud Build notifications through the connected GCP integration and falls back to polling if an event does not arrive.
- SuperPlane automatically creates the shared ` + "`cloud-builds`" + ` Pub/Sub topic and push subscription when the GCP integration has ` + "`roles/pubsub.admin`" + ` and both the **Cloud Build** and **Pub/Sub** APIs are enabled.
- Cancelling the running execution from the UI sends a Cloud Build cancel request for the active build.`
}

func (c *CreateBuild) Icon() string  { return "gcp" }
func (c *CreateBuild) Color() string { return "gray" }

func (c *CreateBuild) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  createBuildPassedOutputChannel,
			Label: "Passed",
		},
		{
			Name:  createBuildFailedOutputChannel,
			Label: "Failed",
		},
	}
}

func (c *CreateBuild) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "steps",
			Label:       "Steps",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: `JSON array of build steps. Each step needs a "name" (builder image) and optional "args".`,
			Placeholder: `[{"name":"gcr.io/cloud-builders/docker","args":["build","."]}]`,
		},
		{
			Name:        "source",
			Label:       "Source",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: `Optional JSON object for the build source. Supports gitSource, repoSource, connectedRepository, or storageSource. Use this for advanced source configuration.`,
			Placeholder: `{"gitSource":{"url":"https://github.com/org/repo.git","revision":"main"}}`,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"", "false"}},
			},
		},
		{
			Name:        "useConnectedRepository",
			Label:       "Use Connected Repository",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable this to pick a Cloud Build 2nd-gen connected repository, then leave Source and Repository URL / Cloud Source Repo empty.",
		},
		{
			Name:        "connectionLocation",
			Label:       "Cloud Build Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the Cloud Build region that contains the source connection.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeLocation,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "connection",
			Label:       "Connection",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the Cloud Build source connection in the chosen region.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connectionLocation", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connectionLocation", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeConnection,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "connectionLocation"}},
					},
				},
			},
		},
		{
			Name:        "connectedRepository",
			Label:       "Connected Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the connected source repository to build from.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connection", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connection", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRepository,
					Parameters: []configuration.ParameterRef{
						{Name: "connection", ValueFrom: &configuration.ParameterValueFrom{Field: "connection"}},
					},
				},
			},
		},
		{
			Name:        "connectedRevisionType",
			Label:       "Connected Repository Revision",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     createBuildConnectedRevisionBranch,
			Description: "Choose whether the connected repository build should use a branch, a tag, or a commit SHA.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connectedRepository", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connectedRepository", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Branch", Value: createBuildConnectedRevisionBranch},
						{Label: "Tag", Value: createBuildConnectedRevisionTag},
						{Label: "Commit SHA", Value: createBuildConnectedRevisionCommit},
					},
				},
			},
		},
		{
			Name:        "connectedBranch",
			Label:       "Connected Branch",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the branch to build from in the connected repository.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connectedRepository", Values: []string{"*"}},
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionBranch}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionBranch}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeBranch,
					Parameters: []configuration.ParameterRef{
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "connectedRepository"}},
					},
				},
			},
		},
		{
			Name:        "connectedTag",
			Label:       "Connected Tag",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select the tag to build from in the connected repository.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connectedRepository", Values: []string{"*"}},
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionTag}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionTag}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTag,
					Parameters: []configuration.ParameterRef{
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "connectedRepository"}},
					},
				},
			},
		},
		{
			Name:        "connectedCommitSha",
			Label:       "Connected Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Commit SHA to build from in the connected repository.",
			Placeholder: "e.g. 5d7363a99d19e45830e1bc9622d2e4fa72d7229f",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"true"}},
				{Field: "connectedRepository", Values: []string{"*"}},
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionCommit}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "connectedRevisionType", Values: []string{createBuildConnectedRevisionCommit}},
			},
		},
		{
			Name:        "repoName",
			Label:       "Repository URL / Cloud Source Repo",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Manual repository shortcut. Use a Git URL for gitSource builds, or a Cloud Source Repository name for repoSource builds. Leave empty when using the connected repository selector or Source JSON.",
			Placeholder: "e.g. https://github.com/org/repo.git",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"", "false"}},
			},
		},
		{
			Name:        "branchName",
			Label:       "Branch Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Branch to build from when Repository is used. Mutually exclusive with Tag Name and Commit SHA.",
			Placeholder: "e.g. main",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"", "false"}},
			},
		},
		{
			Name:        "tagName",
			Label:       "Tag Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Tag to build from when Repository is used. Mutually exclusive with Branch Name and Commit SHA.",
			Placeholder: "e.g. v1.0.0",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"", "false"}},
			},
		},
		{
			Name:        "commitSha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Commit SHA to build from when Repository is used. Mutually exclusive with Branch Name and Tag Name.",
			Placeholder: "e.g. 5d7363a99d19e45830e1bc9622d2e4fa72d7229f",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useConnectedRepository", Values: []string{"", "false"}},
			},
		},
		{
			Name:        "images",
			Label:       "Images",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional list of Docker image names to push after the build completes.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Image",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "substitutions",
			Label:       "Substitutions",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: `JSON object of substitution key-value pairs.`,
			Placeholder: `{"_ENV":"production","_VERSION":"1.0.0"}`,
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Build timeout in seconds format (e.g. 600s). Defaults to 10 minutes.",
			Placeholder: "e.g. 600s",
		},
	}
}

func (c *CreateBuild) Setup(ctx core.SetupContext) error {
	if _, err := decodeCreateBuildConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this component to create builds")
	}

	if err := scheduleCloudBuildSetupIfNeeded(ctx.Integration); err != nil {
		return err
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{"type": SubscriptionType})
	if err != nil {
		return fmt.Errorf("failed to subscribe to Cloud Build notifications: %w", err)
	}

	return ctx.Metadata.Set(CreateBuildNodeMetadata{
		SubscriptionID: subscriptionID.String(),
	})
}

func (c *CreateBuild) Execute(ctx core.ExecutionContext) error {
	config, err := decodeCreateBuildConfiguration(ctx.Configuration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	build, err := buildRequest(config)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	projectID, url, err := buildCreateTarget(config.ProjectID, client.ProjectID(), build)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	responseBody, err := client.PostURL(context.Background(), url, build)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create build: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse response: %v", err))
	}

	buildData := extractBuildFromOperation(result)
	buildID := strings.TrimSpace(readBuildString(buildData, "id"))
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(buildData, "name"))
	}
	if buildID == "" {
		return ctx.ExecutionState.Fail("error", "Cloud Build create response did not include a build ID")
	}

	if err := storeCreateBuildMetadata(ctx.Metadata, buildData, projectID); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to store build metadata: %v", err))
	}

	if err := ctx.ExecutionState.SetKV(createBuildExecutionKV, buildID); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to track build execution: %v", err))
	}

	if isTerminalBuildStatus(readBuildString(buildData, "status")) {
		return completeCreateBuildExecution(ctx.ExecutionState, buildData)
	}

	return ctx.Requests.ScheduleActionCall(createBuildPollAction, map[string]any{}, createBuildPollInterval)
}

func buildRequest(config CreateBuildConfiguration) (map[string]any, error) {
	var steps []any
	if err := json.Unmarshal([]byte(config.Steps), &steps); err != nil {
		return nil, fmt.Errorf("steps must be a valid JSON array: %w", err)
	}

	build := map[string]any{
		"steps": steps,
	}

	if config.ConnectedRepository != "" {
		source, err := buildConnectedRepositorySource(config)
		if err != nil {
			return nil, err
		}
		build["source"] = source
	}

	if config.Source != "" {
		var source map[string]any
		if err := json.Unmarshal([]byte(config.Source), &source); err != nil {
			return nil, fmt.Errorf("source must be a valid JSON object: %w", err)
		}
		build["source"] = source
	}

	if config.RepoName != "" {
		source, err := buildRepositoryShortcutSource(config)
		if err != nil {
			return nil, err
		}
		build["source"] = source
	}

	if len(config.Images) > 0 {
		build["images"] = config.Images
	}

	if config.Substitutions != "" {
		var subs map[string]any
		if err := json.Unmarshal([]byte(config.Substitutions), &subs); err != nil {
			return nil, fmt.Errorf("substitutions must be a valid JSON object: %w", err)
		}
		build["substitutions"] = subs
	}

	if config.Timeout != "" {
		build["timeout"] = config.Timeout
	}

	return build, nil
}

// extractBuildFromOperation extracts the build metadata from the long-running operation response.
// The Cloud Build create API returns an Operation with build metadata.
func extractBuildFromOperation(op map[string]any) map[string]any {
	if meta, ok := op["metadata"].(map[string]any); ok {
		if build, ok := meta["build"].(map[string]any); ok {
			return build
		}
	}
	return op
}

func decodeCreateBuildConfiguration(raw any) (CreateBuildConfiguration, error) {
	config, err := normalizeCreateBuildConfiguration(raw)
	if err != nil {
		return CreateBuildConfiguration{}, err
	}
	if err := validateBuildSteps(config); err != nil {
		return CreateBuildConfiguration{}, err
	}
	if err := validateBuildSource(config); err != nil {
		return CreateBuildConfiguration{}, err
	}
	if usesConnectedRepository(config) {
		return validateConnectedRepositoryConfig(config)
	}
	return config, validateRepoShortcutConfig(config)
}

func normalizeCreateBuildConfiguration(raw any) (CreateBuildConfiguration, error) {
	config := CreateBuildConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return CreateBuildConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.Source = strings.TrimSpace(config.Source)
	config.ConnectionLocation = strings.TrimSpace(config.ConnectionLocation)
	config.Connection = strings.TrimSpace(config.Connection)
	config.ConnectedRepository = strings.TrimSpace(config.ConnectedRepository)
	config.ConnectedRevisionType = strings.ToLower(strings.TrimSpace(config.ConnectedRevisionType))
	config.ConnectedBranch = strings.TrimSpace(config.ConnectedBranch)
	config.ConnectedTag = strings.TrimSpace(config.ConnectedTag)
	config.ConnectedCommitSHA = strings.TrimSpace(config.ConnectedCommitSHA)
	config.RepoName = strings.TrimSpace(config.RepoName)
	config.BranchName = strings.TrimSpace(config.BranchName)
	config.TagName = strings.TrimSpace(config.TagName)
	config.CommitSHA = strings.TrimSpace(config.CommitSHA)
	config.Steps = strings.TrimSpace(config.Steps)
	config.Substitutions = strings.TrimSpace(config.Substitutions)
	config.Timeout = strings.TrimSpace(config.Timeout)
	return config, nil
}

func validateBuildSteps(config CreateBuildConfiguration) error {
	if config.Steps == "" {
		return fmt.Errorf("steps is required")
	}
	var steps []any
	if err := json.Unmarshal([]byte(config.Steps), &steps); err != nil {
		return fmt.Errorf("steps must be a valid JSON array: %w", err)
	}
	if len(steps) == 0 {
		return fmt.Errorf("steps must contain at least one build step")
	}
	return nil
}

func validateBuildSource(config CreateBuildConfiguration) error {
	if config.Source == "" {
		return nil
	}
	var source map[string]any
	if err := json.Unmarshal([]byte(config.Source), &source); err != nil {
		return fmt.Errorf("source must be a valid JSON object: %w", err)
	}
	if len(source) == 0 {
		return fmt.Errorf("source must not be empty when provided")
	}
	if config.RepoName != "" || config.BranchName != "" || config.TagName != "" || config.CommitSHA != "" {
		return fmt.Errorf("source cannot be combined with repoName, branchName, tagName, or commitSha")
	}
	return nil
}

func usesConnectedRepository(config CreateBuildConfiguration) bool {
	return config.UseConnectedRepository ||
		config.ConnectionLocation != "" ||
		config.Connection != "" ||
		config.ConnectedRepository != "" ||
		config.ConnectedRevisionType != "" ||
		config.ConnectedBranch != "" ||
		config.ConnectedTag != "" ||
		config.ConnectedCommitSHA != ""
}

func validateConnectedRepositoryConfig(config CreateBuildConfiguration) (CreateBuildConfiguration, error) {
	if config.Source != "" || config.RepoName != "" || config.BranchName != "" || config.TagName != "" || config.CommitSHA != "" {
		return CreateBuildConfiguration{}, fmt.Errorf("connected repository fields cannot be combined with source, repoName, branchName, tagName, or commitSha")
	}
	if config.ConnectionLocation == "" {
		return CreateBuildConfiguration{}, fmt.Errorf("connectionLocation is required when using a connected repository")
	}
	if config.Connection == "" {
		return CreateBuildConfiguration{}, fmt.Errorf("connection is required when using a connected repository")
	}
	if config.ConnectedRepository == "" {
		return CreateBuildConfiguration{}, fmt.Errorf("connectedRepository is required when using a connected repository")
	}

	repositoryProjectID, repositoryLocation, repositoryConnectionID, _ := parseCloudBuildRepositoryName(config.ConnectedRepository)
	if repositoryProjectID == "" || repositoryLocation == "" || repositoryConnectionID == "" {
		return CreateBuildConfiguration{}, fmt.Errorf("connectedRepository must be a valid Cloud Build repository resource name")
	}
	if config.ProjectID != "" && config.ProjectID != repositoryProjectID {
		return CreateBuildConfiguration{}, fmt.Errorf("projectId override must match the connected repository project")
	}
	if config.ConnectionLocation != repositoryLocation {
		return CreateBuildConfiguration{}, fmt.Errorf("connectionLocation must match the connected repository location")
	}
	if config.Connection != fmt.Sprintf("projects/%s/locations/%s/connections/%s", repositoryProjectID, repositoryLocation, repositoryConnectionID) {
		return CreateBuildConfiguration{}, fmt.Errorf("connection must match the selected connected repository")
	}

	config = inferConnectedRevisionType(config)

	if err := validateConnectedRevision(config); err != nil {
		return CreateBuildConfiguration{}, err
	}

	return config, nil
}

func inferConnectedRevisionType(config CreateBuildConfiguration) CreateBuildConfiguration {
	if config.ConnectedRevisionType != "" {
		return config
	}
	switch {
	case config.ConnectedBranch != "" && config.ConnectedTag == "" && config.ConnectedCommitSHA == "":
		config.ConnectedRevisionType = createBuildConnectedRevisionBranch
	case config.ConnectedTag != "" && config.ConnectedBranch == "" && config.ConnectedCommitSHA == "":
		config.ConnectedRevisionType = createBuildConnectedRevisionTag
	case config.ConnectedCommitSHA != "" && config.ConnectedBranch == "" && config.ConnectedTag == "":
		config.ConnectedRevisionType = createBuildConnectedRevisionCommit
	default:
		config.ConnectedRevisionType = createBuildConnectedRevisionBranch
	}
	return config
}

func validateConnectedRevision(config CreateBuildConfiguration) error {
	set := 0
	if config.ConnectedBranch != "" {
		set++
	}
	if config.ConnectedTag != "" {
		set++
	}
	if config.ConnectedCommitSHA != "" {
		set++
	}
	if set > 1 {
		return fmt.Errorf("connectedBranch, connectedTag, and connectedCommitSha are mutually exclusive")
	}

	switch config.ConnectedRevisionType {
	case createBuildConnectedRevisionBranch:
		if config.ConnectedBranch == "" {
			return fmt.Errorf("connectedBranch is required when connectedRevisionType is branch")
		}
	case createBuildConnectedRevisionTag:
		if config.ConnectedTag == "" {
			return fmt.Errorf("connectedTag is required when connectedRevisionType is tag")
		}
	case createBuildConnectedRevisionCommit:
		if config.ConnectedCommitSHA == "" {
			return fmt.Errorf("connectedCommitSha is required when connectedRevisionType is commit")
		}
	default:
		return fmt.Errorf("connectedRevisionType must be one of branch, tag, or commit")
	}

	return nil
}

func validateRepoShortcutConfig(config CreateBuildConfiguration) error {
	if config.RepoName == "" && (config.BranchName != "" || config.TagName != "" || config.CommitSHA != "") {
		return fmt.Errorf("repoName is required when branchName, tagName, or commitSha is set")
	}
	if config.RepoName != "" && config.BranchName == "" && config.TagName == "" && config.CommitSHA == "" {
		return fmt.Errorf("branchName, tagName, or commitSha is required when repoName is set")
	}
	set := 0
	if config.BranchName != "" {
		set++
	}
	if config.TagName != "" {
		set++
	}
	if config.CommitSHA != "" {
		set++
	}
	if set > 1 {
		return fmt.Errorf("branchName, tagName, and commitSha are mutually exclusive")
	}
	return nil
}

func buildConnectedRepositorySource(config CreateBuildConfiguration) (map[string]any, error) {
	revision, err := connectedRepositoryRevision(config)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"connectedRepository": map[string]any{
			"repository": config.ConnectedRepository,
			"revision":   revision,
		},
	}, nil
}

func connectedRepositoryRevision(config CreateBuildConfiguration) (string, error) {
	switch config.ConnectedRevisionType {
	case createBuildConnectedRevisionBranch:
		if config.ConnectedBranch == "" {
			return "", fmt.Errorf("connectedBranch is required when connectedRevisionType is branch")
		}
		return config.ConnectedBranch, nil
	case createBuildConnectedRevisionTag:
		if config.ConnectedTag == "" {
			return "", fmt.Errorf("connectedTag is required when connectedRevisionType is tag")
		}
		return config.ConnectedTag, nil
	case createBuildConnectedRevisionCommit:
		if config.ConnectedCommitSHA == "" {
			return "", fmt.Errorf("connectedCommitSha is required when connectedRevisionType is commit")
		}
		return config.ConnectedCommitSHA, nil
	default:
		return "", fmt.Errorf("connectedRevisionType must be one of branch, tag, or commit")
	}
}

func buildRepositoryShortcutSource(config CreateBuildConfiguration) (map[string]any, error) {
	revisionValue, revisionKey, err := repositoryRevision(config)
	if err != nil {
		return nil, err
	}

	repo := config.RepoName
	if isGitRepositoryURL(repo) {
		repo = normalizeGitRepositoryURL(repo)
		return map[string]any{
			"gitSource": map[string]any{
				"url":      repo,
				"revision": revisionValue,
			},
		}, nil
	}

	return map[string]any{
		"repoSource": map[string]any{
			"repoName":  repo,
			revisionKey: revisionValue,
		},
	}, nil
}

func repositoryRevision(config CreateBuildConfiguration) (string, string, error) {
	revisionFields := 0
	if config.BranchName != "" {
		revisionFields++
	}
	if config.TagName != "" {
		revisionFields++
	}
	if config.CommitSHA != "" {
		revisionFields++
	}
	if revisionFields > 1 {
		return "", "", fmt.Errorf("branchName, tagName, and commitSha are mutually exclusive")
	}

	if config.BranchName != "" {
		return config.BranchName, "branchName", nil
	}
	if config.TagName != "" {
		return config.TagName, "tagName", nil
	}
	if config.CommitSHA != "" {
		return config.CommitSHA, "commitSha", nil
	}

	return "", "", fmt.Errorf("branchName, tagName, or commitSha is required when repoName is set")
}

func isGitRepositoryURL(value string) bool {
	repo := strings.ToLower(value)
	return strings.HasPrefix(repo, "https://") ||
		strings.HasPrefix(repo, "http://") ||
		strings.HasPrefix(repo, "ssh://") ||
		strings.HasPrefix(repo, "git@") ||
		strings.Contains(repo, "github.com/") ||
		strings.Contains(repo, "gitlab.com/") ||
		strings.Contains(repo, "bitbucket.org/") ||
		strings.HasSuffix(repo, ".git")
}

func normalizeGitRepositoryURL(value string) string {
	repo := value
	lower := strings.ToLower(repo)
	if strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "ssh://") ||
		strings.HasPrefix(lower, "git@") {
		return repo
	}

	if strings.Contains(lower, "github.com/") ||
		strings.Contains(lower, "gitlab.com/") ||
		strings.Contains(lower, "bitbucket.org/") {
		return "https://" + repo
	}

	return repo
}

func storeCreateBuildMetadata(metadataCtx core.MetadataContext, build map[string]any, projectID string) error {
	buildCopy := copyBuildMetadata(build)

	if readBuildString(buildCopy, "projectId") == "" && projectID != "" {
		buildCopy["projectId"] = projectID
	}

	return metadataCtx.Set(CreateBuildExecutionMetadata{Build: buildCopy})
}

func copyBuildMetadata(build map[string]any) map[string]any {
	if build == nil {
		return map[string]any{}
	}

	buildCopy := make(map[string]any, len(build))
	maps.Copy(buildCopy, build)
	return buildCopy
}

func readBuildString(build map[string]any, key string) string {
	value, ok := build[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func isTerminalBuildStatus(status string) bool {
	return slices.Contains([]string{
		"SUCCESS",
		"FAILURE",
		"INTERNAL_ERROR",
		"TIMEOUT",
		"CANCELLED",
		"EXPIRED",
	}, strings.ToUpper(strings.TrimSpace(status)))
}

func completeCreateBuildExecution(executionState core.ExecutionStateContext, build map[string]any) error {
	status := strings.ToUpper(readBuildString(build, "status"))
	if status == "SUCCESS" {
		return executionState.Emit(createBuildPassedOutputChannel, createBuildPayloadType, []any{build})
	}

	return executionState.Emit(createBuildFailedOutputChannel, createBuildPayloadType, []any{build})
}

func (c *CreateBuild) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	config, err := decodeCreateBuildConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	var metadata CreateBuildExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	buildID := readBuildString(metadata.Build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(metadata.Build, "name"))
	}
	if buildID == "" {
		return fmt.Errorf("build metadata is missing id")
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	projectID := config.ProjectID
	if projectID == "" {
		projectID = client.ProjectID()
	}

	url := buildGetURL(projectID, buildID, readBuildString(metadata.Build, "name"))
	responseBody, err := client.GetURL(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}

	var build map[string]any
	if err := json.Unmarshal(responseBody, &build); err != nil {
		return fmt.Errorf("failed to parse build response: %w", err)
	}

	if err := storeCreateBuildMetadata(ctx.Metadata, build, projectID); err != nil {
		return fmt.Errorf("failed to store build metadata: %w", err)
	}

	if !isTerminalBuildStatus(readBuildString(build, "status")) {
		return ctx.Requests.ScheduleActionCall(createBuildPollAction, map[string]any{}, createBuildPollInterval)
	}

	return completeCreateBuildExecution(ctx.ExecutionState, build)
}

func (c *CreateBuild) Actions() []core.Action {
	return []core.Action{
		{
			Name:           createBuildPollAction,
			UserAccessible: false,
		},
	}
}

func (c *CreateBuild) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case createBuildPollAction:
		return c.poll(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *CreateBuild) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	if ctx.FindExecutionByKV == nil {
		return nil
	}

	build, ok := ctx.Message.(map[string]any)
	if !ok {
		return nil
	}

	buildID := readBuildString(build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(build, "name"))
	}
	if buildID == "" {
		return nil
	}

	executionCtx, err := ctx.FindExecutionByKV(createBuildExecutionKV, buildID)
	if err != nil || executionCtx == nil {
		return err
	}

	if executionCtx.ExecutionState.IsFinished() {
		return nil
	}

	if err := storeCreateBuildMetadata(executionCtx.Metadata, build, readBuildString(build, "projectId")); err != nil {
		return fmt.Errorf("failed to store build metadata: %w", err)
	}

	if !isTerminalBuildStatus(readBuildString(build, "status")) {
		return nil
	}

	return completeCreateBuildExecution(executionCtx.ExecutionState, build)
}

func (c *CreateBuild) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateBuild) Cancel(ctx core.ExecutionContext) error {
	var metadata CreateBuildExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("decode create build metadata: %w", err)
	}

	buildID := readBuildString(metadata.Build, "id")
	if buildID == "" {
		buildID = buildIDFromName(readBuildString(metadata.Build, "name"))
	}
	if buildID == "" {
		return nil
	}

	if isTerminalBuildStatus(readBuildString(metadata.Build, "status")) {
		return nil
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create GCP client: %w", err)
	}

	projectID := readBuildString(metadata.Build, "projectId")
	if projectID == "" {
		projectID = client.ProjectID()
	}

	cancelURL := buildCancelURL(projectID, buildID, readBuildString(metadata.Build, "name"))
	if _, err := client.PostURL(context.Background(), cancelURL, map[string]any{}); err != nil {
		return fmt.Errorf("cancel Cloud Build build %s: %w", buildID, err)
	}

	cancelledBuild := copyBuildMetadata(metadata.Build)
	cancelledBuild["status"] = "CANCELLED"
	if err := storeCreateBuildMetadata(ctx.Metadata, cancelledBuild, projectID); err != nil {
		return fmt.Errorf("store cancelled build metadata: %w", err)
	}

	return nil
}
func (c *CreateBuild) Cleanup(_ core.SetupContext) error { return nil }
func (c *CreateBuild) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
