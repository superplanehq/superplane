package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const kbPollInterval = 30 * time.Second

type CreateKnowledgeBase struct{}

// DataSourceSpec represents one data source item in the dataSources list
type DataSourceSpec struct {
	// "spaces" | "web"
	Type string `json:"type" mapstructure:"type"`

	// Spaces bucket field — value is the integration resource ID in "region/bucket-name" format
	SpacesBucket string `json:"spacesBucket" mapstructure:"spacesBucket"`

	// Web / sitemap URL fields
	WebURL             string `json:"webURL" mapstructure:"webURL"`
	CrawlType          string `json:"crawlType" mapstructure:"crawlType"`           // "seed" | "sitemap"
	CrawlingOption     string `json:"crawlingOption" mapstructure:"crawlingOption"` // seed only: SCOPED, URL_AND_LINKED_PAGES_IN_PATH, etc.
	WebEmbedMedia      bool   `json:"webEmbedMedia" mapstructure:"webEmbedMedia"`
	WebIncludeNavLinks bool   `json:"webIncludeNavLinks" mapstructure:"webIncludeNavLinks"`

	// Chunking (applies to all source types)
	ChunkingAlgorithm string  `json:"chunkingAlgorithm" mapstructure:"chunkingAlgorithm"`
	MaxChunkSize      int     `json:"maxChunkSize" mapstructure:"maxChunkSize"`
	SemanticThreshold float64 `json:"semanticThreshold" mapstructure:"semanticThreshold"`
	ParentChunkSize   int     `json:"parentChunkSize" mapstructure:"parentChunkSize"`
	ChildChunkSize    int     `json:"childChunkSize" mapstructure:"childChunkSize"`
}

// CreateKnowledgeBaseSpec is the decoded configuration for the CreateKnowledgeBase component
type CreateKnowledgeBaseSpec struct {
	Name           string           `json:"name" mapstructure:"name"`
	EmbeddingModel string           `json:"embeddingModel" mapstructure:"embeddingModel"`
	Region         string           `json:"region" mapstructure:"region"`
	Project        string           `json:"project" mapstructure:"project"`
	Tags           []string         `json:"tags" mapstructure:"tags"`
	DatabaseOption string           `json:"databaseOption" mapstructure:"databaseOption"` // "new" | "existing"
	Database       string           `json:"database" mapstructure:"database"`
	DataSources    []DataSourceSpec `json:"dataSources" mapstructure:"dataSources"`
}

func (c *CreateKnowledgeBase) Name() string {
	return "digitalocean.createKnowledgeBase"
}

func (c *CreateKnowledgeBase) Label() string {
	return "Create Knowledge Base"
}

func (c *CreateKnowledgeBase) Description() string {
	return "Create a DigitalOcean Gradient AI knowledge base with one or more data sources"
}

func (c *CreateKnowledgeBase) Documentation() string {
	return `The Create Knowledge Base component creates a new knowledge base on the DigitalOcean Gradient AI Platform, ready for use with AI agents via retrieval-augmented generation (RAG).

## How it works

A knowledge base converts your data sources into vector embeddings using the selected embedding model. Those embeddings are stored in an OpenSearch database — either a newly provisioned one or one you already have. Once created, the knowledge base can be attached to any Gradient AI agent.

## Data Sources

You can add multiple data sources of different types:

- **Spaces Bucket or Folder** — indexes all supported files in a DigitalOcean Spaces bucket or folder
- **Web or Sitemap URL** — crawls a public website (seed URL) or a list of URLs from a sitemap

Each data source has its own independent chunking strategy.

## Chunking Strategies

- **Section-based** (default) — splits on structural elements like headings and paragraphs; fast and low-cost
- **Semantic** — groups sentences by meaning; slower but context-aware
- **Hierarchical** — creates parent (context) and child (retrieval) chunk pairs
- **Fixed-length** — splits strictly by token count; best for logs and unstructured text

## OpenSearch Database

The knowledge base requires an OpenSearch database to store the vector embeddings:
- **Create new** — provisions a new database automatically sized to your data
- **Use existing** — connects to a database you already have by providing its ID

## Output

Returns the created knowledge base including:
- **uuid**: Knowledge base UUID for use in downstream components
- **name**: Name of the knowledge base
- **region**: Datacenter region
- **embeddingModelUUID**: UUID of the embedding model used
- **projectId**: Associated project ID
- **databaseId**: UUID of the OpenSearch database (populated after provisioning completes for new databases)
- **createdAt**: Creation timestamp`
}

