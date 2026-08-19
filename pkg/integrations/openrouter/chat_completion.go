package openrouter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/attachments"
	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

const ChatCompletionPayloadType = "openrouter.chatCompletion.result"

// maxAttachmentBytes caps the combined size of attachments on one request.
// OpenRouter has no Files API, so attachments are inlined into the request body
// as base64 data URLs — roughly a third larger again on the wire.
const maxAttachmentBytes = 8 * 1024 * 1024

// sortAuto leaves provider ranking to OpenRouter. It needs a non-empty value
// because the select renderer cannot represent an empty option.
const sortAuto = "auto"

var sortOptions = []configuration.FieldOption{
	{Label: "Balanced (OpenRouter default)", Value: sortAuto},
	{Label: "Lowest price", Value: "price"},
	{Label: "Highest throughput", Value: "throughput"},
	{Label: "Lowest latency", Value: "latency"},
}

var dataCollectionOptions = []configuration.FieldOption{
	{Label: "Allow", Value: "allow"},
	{Label: "Deny", Value: "deny"},
}

type ChatCompletion struct{}

type ChatCompletionSpec struct {
	Model         string               `json:"model" mapstructure:"model"`
	Prompt        string               `json:"prompt" mapstructure:"prompt"`
	SystemPrompt  string               `json:"systemPrompt" mapstructure:"systemPrompt"`
	Files         []string             `json:"files" mapstructure:"files"`
	MaxTokens     *int                 `json:"maxTokens" mapstructure:"maxTokens"`
	Temperature   *float64             `json:"temperature" mapstructure:"temperature"`
	Models        []string             `json:"models" mapstructure:"models"`
	Provider      *ProviderRoutingSpec `json:"provider" mapstructure:"provider"`
	OutputSchema  string               `json:"outputSchema" mapstructure:"outputSchema"`
	WebSearch     bool                 `json:"webSearch" mapstructure:"webSearch"`
	WebMaxResults *int                 `json:"webMaxResults" mapstructure:"webMaxResults"`
}

type ProviderRoutingSpec struct {
	Sort              string   `json:"sort" mapstructure:"sort"`
	Only              []string `json:"only" mapstructure:"only"`
	Ignore            []string `json:"ignore" mapstructure:"ignore"`
	AllowFallbacks    *bool    `json:"allowFallbacks" mapstructure:"allowFallbacks"`
	RequireParameters *bool    `json:"requireParameters" mapstructure:"requireParameters"`
	DataCollection    string   `json:"dataCollection" mapstructure:"dataCollection"`
}

type ChatCompletionPayload struct {
	ID           string                  `json:"id"`
	Model        string                  `json:"model"`
	Provider     string                  `json:"provider"`
	Text         string                  `json:"text"`
	Parsed       any                     `json:"parsed,omitempty"`
	Reasoning    string                  `json:"reasoning,omitempty"`
	Citations    []Citation              `json:"citations,omitempty"`
	FinishReason string                  `json:"finishReason"`
	Usage        *Usage                  `json:"usage,omitempty"`
	Response     *ChatCompletionResponse `json:"response"`
}

// Citation is a source the web plugin used, flattened for downstream nodes.
type Citation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ChatCompletionNodeMetadata is node-level metadata surfaced in the UI so the
// configured model and routing are visible without opening the node.
type ChatCompletionNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	ProviderRouting  bool   `json:"providerRouting" mapstructure:"providerRouting"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
	WebSearch        bool   `json:"webSearch" mapstructure:"webSearch"`
}

func (c *ChatCompletion) Name() string {
	return "openrouter.chatCompletion"
}

func (c *ChatCompletion) Label() string {
	return "Chat Completion"
}

func (c *ChatCompletion) Description() string {
	return "Generate a response with any model available on OpenRouter"
}

func (c *ChatCompletion) Documentation() string {
	return `The Chat Completion component sends a prompt to any of the models OpenRouter brokers and returns the generated response.

## Use Cases

- **Model comparison**: Run the same prompt against models from different vendors without changing integrations
- **Cost control**: Route to the cheapest provider serving a model, or cap which providers may serve it
- **Resilience**: Fall back to other models when the primary is rate limited or down
- **Document analysis**: Attach PDFs, images, or text files from the Files tab alongside the prompt

## Configuration