func (c *CreateKnowledgeBase) Icon() string {
	return "brain"
}

func (c *CreateKnowledgeBase) Color() string {
	return "blue"
}

func (c *CreateKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Knowledge Base Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A unique name for the knowledge base.",
			Placeholder: "my-knowledge-base",
		},
		{
			Name:        "embeddingModel",
			Label:       "Embedding Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The embedding model used to convert your data into vector embeddings. This cannot be changed after the knowledge base is created. When using an expression, provide the embedding model UUID.",
			Placeholder: "Select an embedding model",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "embedding_model",
				},
			},
		},
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "DigitalOcean project to associate with the knowledge base. When using an expression, provide the project UUID.",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:      "tags",
			Label:     "Tags",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			Description: "Optional tags to organize and filter the knowledge base. " +
				"Tags can include letters, numbers, colons, dashes, and underscores.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:     "databaseOption",
			Label:    "OpenSearch Database",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "new",
			Description: "The knowledge base stores vector embeddings in an OpenSearch database. " +
				"Choose to create a new one or connect to an existing database.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Create new database", Value: "new"},
						{Label: "Use existing database", Value: "existing"},
					},
				},
			},
		},
		{
			Name:        "database",
			Label:       "OpenSearch Database",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The existing OpenSearch database cluster to connect to. When using an expression, provide the database UUID.",
			Placeholder: "Select an OpenSearch database",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "databaseOption", Values: []string{"existing"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "databaseOption", Values: []string{"existing"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "opensearch_database",
				},
			},
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "tor1",
			Description: "Datacenter region where the new OpenSearch database and knowledge base will be provisioned. TOR1 is recommended — most Gradient AI Platform infrastructure is located there.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "databaseOption", Values: []string{"new"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "databaseOption", Values: []string{"new"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: kbRegionOptions,
				},
			},
		},
		{
			Name:        "dataSources",
			Label:       "Data Sources",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "One or more data sources to index into the knowledge base. Each source has its own independent chunking strategy.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Data Source",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: dataSourceItemSchema(),
					},
				},
			},
		},
	}
}

func (c *CreateKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := CreateKnowledgeBaseSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.EmbeddingModel == "" {
		return errors.New("embeddingModel is required")
	}

	if spec.Project == "" {
		return errors.New("project is required")
	}

	if spec.DatabaseOption == "new" && spec.Region == "" {
		return errors.New("region is required when creating a new database")
	}

	if spec.DatabaseOption == "existing" && spec.Database == "" {
		return errors.New("database is required when using an existing database")
	}

	if len(spec.DataSources) == 0 {
		return errors.New("at least one data source is required")
	}

	for i, ds := range spec.DataSources {
		if err := validateDataSource(i+1, ds); err != nil {
			return err
		}
	}

	return nil
}

// kbMetadata is stored between poll ticks
type kbMetadata struct {
	KBUUID   string         `json:"kbUUID" mapstructure:"kbUUID"`
	KBOutput map[string]any `json:"kbOutput" mapstructure:"kbOutput"`
}

func (c *CreateKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := CreateKnowledgeBaseSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	req := CreateKnowledgeBaseRequest{
		Name:               spec.Name,
		EmbeddingModelUUID: spec.EmbeddingModel,
		ProjectID:          spec.Project,
		Tags:               spec.Tags,
		DataSources:        buildKBDataSources(spec.DataSources),
	}

	if spec.Region != "" {
		req.Region = spec.Region
	}

	if spec.DatabaseOption == "existing" {
		req.DatabaseID = spec.Database
	}

	kb, err := client.CreateKnowledgeBase(req)
	if err != nil {
		return fmt.Errorf("failed to create knowledge base: %v", err)
	}

	output := map[string]any{
		"uuid":               kb.UUID,
		"name":               kb.Name,
		"region":             kb.Region,
		"embeddingModelUUID": kb.EmbeddingModelUUID,
		"projectId":          kb.ProjectID,
		"databaseId":         kb.DatabaseID,
		"tags":               kb.Tags,
		"createdAt":          kb.CreatedAt,
	}

	resolveDisplayNames(client, spec, output)

	if err := ctx.Metadata.Set(kbMetadata{
		KBUUID:   kb.UUID,
		KBOutput: output,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, kbPollInterval)
}

// mergeCreateKBOutputFromFetchedKB updates the Execute-time output map with fields that the create
// response may omit until async provisioning finishes (e.g. databaseId for a newly created OpenSearch cluster).
func mergeCreateKBOutputFromFetchedKB(output map[string]any, kb *KnowledgeBase) {
	if output == nil || kb == nil {
		return
	}
	if kb.DatabaseID != "" {
		output["databaseId"] = kb.DatabaseID
	}
}

// resolveDisplayNames enriches the output map with human-readable names for
// the embedding model, project, and OpenSearch database. Failures are ignored
// so a lookup error never blocks the execution result.
func resolveDisplayNames(client *Client, spec CreateKnowledgeBaseSpec, output map[string]any) {
	if models, err := client.ListEmbeddingModels(); err == nil {
		for _, m := range models {
			if m.UUID == spec.EmbeddingModel {
				output["embeddingModelName"] = m.Name
				break
			}
		}
	}

	if projects, err := client.ListProjects(); err == nil {
		for _, p := range projects {
			if p.ID == spec.Project {
				output["projectName"] = p.Name
				break
			}
		}
	}

	if spec.DatabaseOption == "existing" && spec.Database != "" {
		if databases, err := client.ListDatabasesByEngine("opensearch"); err == nil {
			for _, db := range databases {
				if db.ID == spec.Database {
					output["databaseName"] = db.Name
					output["databaseStatus"] = db.Status
					break
				}
			}
		}
	}
}

// validateDataSource validates a single data source spec and returns an error if invalid
func validateDataSource(index int, ds DataSourceSpec) error {
	switch ds.Type {
	case "spaces":
		if err := validateSpacesDataSource(index, ds); err != nil {
			return err
		}
	case "web":
		if err := validateWebDataSource(index, ds); err != nil {
			return err
		}
	case "":
		return fmt.Errorf("data source %d: type is required", index)
	default:
		return fmt.Errorf("data source %d: unsupported type %q, must be 'spaces' or 'web'", index, ds.Type)
	}

	return validateChunking(index, ds)
}

func validateSpacesDataSource(index int, ds DataSourceSpec) error {
	if ds.SpacesBucket == "" {
		return fmt.Errorf("data source %d: spacesBucket is required", index)
	}

	if _, _, err := parseSpacesBucketID(ds.SpacesBucket); err != nil {
		return fmt.Errorf("data source %d: %w", index, err)
	}

	return nil
}

func validateWebDataSource(index int, ds DataSourceSpec) error {
	if ds.WebURL == "" {
		return fmt.Errorf("data source %d: webURL is required", index)
	}

	if ds.CrawlType == "" {
		return fmt.Errorf("data source %d: crawlType is required", index)
	}

	if ds.CrawlType != "seed" && ds.CrawlType != "sitemap" {
		return fmt.Errorf("data source %d: crawlType must be 'seed' or 'sitemap', got %q", index, ds.CrawlType)
	}

	if ds.CrawlType == "seed" && ds.CrawlingOption == "" {
		return fmt.Errorf("data source %d: crawlingOption is required for seed URLs", index)
	}

	return nil
}

// validateChunking validates the chunking configuration for a data source
func validateChunking(index int, ds DataSourceSpec) error {
	if ds.ChunkingAlgorithm == "" {
		return nil
	}

	switch ds.ChunkingAlgorithm {
	case chunkingSectionBased, chunkingSemantic, chunkingHierarchical, chunkingFixedLength:
		// valid
	default:
		return fmt.Errorf("data source %d: unsupported chunking algorithm %q", index, ds.ChunkingAlgorithm)
	}

	if ds.ChunkingAlgorithm == chunkingHierarchical {
		if ds.ParentChunkSize == 0 {
			return fmt.Errorf("data source %d: parentChunkSize is required for hierarchical chunking", index)
		}
		if ds.ChildChunkSize == 0 {
			return fmt.Errorf("data source %d: childChunkSize is required for hierarchical chunking", index)
		}
		if ds.ChildChunkSize >= ds.ParentChunkSize {
			return fmt.Errorf("data source %d: childChunkSize (%d) must be smaller than parentChunkSize (%d)",
				index, ds.ChildChunkSize, ds.ParentChunkSize)
		}
	}

	return nil
}

// parseSpacesBucketID splits the integration resource ID "region/bucket-name" into its parts.
// Returns an error if the format is unexpected.
func parseSpacesBucketID(id string) (region, bucketName string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid spacesBucket value %q: expected \"region/bucket-name\" format", id)
	}
	return parts[0], parts[1], nil
}