- **Model**: The model to prompt. Picked from the models OpenRouter currently lists.
- **Prompt**: The user message (supports expressions)
- **System Prompt**: (Optional) System-level instructions
- **Files**: (Optional) Files from the Files tab to attach. Text files are added to the prompt directly. Images require a vision model. PDFs are parsed by OpenRouter and work with any model, but parsing is a paid feature and the request is rejected below a minimum account balance.
- **Max Tokens**: (Optional) Upper bound on generated tokens. Reasoning models bill their reasoning against this budget, so a low value can return an empty response.
- **Temperature**: (Optional) Sampling temperature
- **Fallback Models**: (Optional) Models to try when the primary fails at runtime. Tried in order.
- **Web Search**: (Optional) Search the web before answering and cite the sources used. Billed on top of tokens, including on free models.
- **Web Search Results**: (Optional) How many results to gather when Web Search is on. More results cost more.
- **Structured Output**: (Optional) A JSON Schema for the response. The model returns JSON matching it, available on the ` + "`parsed`" + ` output. Only some models support this, so enabling it automatically restricts routing to providers that honour it.
- **Provider Routing**: (Optional) Control which upstream provider serves the request. See below.

## Provider Routing

A model ID on OpenRouter is a listing, not a server: several providers compete to serve the same model and they are not equivalent. Across the providers serving one popular model, input price can vary more than fourfold, the context ceiling can vary tenfold, and quantization ranges from fp4 to fp8 — which affects answer quality. Left alone, OpenRouter picks one and the choice can change between requests.

- **Optimize For**: Sort candidate providers by price, throughput, or latency
- **Allowed Providers** / **Excluded Providers**: Restrict routing to (or away from) specific providers. The list is narrowed to the providers actually serving the selected model.
- **Allow Fallbacks**: When off, the request fails rather than silently rerouting to a provider outside your preferences
- **Require Parameter Support**: Only route to providers that honour every parameter sent. Off by default, matching OpenRouter — a provider that does not support a parameter accepts the request and ignores it rather than returning an error.
- **Data Collection**: Set to Deny to exclude providers that may train on submitted data

## Output

Returns the completion including:
- **text**: The generated text
- **parsed**: When Structured Output is configured, the response parsed into an object
- **citations**: When Web Search is on, the sources the model cited, with url and title
- **reasoning**: The model's reasoning trace, when it returns one
- **model**: The model that served the request
- **provider**: The upstream provider that actually served it, which can differ between requests
- **finishReason**: Why generation stopped
- **usage**: Token counts and cost, including reasoning tokens
- **response**: The full API response

## Notes