// buildKBDataSources maps a slice of DataSourceSpec to API KBDataSource objects
func buildKBDataSources(specs []DataSourceSpec) []KBDataSource {
	result := make([]KBDataSource, 0, len(specs))
	for _, ds := range specs {
		result = append(result, buildKBDataSource(ds))
	}
	return result
}

func buildKBDataSource(ds DataSourceSpec) KBDataSource {
	kbDS := KBDataSource{}

	switch ds.Type {
	case "spaces":
		kbDS.SpacesDataSource = buildSpacesDataSource(ds)
	case "web":
		kbDS.WebCrawlerDataSource = buildWebDataSource(ds)
	}

	algorithm := ds.ChunkingAlgorithm
	if algorithm == "" {
		algorithm = chunkingSectionBased
	}
	kbDS.ChunkingAlgorithm = algorithm

	if opts := buildChunkingOptions(ds); opts != nil {
		kbDS.ChunkingOptions = opts
	}

	return kbDS
}

func buildSpacesDataSource(ds DataSourceSpec) *KBSpacesDataSource {
	region, bucketName, _ := parseSpacesBucketID(ds.SpacesBucket) // already validated in Setup
	return &KBSpacesDataSource{
		BucketName: bucketName,
		Region:     region,
	}
}

func buildWebDataSource(ds DataSourceSpec) *KBWebCrawlerDataSource {
	crawlingOption := ds.CrawlingOption
	if ds.CrawlType == "sitemap" {
		crawlingOption = "SITEMAP"
	}

	excludeTags := defaultWebExcludeTags
	if ds.WebIncludeNavLinks {
		excludeTags = webExcludeTagsWithNav
	}

	return &KBWebCrawlerDataSource{
		BaseURL:        ds.WebURL,
		CrawlingOption: crawlingOption,
		EmbedMedia:     ds.WebEmbedMedia,
		ExcludeTags:    excludeTags,
	}
}

func buildChunkingOptions(ds DataSourceSpec) *KBChunkingOptions {
	opts := &KBChunkingOptions{}
	hasOptions := false

	if ds.MaxChunkSize > 0 {
		opts.MaxChunkSize = ds.MaxChunkSize
		hasOptions = true
	}
	if ds.SemanticThreshold > 0 {
		opts.SemanticThreshold = ds.SemanticThreshold
		hasOptions = true
	}
	if ds.ParentChunkSize > 0 {
		opts.ParentChunkSize = ds.ParentChunkSize
		hasOptions = true
	}
	if ds.ChildChunkSize > 0 {
		opts.ChildChunkSize = ds.ChildChunkSize
		hasOptions = true
	}

	if !hasOptions {
		return nil
	}
	return opts
}

// Chunking algorithm constants
const (
	chunkingSectionBased = "CHUNKING_ALGORITHM_SECTION_BASED"
	chunkingSemantic     = "CHUNKING_ALGORITHM_SEMANTIC"
	chunkingHierarchical = "CHUNKING_ALGORITHM_HIERARCHICAL"
	chunkingFixedLength  = "CHUNKING_ALGORITHM_FIXED_LENGTH"
)

// defaultWebExcludeTags excludes navigation/layout and non-content tags.
// This is the default — navigation links repeat on every page and dilute embedding quality.
var defaultWebExcludeTags = []string{
	"nav", "footer", "header", "aside", "script", "style", "form", "iframe", "noscript",
}

// webExcludeTagsWithNav keeps navigation elements but still strips non-content tags.
// Used when the user opts in to include header/footer navigation links.
var webExcludeTagsWithNav = []string{
	"script", "style", "form", "iframe", "noscript",
}

var kbRegionOptions = []configuration.FieldOption{
	{Label: "Toronto (tor1) — Recommended", Value: "tor1"},
	{Label: "New York City 1 (nyc1)", Value: "nyc1"},
	{Label: "New York City 3 (nyc3)", Value: "nyc3"},
	{Label: "San Francisco 2 (sfo2)", Value: "sfo2"},
	{Label: "San Francisco 3 (sfo3)", Value: "sfo3"},
	{Label: "Amsterdam 3 (ams3)", Value: "ams3"},
	{Label: "Singapore (sgp1)", Value: "sgp1"},
	{Label: "Frankfurt (fra1)", Value: "fra1"},
	{Label: "Bangalore (blr1)", Value: "blr1"},
	{Label: "Sydney (syd1)", Value: "syd1"},
	{Label: "London (lon1)", Value: "lon1"},
}

var chunkingAlgorithmOptions = []configuration.FieldOption{
	{Label: "Section-based (default)", Value: chunkingSectionBased},
	{Label: "Semantic", Value: chunkingSemantic},
	{Label: "Hierarchical", Value: chunkingHierarchical},
	{Label: "Fixed-length", Value: chunkingFixedLength},
}