- Fallback models cover runtime failures such as rate limits and provider outages. An invalid model ID still fails the request outright.
- Free model variants (IDs ending in ` + "`:free`" + `) draw from a shared upstream pool and are rate limited independently of your balance.
- Attachments are inlined into the request body rather than uploaded, so the combined size is capped at 8MB.
- Only PDFs and images are sent as attachments. Text files become part of the prompt, so they cost prompt tokens and are not subject to OpenRouter's document parsing.
- Web search is billed per request on top of tokens, so it costs money even when the model itself is free.
- Structured Output is supported by most but not all models. Because a provider that does not support it accepts the request and ignores the schema, enabling it forces Require Parameter Support on, which can make the request fail rather than silently return prose.
- Image generation is not available here. OpenRouter generates images through a separate endpoint, and the ` + "`modalities`" + ` parameter is ignored by chat completions.`
}

func (c *ChatCompletion) Icon() string {
	return "sparkles"
}

func (c *ChatCompletion) Color() string {
	return "gray"
}

func (c *ChatCompletion) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ChatCompletion) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "openai/gpt-4o-mini",
			Placeholder: "e.g. openai/gpt-4o-mini",
			Description: "Model to prompt",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeModel,
				},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the prompt text",
			Description: "The user message (supports expressions)",
		},
		{
			Name:        "systemPrompt",
			Label:       "System Prompt",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Placeholder: "Optional system-level instructions",
			Description: "System-level instructions sent ahead of the prompt",
		},
		{
			Name:        "files",
			Label:       "Files",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Files from the Files tab to attach to the prompt (images, PDFs, or text). Images require a vision model; PDFs work with any model.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File path",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeRepositoryFile,
					},
				},
			},
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Upper bound on generated tokens. Reasoning models bill reasoning against this budget, so a low value can return an empty response.",
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Sampling temperature. Providers that do not support it ignore it unless Require Parameter Support is on.",
		},
		{
			Name:        "models",
			Label:       "Fallback Models",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Models to try, in order, when the primary fails at runtime. Covers rate limits and provider outages, not an invalid model ID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeModel,
					Multi: true,
				},
			},
		},
		{
			Name:        "webSearch",
			Label:       "Web Search",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Search the web before answering and cite the sources used. Billed on top of tokens, including on free models.",
		},
		{
			Name:        "webMaxResults",
			Label:       "Web Search Results",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "How many search results to gather. More results cost more.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "webSearch", Values: []string{"true"}},
			},
		},
		structuredoutput.ConfigField(
			"outputSchema",
			"Structured Output",
			"A JSON Schema describing the response. The model is constrained to return JSON matching it (available on the `parsed` output). The schema is validated before the request and sent in strict mode; strict mode marks every property required, so express optional fields by making their type nullable. Only some models support this, so the request is automatically restricted to providers that honour it.",
		),
		{
			Name:        "provider",
			Label:       "Provider Routing",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Control which upstream provider serves the request. Providers serving the same model differ in price, context length and quantization.",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: providerRoutingSchema(),
				},
			},
		},
	}
}

func providerRoutingSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sort",
			Label:       "Optimize For",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     sortAuto,
			Description: "How to rank the providers serving this model",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: sortOptions,
				},
			},
		},
		{
			Name:        "only",
			Label:       "Allowed Providers",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Route only to these providers. Leave empty to allow any.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeProvider,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "model",
							ValueFrom: &configuration.ParameterValueFrom{Field: "model"},
						},
					},
				},
			},
		},
		{
			Name:        "ignore",
			Label:       "Excluded Providers",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Never route to these providers",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeProvider,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "model",
							ValueFrom: &configuration.ParameterValueFrom{Field: "model"},
						},
					},
				},
			},
		},
		{
			Name:        "allowFallbacks",
			Label:       "Allow Fallbacks",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "When off, the request fails instead of silently rerouting to a provider outside your preferences",
		},
		{
			Name:        "requireParameters",
			Label:       "Require Parameter Support",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Only route to providers that honour every parameter sent. Otherwise unsupported parameters are accepted and ignored.",
		},
		{
			Name:        "dataCollection",
			Label:       "Data Collection",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "allow",
			Description: "Set to Deny to exclude providers that may train on submitted data",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: dataCollectionOptions,
				},
			},
		},
	}
}

func (c *ChatCompletion) Setup(ctx core.SetupContext) error {
	spec := ChatCompletionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if err := validateRouting(spec.Provider); err != nil {
		return err
	}

	if len(spec.Files) > 0 {
		if ctx.Files == nil {
			return fmt.Errorf("files configured but file access is not available")
		}
		available, err := ctx.Files.List()
		if err != nil {
			return fmt.Errorf("failed to list repository files: %v", err)
		}
		fileSet := make(map[string]bool, len(available))
		for _, f := range available {
			if norm, err := gitprovider.NormalizePath(f); err == nil {
				fileSet[norm] = true
			}
		}
		for _, f := range spec.Files {
			norm, err := gitprovider.ValidateUserPath(f)
			if err != nil {
				return fmt.Errorf("invalid file path %q: %v", f, err)
			}
			if !fileSet[norm] {
				return fmt.Errorf("file %q not found in app repository", f)
			}
		}

		// Read the files now so unsupported types, empty files, and the inline
		// size limit are caught at config time rather than on every execution.
		atts, err := attachments.Read(ctx.Files, spec.Files)
		if err != nil {
			return err
		}
		if err := checkAttachmentSize(atts); err != nil {
			return err
		}
	}

	// The schema supports expressions (like the prompt), which resolve only at
	// execution. Validate it as JSON when it has no unresolved expression;
	// Execute re-parses the resolved value.
	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
	if hasSchema && !strings.Contains(spec.OutputSchema, "{{") {
		if _, err := structuredoutput.Parse(spec.OutputSchema); err != nil {
			return err
		}
	}

	if ctx.Metadata != nil {
		_ = ctx.Metadata.Set(ChatCompletionNodeMetadata{
			Model:            spec.Model,
			ProviderRouting:  spec.Provider != nil,
			StructuredOutput: hasSchema,
			WebSearch:        spec.WebSearch,
		})
	}

	return nil
}

// validateRouting rejects routing combinations OpenRouter cannot satisfy before
// they cost a request.
func validateRouting(routing *ProviderRoutingSpec) error {
	if routing == nil {
		return nil
	}

	switch routing.Sort {
	case "", sortAuto, "price", "throughput", "latency":
	default:
		return fmt.Errorf("invalid sort: %s", routing.Sort)
	}

	switch routing.DataCollection {
	case "", "allow", "deny":
	default:
		return fmt.Errorf("invalid data collection policy: %s", routing.DataCollection)
	}

	ignored := make(map[string]bool, len(routing.Ignore))
	for _, provider := range routing.Ignore {
		ignored[provider] = true
	}
	for _, provider := range routing.Only {
		if ignored[provider] {
			return fmt.Errorf("provider %q is both allowed and excluded", provider)
		}
	}

	return nil
}

func checkAttachmentSize(atts []attachments.Attachment) error {
	total := 0
	for _, att := range atts {
		total += len(att.Data)
	}
	if total > maxAttachmentBytes {
		return fmt.Errorf("attachments total %d bytes, which exceeds the %d byte limit for inlined files", total, maxAttachmentBytes)
	}
	return nil
}

func (c *ChatCompletion) Execute(ctx core.ExecutionContext) error {
	spec := ChatCompletionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if err := validateRouting(spec.Provider); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	atts, err := attachments.Read(ctx.Files, spec.Files)
	if err != nil {
		return fmt.Errorf("failed to read attachments: %v", err)
	}
	if err := checkAttachmentSize(atts); err != nil {
		return err
	}

	schema, err := structuredoutput.Parse(spec.OutputSchema)
	if err != nil {
		return err
	}

	req := ChatCompletionRequest{
		Messages:    buildMessages(spec.SystemPrompt, spec.Prompt, atts),
		MaxTokens:   spec.MaxTokens,
		Temperature: spec.Temperature,
		Provider:    buildRouting(spec.Provider),
		Plugins:     buildPlugins(spec),
	}

	if schema != nil {
		req.ResponseFormat = &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "structured_output",
				Strict: true,
				Schema: structuredoutput.Prepare(schema, true),
			},
		}

		// Providers that do not support response_format accept the request and
		// ignore it, returning prose where the workflow expects JSON. Pinning
		// routing to providers that honour every parameter is what makes the
		// schema binding actually hold.
		req.Provider = requireParameters(req.Provider)
	}

	// A fallback chain replaces the single model field: OpenRouter takes the
	// first entry as the primary and tries the rest in order.
	if chain := modelChain(spec.Model, spec.Models); len(chain) > 1 {
		req.Models = chain
	} else {
		req.Model = spec.Model
	}

	ctx.Logger.Infof("Running OpenRouter chat completion with model %s", spec.Model)

	response, err := client.CreateChatCompletion(req)
	if err != nil {
		return err
	}

	payload, err := buildPayload(response)
	if err != nil {
		return err
	}

	// A refusal arrives as prose rather than schema-shaped JSON, so only parse
	// when the model actually answered.
	if schema != nil && response.Choices[0].Message.Refusal == "" && payload.Text != "" {
		var parsed any
		if err := json.Unmarshal([]byte(payload.Text), &parsed); err == nil {
			payload.Parsed = parsed
		}
	}

	ctx.Logger.Infof("OpenRouter chat completion served by %s", response.Provider)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ChatCompletionPayloadType,
		[]any{payload},
	)
}

// modelChain puts the primary model first and appends the configured fallbacks,
// dropping duplicates so the primary is not retried as its own fallback.
func modelChain(model string, fallbacks []string) []string {
	chain := []string{model}
	seen := map[string]bool{model: true}

	for _, fallback := range fallbacks {
		if fallback == "" || seen[fallback] {
			continue
		}
		seen[fallback] = true
		chain = append(chain, fallback)
	}

	return chain
}

// buildPlugins enables the web plugin when web search is on.
func buildPlugins(spec ChatCompletionSpec) []Plugin {
	if !spec.WebSearch {
		return nil
	}

	return []Plugin{{ID: "web", MaxResults: spec.WebMaxResults}}
}

// requireParameters turns on provider parameter matching, creating the routing
// block when the user configured none.
func requireParameters(routing *ProviderRouting) *ProviderRouting {
	required := true
	if routing == nil {
		return &ProviderRouting{RequireParameters: &required}
	}

	if routing.RequireParameters == nil {
		routing.RequireParameters = &required
	}
	return routing
}

// citations flattens the web plugin's annotations for downstream nodes.
func citations(message ChoiceMessage) []Citation {
	var out []Citation
	for _, annotation := range message.Annotations {
		if annotation.Type != "url_citation" || annotation.URLCitation == nil {
			continue
		}
		out = append(out, Citation{
			URL:   annotation.URLCitation.URL,
			Title: annotation.URLCitation.Title,
		})
	}
	return out
}

// buildRouting maps the configured routing onto the API's provider object.
func buildRouting(spec *ProviderRoutingSpec) *ProviderRouting {
	if spec == nil {
		return nil
	}

	sort := spec.Sort
	if sort == sortAuto {
		sort = ""
	}

	return &ProviderRouting{
		Sort:              sort,
		Only:              spec.Only,
		Ignore:            spec.Ignore,
		AllowFallbacks:    spec.AllowFallbacks,
		RequireParameters: spec.RequireParameters,
		DataCollection:    spec.DataCollection,
	}
}

// buildMessages assembles the chat messages. Attachments are inlined into the
// user message as base64 data URLs, since OpenRouter has no Files API.
func buildMessages(systemPrompt, prompt string, atts []attachments.Attachment) []Message {
	messages := make([]Message, 0, 2)

	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}

	if len(atts) == 0 {
		return append(messages, Message{Role: "user", Content: prompt})
	}

	parts := make([]ContentPart, 0, len(atts)+1)
	parts = append(parts, ContentPart{Type: "text", Text: prompt})
	for _, att := range atts {
		if att.IsImage() {
			parts = append(parts, ContentPart{Type: "image_url", ImageURL: &ImageURL{URL: dataURL(att)}})
			continue
		}

		if att.IsPDF() {
			parts = append(parts, ContentPart{
				Type: "file",
				File: &FilePart{Filename: att.Name, FileData: dataURL(att)},
			})
			continue
		}

		// Text goes in as prompt text rather than a file part. File parts run
		// through OpenRouter's document parser, which is a paid feature that
		// rejects the request below a minimum balance, and text needs no parsing.
		parts = append(parts, ContentPart{
			Type: "text",
			Text: fmt.Sprintf("--- %s ---\n%s", att.Name, att.Data),
		})
	}

	return append(messages, Message{Role: "user", Content: parts})
}

// dataURL inlines an attachment, since OpenRouter has no Files API to upload to.
func dataURL(att attachments.Attachment) string {
	return "data:" + att.UploadMIME() + ";base64," + base64.StdEncoding.EncodeToString(att.Data)
}

// buildPayload flattens the first choice into the node output. Content is null
// rather than empty when reasoning tokens consumed the whole token budget, so
// the reasoning trace stands in for the text when there is one, and a run that
// produced neither fails instead of emitting a blank result.
func buildPayload(response *ChatCompletionResponse) (*ChatCompletionPayload, error) {
	if response == nil || len(response.Choices) == 0 {
		return nil, fmt.Errorf("chat completion returned no choices")
	}

	choice := response.Choices[0]

	text := ""
	if choice.Message.Content != nil {
		text = *choice.Message.Content
	}

	if text == "" {
		switch {
		case choice.Message.Refusal != "":
			text = choice.Message.Refusal
		case choice.Message.Reasoning != "":
			text = choice.Message.Reasoning
		case choice.FinishReason == "length":
			return nil, fmt.Errorf("model returned no content: the token budget was exhausted before any output was produced, which reasoning models do when Max Tokens is too low")
		}
	}

	return &ChatCompletionPayload{
		ID:           response.ID,
		Model:        response.Model,
		Provider:     response.Provider,
		Text:         text,
		Reasoning:    choice.Message.Reasoning,
		Citations:    citations(choice.Message),
		FinishReason: choice.FinishReason,
		Usage:        response.Usage,
		Response:     response,
	}, nil
}

func (c *ChatCompletion) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ChatCompletion) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ChatCompletion) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ChatCompletion) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ChatCompletion) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