func (c *CreateKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// indexJobState normalises a DO indexing job status to a simple lowercase keyword.
// The DO API returns prefixed enum values like "INDEXING_JOB_STATUS_COMPLETED",
// but may also return plain values like "completed". We match by suffix so both
// forms are handled correctly.
func indexJobState(status string) string {
	lower := strings.ToLower(status)
	for _, state := range []string{"completed", "successful", "running", "pending", "failed", "cancelled"} {
		if strings.HasSuffix(lower, state) {
			return state
		}
	}
	return lower
}

func (c *CreateKnowledgeBase) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta kbMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// The DO API embeds the latest indexing job directly in the KB response
	// under the "last_indexing_job" field — no separate list endpoint needed.
	kb, err := client.GetKnowledgeBase(meta.KBUUID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge base: %v", err)
	}

	mergeCreateKBOutputFromFetchedKB(meta.KBOutput, kb)

	if kb.LastIndexingJob != nil {
		switch indexJobState(kb.LastIndexingJob.Status) {
		case "completed", "successful":
			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"digitalocean.knowledge_base.created",
				[]any{meta.KBOutput},
			)
		case "running", "pending":
			return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, kbPollInterval)
		case "failed", "cancelled":
			return fmt.Errorf("indexing job %s for knowledge base %s", kb.LastIndexingJob.Status, meta.KBUUID)
		default:
			// Unknown status — wait and retry rather than failing
			return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, kbPollInterval)
		}
	}

	// No indexing job yet — database may still be provisioning.
	if strings.Contains(strings.ToLower(kb.Status), "provisioning") || strings.Contains(strings.ToLower(kb.Status), "pending") {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, kbPollInterval)
	}

	// KB is ready but no job started yet — start one.
	if _, err := client.StartIndexingJob(meta.KBUUID); err != nil {
		return fmt.Errorf("failed to start indexing job: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, kbPollInterval)
}

func (c *CreateKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}

// dataSourceItemSchema returns the schema for a single data source object inside the list
func dataSourceItemSchema() []configuration.Field {
	return []configuration.Field{
		// ── Source type selector ───────────────────────────────────────────
		{
			Name:     "type",
			Label:    "Source Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Spaces Bucket or Folder", Value: "spaces"},
						{Label: "Web or Sitemap URL", Value: "web"},
					},
				},
			},
		},

		// ── SPACES FIELDS ─────────────────────────────────────────────────
		{
			Name:        "spacesBucket",
			Label:       "Spaces Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The DigitalOcean Spaces bucket to index. Region is inferred automatically from the selected bucket.",
			Placeholder: "Select a bucket",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "spaces_bucket",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"spaces"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"spaces"}},
			},
		},

		// ── WEB FIELDS ────────────────────────────────────────────────────
		{
			Name:        "crawlType",
			Label:       "URL Type",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "seed",
			Description: "Seed URL crawls the page and linked pages. Sitemap URL crawls all URLs listed in the sitemap (.xml format).",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Seed URL", Value: "seed"},
						{Label: "Sitemap URL", Value: "sitemap"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"web"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"web"}},
			},
		},
		{
			Name:        "webURL",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The public URL to crawl, or the sitemap URL (must be in .xml format for sitemaps)",
			Placeholder: "https://docs.example.com",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"web"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"web"}},
			},
		},
		{
			Name:        "crawlingOption",
			Label:       "Crawling Scope",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "SCOPED",
			Description: "Defines how far the crawler follows links from the seed URL",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Scoped (seed URL only)", Value: "SCOPED"},
						{Label: "URL and linked pages in path", Value: "URL_AND_LINKED_PAGES_IN_PATH"},
						{Label: "URL and linked pages in domain", Value: "URL_AND_LINKED_PAGES_IN_DOMAIN"},
						{Label: "Subdomains", Value: "SUBDOMAINS"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"web"}},
				{Field: "crawlType", Values: []string{"seed"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: []string{"web"}},
				{Field: "crawlType", Values: []string{"seed"}},
			},
		},
		{
			Name:        "webEmbedMedia",
			Label:       "Index Embedded Media",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Index supported images and other media encountered during the crawl. May significantly increase indexing token count.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"web"}},
			},
		},
		{
			Name:        "webIncludeNavLinks",
			Label:       "Include Header and Footer Navigation Links",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Crawl links found in headers, footers, and navigation elements. Disabled by default as navigation links repeat on every page and dilute embedding quality.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{"web"}},
			},
		},

		// ── CHUNKING (ALL SOURCE TYPES) ───────────────────────────────────
		{
			Name:     "chunkingAlgorithm",
			Label:    "Chunking Strategy",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  chunkingSectionBased,
			Description: "How content is split before embedding and indexing. " +
				"Section-based is fast and low-cost. Semantic groups text by meaning. " +
				"Hierarchical creates parent and child chunk levels. Fixed-length splits strictly by token count.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: chunkingAlgorithmOptions,
				},
			},
		},
		{
			Name:        "maxChunkSize",
			Label:       "Maximum Chunk Size (tokens)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Maximum number of tokens per chunk. The valid range is shown in the selected embedding model above (e.g. 100–512 tokens).",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "chunkingAlgorithm", Values: []string{
					chunkingSectionBased,
					chunkingSemantic,
					chunkingFixedLength,
				}},
			},
		},
		{
			Name:      "semanticThreshold",
			Label:     "Similarity Threshold",
			Type:      configuration.FieldTypeNumber,
			Required:  false,
			Togglable: true,
			Description: "Value between 0 and 1. Lower values create larger sentence groups by allowing " +
				"less similar sentences to be grouped together.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "chunkingAlgorithm", Values: []string{chunkingSemantic}},
			},
		},
		{
			Name:        "parentChunkSize",
			Label:       "Maximum Parent Chunk Size (tokens)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Placeholder: "e.g. 256 for All MiniLM, 750 for GTE Large / Qwen3",
			Description: "Maximum tokens in the parent (context) chunk. The valid range is shown in the selected embedding model above. Must be larger than the child chunk size.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "chunkingAlgorithm", Values: []string{chunkingHierarchical}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "chunkingAlgorithm", Values: []string{chunkingHierarchical}},
			},
		},
		{
			Name:        "childChunkSize",
			Label:       "Maximum Child Chunk Size (tokens)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Placeholder: "e.g. 128 for All MiniLM, 375 for GTE Large / Qwen3",
			Description: "Maximum tokens in the child (retrieval) chunk. The valid range is shown in the selected embedding model above. Must be smaller than the parent chunk size.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "chunkingAlgorithm", Values: []string{chunkingHierarchical}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "chunkingAlgorithm", Values: []string{chunkingHierarchical}},
			},
		},
	}
}
