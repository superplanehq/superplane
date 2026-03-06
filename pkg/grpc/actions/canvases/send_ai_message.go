package canvases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

const (
	canvasAIOpenAIModel                  = "gpt-5.3-codex"
	canvasAIAssociatedEncryptionDataName = "agent_mode_openai_api_key"
	canvasSystemSkillPath                = "templates/skills/system.md"
	canvasComponentSkillsDir             = "templates/skills/components"
	componentSkillMaxCharsPerBlock       = 3000
	componentSkillMaxCharsTotal          = 14000
	componentSkillMissingPreviewLimit    = 10
	canvasContextRequestLimit            = 8
	canvasContextRequestNodeEventsLimit  = 5
	canvasContextRequestNodeEventsTotal  = 20
	canvasContextEnrichmentRounds        = 1
	implicitRepoNodeConfigRequestLimit   = 12
)

type openAIResponsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIResponsesResult struct {
	OutputText string                  `json:"output_text"`
	Output     []openAIResponsesOutput `json:"output"`
}

type openAIResponsesOutput struct {
	Content []openAIResponsesOutputText `json:"content"`
}

type openAIResponsesOutputText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAICanvasPlan struct {
	AssistantMessage string                   `json:"assistantMessage"`
	Operations       []map[string]interface{} `json:"operations"`
	ContextRequests  []openAICanvasContextReq `json:"contextRequests,omitempty"`
}

type openAICanvasContextReq struct {
	Type      string `json:"type"`
	NodeID    string `json:"nodeId,omitempty"`
	BlockName string `json:"blockName,omitempty"`
	BlockType string `json:"blockType,omitempty"`
	MaxItems  int    `json:"maxItems,omitempty"`
}

type canvasSkillPromptContext struct {
	PromptSection string
	AppliedBlocks []string
	MissingBlocks []string
}

func SendAiMessage(
	ctx context.Context,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	organizationID string,
	req *pb.SendAiMessageRequest,
) (*pb.SendAiMessageResponse, error) {
	canvasID, err := uuid.Parse(req.GetCanvasId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	if _, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID); err != nil {
		return nil, status.Error(codes.NotFound, "canvas not found")
	}

	prompt := strings.TrimSpace(req.GetPrompt())
	if prompt == "" {
		return nil, status.Error(codes.InvalidArgument, "prompt is required")
	}

	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(organizationID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "agent mode settings not found")
	}
	if !settings.AgentModeEnabled || len(settings.OpenAIApiKeyCiphertext) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "agent mode is not configured for this organization")
	}

	apiKeyBytes, err := encryptor.Decrypt(ctx, settings.OpenAIApiKeyCiphertext, []byte(canvasAIAssociatedEncryptionDataName))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decrypt organization ai key")
	}

	plan, err := generateCanvasAIPlan(ctx, registry, string(apiKeyBytes), req)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to generate ai response")
	}

	operations := make([]*structpb.Struct, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		operationStruct, structErr := structpb.NewStruct(operation)
		if structErr != nil {
			continue
		}
		operations = append(operations, operationStruct)
	}

	return &pb.SendAiMessageResponse{
		AssistantMessage: plan.AssistantMessage,
		Operations:       operations,
	}, nil
}

func generateCanvasAIPlan(
	ctx context.Context,
	registry *registry.Registry,
	apiKey string,
	req *pb.SendAiMessageRequest,
) (*openAICanvasPlan, error) {
	canvasContextJSON, err := buildCanvasContextJSON(req.GetCanvasContext())
	if err != nil {
		return nil, err
	}
	skillContext := buildCanvasSkillPromptContext(req.GetCanvasContext())
	systemPrompt, err := loadCanvasSystemPrompt()
	if err != nil {
		return nil, err
	}

	firstPrompt := buildCanvasPlannerPrompt(
		systemPrompt,
		string(canvasContextJSON),
		skillContext.PromptSection,
		"",
		req.GetPrompt(),
		true,
	)

	parsedPlan, err := requestCanvasAIPlan(ctx, registry, apiKey, firstPrompt)
	if err != nil {
		return nil, err
	}
	if parsedPlan == nil {
		return &openAICanvasPlan{
			AssistantMessage: "I couldn't produce executable operations for that request. Please rephrase with specific component names or desired flow.",
			Operations:       []map[string]interface{}{},
		}, nil
	}

	rounds := 0
	for rounds < canvasContextEnrichmentRounds {
		contextRequests := sanitizeCanvasContextRequests(parsedPlan.ContextRequests, req.GetCanvasContext())
		if len(contextRequests) == 0 {
			contextRequests = deriveImplicitContextRequests(parsedPlan, req.GetCanvasContext())
		}
		if len(contextRequests) == 0 {
			break
		}

		additionalContext, contextErr := buildRequestedCanvasContextData(
			registry,
			req.GetCanvasId(),
			req.GetCanvasContext(),
			contextRequests,
		)
		if contextErr != nil || strings.TrimSpace(additionalContext) == "" {
			break
		}

		rounds++
		nextPrompt := buildCanvasPlannerPrompt(
			systemPrompt,
			string(canvasContextJSON),
			skillContext.PromptSection,
			additionalContext,
			req.GetPrompt(),
			false,
		)

		nextPlan, nextErr := requestCanvasAIPlan(ctx, registry, apiKey, nextPrompt)
		if nextErr != nil || nextPlan == nil {
			break
		}

		parsedPlan = nextPlan
	}

	if parsedPlan.AssistantMessage == "" {
		parsedPlan.AssistantMessage = "I prepared a draft change set you can review and apply."
	}
	parsedPlan.AssistantMessage = sanitizeAssistantMessage(parsedPlan.AssistantMessage)
	if parsedPlan.Operations == nil {
		parsedPlan.Operations = []map[string]interface{}{}
	}

	var hasFilteredInvalidAddNode bool
	parsedPlan.Operations, hasFilteredInvalidAddNode = sanitizeCanvasOperations(parsedPlan.Operations, req.GetCanvasContext())
	if hasFilteredInvalidAddNode && len(parsedPlan.Operations) == 0 {
		parsedPlan.AssistantMessage = "I couldn't find one or more requested components in this workspace. Please tell me which existing component should be used instead."
	}

	return parsedPlan, nil
}

func buildCanvasPlannerPrompt(
	systemPrompt string,
	canvasContextJSON string,
	skillPromptSection string,
	additionalContextJSON string,
	userPrompt string,
	allowContextRequests bool,
) string {
	sections := []string{
		systemPrompt,
		"",
		"Current canvas context JSON:",
		canvasContextJSON,
		"",
		skillPromptSection,
	}

	if allowContextRequests {
		sections = append(sections,
			"",
			"If required context is missing, you may ask for additional context by returning strict JSON with:",
			`{"assistantMessage":"short reason","operations":[],"contextRequests":[{"type":"node_recent_outputs","nodeId":"existing-node-id","maxItems":3},{"type":"node_configuration","nodeId":"existing-node-id"},{"type":"block_schema","blockName":"component.or.trigger.name","blockType":"component|trigger"},{"type":"block_example_output","blockName":"component.or.trigger.name","blockType":"component|trigger"},{"type":"component_skill","blockName":"component.or.trigger.name","blockType":"component|trigger"}]}`,
			"Always inspect both relevant existing node configuration and block schema before proposing operations that depend on node behavior, channels, or outputs.",
			"If output field names are needed and recent node outputs are unavailable, request block_example_output before asking the user.",
			"Never ask the user to provide schema or output-channel details that can be fetched through contextRequests; fetch them first.",
			"If required configuration/schema/example output is missing, request the needed context first and return operations as [].",
			"Only request context when necessary. Use existing node IDs from canvas context. Keep requests minimal.",
		)
	} else {
		sections = append(sections,
			"",
			"Additional context has already been provided. Return final operations now and do not return contextRequests.",
		)
	}

	if strings.TrimSpace(additionalContextJSON) != "" {
		sections = append(sections,
			"",
			"Additional retrieved context JSON:",
			additionalContextJSON,
		)
	}

	sections = append(sections,
		"",
		"User request:",
		userPrompt,
	)

	return strings.Join(sections, "\n")
}

func requestCanvasAIPlan(
	ctx context.Context,
	registry *registry.Registry,
	apiKey string,
	prompt string,
) (*openAICanvasPlan, error) {
	body, err := json.Marshal(openAIResponsesRequest{
		Model: canvasAIOpenAIModel,
		Input: prompt,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("authorization", "Bearer "+apiKey)

	httpRes, err := registry.HTTPContext().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpRes.Body.Close()

	responseBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, err
	}
	if httpRes.StatusCode < http.StatusOK || httpRes.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openai request failed with status %d", httpRes.StatusCode)
	}

	var openAIRes openAIResponsesResult
	if err := json.Unmarshal(responseBody, &openAIRes); err != nil {
		return nil, err
	}

	rawPlanText := openAIRes.OutputText
	if strings.TrimSpace(rawPlanText) == "" {
		rawPlanText = extractOpenAIText(openAIRes)
	}

	parsedPlan, parseErr := parseOpenAICanvasPlan(rawPlanText)
	if parseErr != nil {
		return nil, nil
	}

	return parsedPlan, nil
}

func sanitizeCanvasContextRequests(
	requests []openAICanvasContextReq,
	canvasContext *pb.CanvasAiContext,
) []openAICanvasContextReq {
	if len(requests) == 0 {
		return []openAICanvasContextReq{}
	}

	allowedNodeIDs := map[string]struct{}{}
	if canvasContext != nil {
		for _, node := range canvasContext.GetNodes() {
			id := strings.TrimSpace(node.GetId())
			if id == "" {
				continue
			}
			allowedNodeIDs[id] = struct{}{}
		}
	}

	result := make([]openAICanvasContextReq, 0, len(requests))
	seen := map[string]struct{}{}
	for _, req := range requests {
		if len(result) >= canvasContextRequestLimit {
			break
		}

		reqType := strings.TrimSpace(strings.ToLower(req.Type))
		switch reqType {
		case "node_recent_outputs", "node_configuration":
			nodeID := strings.TrimSpace(req.NodeID)
			if nodeID == "" {
				continue
			}
			if _, ok := allowedNodeIDs[nodeID]; !ok {
				continue
			}

			maxItems := req.MaxItems
			if maxItems <= 0 {
				maxItems = 3
			}
			if maxItems > canvasContextRequestNodeEventsLimit {
				maxItems = canvasContextRequestNodeEventsLimit
			}

			key := reqType + ":" + nodeID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, openAICanvasContextReq{
				Type:     reqType,
				NodeID:   nodeID,
				MaxItems: maxItems,
			})
		case "component_skill":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}
			blockType := strings.TrimSpace(strings.ToLower(req.BlockType))
			if blockType != "component" && blockType != "trigger" {
				blockType = ""
			}

			key := reqType + ":" + blockType + ":" + blockName
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, openAICanvasContextReq{
				Type:      reqType,
				BlockName: blockName,
				BlockType: blockType,
			})
		case "block_schema":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}
			blockType := strings.TrimSpace(strings.ToLower(req.BlockType))
			if blockType != "component" && blockType != "trigger" {
				blockType = ""
			}

			key := reqType + ":" + blockType + ":" + blockName
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, openAICanvasContextReq{
				Type:      reqType,
				BlockName: blockName,
				BlockType: blockType,
			})
		case "block_example_output":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}
			blockType := strings.TrimSpace(strings.ToLower(req.BlockType))
			if blockType != "component" && blockType != "trigger" {
				blockType = ""
			}

			key := reqType + ":" + blockType + ":" + blockName
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, openAICanvasContextReq{
				Type:      reqType,
				BlockName: blockName,
				BlockType: blockType,
			})
		}
	}

	return result
}

func deriveImplicitContextRequests(
	plan *openAICanvasPlan,
	canvasContext *pb.CanvasAiContext,
) []openAICanvasContextReq {
	if plan == nil {
		return []openAICanvasContextReq{}
	}
	if len(plan.ContextRequests) > 0 || len(plan.Operations) > 0 {
		return []openAICanvasContextReq{}
	}

	message := strings.TrimSpace(plan.AssistantMessage)
	if message == "" {
		return []openAICanvasContextReq{}
	}

	lower := strings.ToLower(message)
	needsSchema := strings.Contains(lower, "schema") ||
		strings.Contains(lower, "output channel") ||
		strings.Contains(lower, "output-channel") ||
		strings.Contains(lower, "outputchannel") ||
		strings.Contains(lower, "channel metadata")
	if !needsSchema {
		return []openAICanvasContextReq{}
	}

	type blockRef struct {
		name    string
		kind    string
		aliases []string
	}
	allowedBlocks := make([]blockRef, 0)
	seenBlockNames := map[string]struct{}{}
	if canvasContext != nil {
		for _, block := range canvasContext.GetAvailableBlocks() {
			name := strings.TrimSpace(block.GetName())
			if name == "" {
				continue
			}
			if _, exists := seenBlockNames[name]; exists {
				continue
			}
			seenBlockNames[name] = struct{}{}

			kind := strings.TrimSpace(strings.ToLower(block.GetType()))
			aliases := []string{name}
			label := strings.TrimSpace(block.GetLabel())
			if label != "" && label != name {
				aliases = append(aliases, label)
			}
			allowedBlocks = append(allowedBlocks, blockRef{
				name:    name,
				kind:    kind,
				aliases: aliases,
			})
		}
	}
	if len(allowedBlocks) == 0 {
		return []openAICanvasContextReq{}
	}

	normalizedMessage := normalizeSchemaMessageForBlockMatching(message)

	reqs := make([]openAICanvasContextReq, 0)
	seen := map[string]struct{}{}
	for _, block := range allowedBlocks {
		if !messageMentionsAnyBlockAlias(normalizedMessage, block.aliases) {
			continue
		}

		key := block.kind + ":" + block.name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		reqs = append(reqs,
			openAICanvasContextReq{
				Type:      "block_schema",
				BlockName: block.name,
				BlockType: block.kind,
			},
			openAICanvasContextReq{
				Type:      "block_example_output",
				BlockName: block.name,
				BlockType: block.kind,
			},
		)
	}

	// If assistant asks user for repository while canvas likely already has it, auto-fetch node configurations.
	if asksForRepositoryInMessage(lower) && canvasContext != nil {
		for _, node := range canvasContext.GetNodes() {
			if len(reqs) >= implicitRepoNodeConfigRequestLimit {
				break
			}
			nodeID := strings.TrimSpace(node.GetId())
			if nodeID == "" {
				continue
			}
			key := "node_configuration:" + nodeID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			reqs = append(reqs, openAICanvasContextReq{
				Type:   "node_configuration",
				NodeID: nodeID,
			})
		}
	}

	return sanitizeCanvasContextRequests(reqs, canvasContext)
}

func asksForRepositoryInMessage(lowerMessage string) bool {
	message := strings.ToLower(strings.TrimSpace(lowerMessage))
	if message == "" {
		return false
	}

	if !strings.Contains(message, "repository") {
		return false
	}

	if strings.Contains(message, "github repository") && (strings.Contains(message, "which") || strings.Contains(message, "what")) {
		return true
	}

	if strings.Contains(message, "which repository") || strings.Contains(message, "what repository") {
		return true
	}

	return false
}

func normalizeSchemaMessageForBlockMatching(message string) string {
	normalized := strings.ToLower(message)
	normalized = strings.ReplaceAll(normalized, "`", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	return normalized
}

func messageMentionsAnyBlockAlias(normalizedMessage string, aliases []string) bool {
	for _, alias := range aliases {
		canonical := strings.ToLower(strings.TrimSpace(alias))
		if canonical == "" {
			continue
		}
		canonical = strings.ReplaceAll(canonical, " ", "")
		canonical = strings.ReplaceAll(canonical, "-", "")
		if strings.Contains(normalizedMessage, canonical) {
			return true
		}

		withoutDots := strings.ReplaceAll(canonical, ".", "")
		if withoutDots != canonical && strings.Contains(normalizedMessage, withoutDots) {
			return true
		}
	}
	return false
}

func buildRequestedCanvasContextData(
	registry *registry.Registry,
	canvasID string,
	canvasContext *pb.CanvasAiContext,
	requests []openAICanvasContextReq,
) (string, error) {
	if len(requests) == 0 {
		return "", nil
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return "", fmt.Errorf("invalid canvas id")
	}

	allowedNodeIDs := map[string]struct{}{}
	if canvasContext != nil {
		for _, node := range canvasContext.GetNodes() {
			id := strings.TrimSpace(node.GetId())
			if id == "" {
				continue
			}
			allowedNodeIDs[id] = struct{}{}
		}
	}

	nodeRecentOutputs := map[string][]map[string]any{}
	nodeConfigurations := map[string]map[string]any{}
	blockSchemas := map[string]map[string]any{}
	blockExampleOutputs := map[string]map[string]any{}
	componentSkills := map[string]map[string]string{}
	errorMessages := []string{}
	totalNodeEvents := 0

	for _, req := range requests {
		switch req.Type {
		case "node_recent_outputs":
			nodeID := strings.TrimSpace(req.NodeID)
			if nodeID == "" {
				continue
			}
			if _, ok := allowedNodeIDs[nodeID]; !ok {
				continue
			}

			if totalNodeEvents >= canvasContextRequestNodeEventsTotal {
				continue
			}

			remaining := canvasContextRequestNodeEventsTotal - totalNodeEvents
			limit := req.MaxItems
			if limit <= 0 {
				limit = 3
			}
			if limit > canvasContextRequestNodeEventsLimit {
				limit = canvasContextRequestNodeEventsLimit
			}
			if limit > remaining {
				limit = remaining
			}
			if limit <= 0 {
				continue
			}

			events, eventsErr := models.ListCanvasEvents(canvasUUID, nodeID, limit, nil)
			if eventsErr != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("node_recent_outputs(%s): %s", nodeID, eventsErr.Error()))
				continue
			}

			samples := make([]map[string]any, 0, len(events))
			for _, event := range events {
				row := map[string]any{
					"channel": event.Channel,
					"data":    event.Data.Data(),
				}
				if event.CreatedAt != nil {
					row["createdAt"] = event.CreatedAt.UTC().Format(time.RFC3339)
				}
				samples = append(samples, row)
			}
			totalNodeEvents += len(samples)
			nodeRecentOutputs[nodeID] = samples
		case "node_configuration":
			nodeID := strings.TrimSpace(req.NodeID)
			if nodeID == "" {
				continue
			}
			if _, ok := allowedNodeIDs[nodeID]; !ok {
				continue
			}

			node, nodeErr := models.FindCanvasNode(database.Conn(), canvasUUID, nodeID)
			if nodeErr != nil {
				if !errors.Is(nodeErr, gorm.ErrRecordNotFound) {
					errorMessages = append(errorMessages, fmt.Sprintf("node_configuration(%s): %s", nodeID, nodeErr.Error()))
				}
				continue
			}

			blockName := ""
			ref := node.Ref.Data()
			switch node.Type {
			case models.NodeTypeComponent:
				if ref.Component != nil {
					blockName = ref.Component.Name
				}
			case models.NodeTypeTrigger:
				if ref.Trigger != nil {
					blockName = ref.Trigger.Name
				}
			}

			nodeConfigurations[nodeID] = map[string]any{
				"name":          node.Name,
				"type":          node.Type,
				"blockName":     blockName,
				"configuration": node.Configuration.Data(),
				"metadata":      node.Metadata.Data(),
			}
			if blockName != "" {
				if _, exists := blockSchemas[blockName]; !exists {
					schemaType := ""
					if node.Type == models.NodeTypeComponent {
						schemaType = "component"
					} else if node.Type == models.NodeTypeTrigger {
						schemaType = "trigger"
					}
					schemaData, schemaErr := loadBlockSchemaFromRegistry(registry, blockName, schemaType)
					if schemaErr != nil {
						errorMessages = append(errorMessages, fmt.Sprintf("block_schema(%s): %s", blockName, schemaErr.Error()))
					} else if schemaData != nil {
						blockSchemas[blockName] = schemaData
					}
				}
				if _, exists := blockExampleOutputs[blockName]; !exists {
					exampleOutput, exampleErr := loadBlockExampleOutputFromRegistry(registry, blockName, "")
					if exampleErr != nil {
						errorMessages = append(errorMessages, fmt.Sprintf("block_example_output(%s): %s", blockName, exampleErr.Error()))
					} else if exampleOutput != nil {
						blockExampleOutputs[blockName] = exampleOutput
					}
				}
			}
		case "block_schema":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}
			if _, exists := blockSchemas[blockName]; exists {
				continue
			}

			schemaData, schemaErr := loadBlockSchemaFromRegistry(registry, blockName, strings.TrimSpace(req.BlockType))
			if schemaErr != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("block_schema(%s): %s", blockName, schemaErr.Error()))
				continue
			}
			if schemaData != nil {
				blockSchemas[blockName] = schemaData
			}
		case "block_example_output":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}
			if _, exists := blockExampleOutputs[blockName]; exists {
				continue
			}

			exampleOutput, exampleErr := loadBlockExampleOutputFromRegistry(registry, blockName, strings.TrimSpace(req.BlockType))
			if exampleErr != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("block_example_output(%s): %s", blockName, exampleErr.Error()))
				continue
			}
			if exampleOutput != nil {
				blockExampleOutputs[blockName] = exampleOutput
			}
		case "component_skill":
			blockName := strings.TrimSpace(req.BlockName)
			if blockName == "" {
				continue
			}

			skillContent, sourcePath, ok := loadComponentSkillContent(blockName, strings.TrimSpace(req.BlockType))
			if !ok {
				errorMessages = append(errorMessages, fmt.Sprintf("component_skill(%s): not found", blockName))
				continue
			}

			componentSkills[blockName] = map[string]string{
				"sourcePath": sourcePath,
				"content":    skillContent,
			}
		}
	}

	payload := map[string]any{
		"nodeRecentOutputs":   nodeRecentOutputs,
		"nodeConfigurations":  nodeConfigurations,
		"blockSchemas":        blockSchemas,
		"blockExampleOutputs": blockExampleOutputs,
		"componentSkills":     componentSkills,
	}
	if len(errorMessages) > 0 {
		payload["errors"] = errorMessages
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func loadBlockSchemaFromRegistry(
	registry *registry.Registry,
	blockName string,
	blockType string,
) (map[string]any, error) {
	name := strings.TrimSpace(blockName)
	if name == "" {
		return nil, fmt.Errorf("block name is required")
	}

	kind := strings.TrimSpace(strings.ToLower(blockType))
	switch kind {
	case "component":
		component, err := registry.GetComponent(name)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":           "component",
			"name":           component.Name(),
			"configuration":  component.Configuration(),
			"outputChannels": component.OutputChannels(nil),
		}, nil
	case "trigger":
		trigger, err := registry.GetTrigger(name)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":          "trigger",
			"name":          trigger.Name(),
			"configuration": trigger.Configuration(),
		}, nil
	default:
		component, componentErr := registry.GetComponent(name)
		if componentErr == nil {
			return map[string]any{
				"type":           "component",
				"name":           component.Name(),
				"configuration":  component.Configuration(),
				"outputChannels": component.OutputChannels(nil),
			}, nil
		}

		trigger, triggerErr := registry.GetTrigger(name)
		if triggerErr == nil {
			return map[string]any{
				"type":          "trigger",
				"name":          trigger.Name(),
				"configuration": trigger.Configuration(),
			}, nil
		}

		return nil, fmt.Errorf("block %s not found as component or trigger", name)
	}
}

func loadBlockExampleOutputFromRegistry(
	registry *registry.Registry,
	blockName string,
	blockType string,
) (map[string]any, error) {
	name := strings.TrimSpace(blockName)
	if name == "" {
		return nil, fmt.Errorf("block name is required")
	}

	kind := strings.TrimSpace(strings.ToLower(blockType))
	switch kind {
	case "component":
		component, err := registry.GetComponent(name)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":          "component",
			"name":          component.Name(),
			"exampleOutput": component.ExampleOutput(),
		}, nil
	case "trigger":
		trigger, err := registry.GetTrigger(name)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"type":          "trigger",
			"name":          trigger.Name(),
			"exampleOutput": trigger.ExampleData(),
		}, nil
	default:
		component, componentErr := registry.GetComponent(name)
		if componentErr == nil {
			return map[string]any{
				"type":          "component",
				"name":          component.Name(),
				"exampleOutput": component.ExampleOutput(),
			}, nil
		}

		trigger, triggerErr := registry.GetTrigger(name)
		if triggerErr == nil {
			return map[string]any{
				"type":          "trigger",
				"name":          trigger.Name(),
				"exampleOutput": trigger.ExampleData(),
			}, nil
		}

		return nil, fmt.Errorf("block %s not found as component or trigger", name)
	}
}

func loadCanvasSystemPrompt() (string, error) {
	content, err := os.ReadFile(canvasSystemSkillPath)
	if err != nil {
		return "", fmt.Errorf("failed to read canvas system prompt from %s: %w", canvasSystemSkillPath, err)
	}

	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return "", fmt.Errorf("canvas system prompt file %s is empty", canvasSystemSkillPath)
	}

	return trimmed, nil
}

func sanitizeCanvasOperations(
	operations []map[string]interface{},
	canvasContext *pb.CanvasAiContext,
) ([]map[string]interface{}, bool) {
	allowedBlocks := map[string]struct{}{}
	if canvasContext != nil {
		for _, block := range canvasContext.GetAvailableBlocks() {
			name := strings.TrimSpace(block.GetName())
			if name == "" {
				continue
			}
			allowedBlocks[name] = struct{}{}
		}
	}

	filtered := make([]map[string]interface{}, 0, len(operations))
	filteredInvalidAddNode := false

	for _, operation := range operations {
		opType, _ := operation["type"].(string)
		if opType != "add_node" {
			filtered = append(filtered, operation)
			continue
		}

		blockName, _ := operation["blockName"].(string)
		if _, ok := allowedBlocks[blockName]; !ok {
			filteredInvalidAddNode = true
			continue
		}

		filtered = append(filtered, operation)
	}

	return filtered, filteredInvalidAddNode
}

func buildCanvasContextJSON(ctx *pb.CanvasAiContext) ([]byte, error) {
	if ctx == nil {
		return json.Marshal(map[string]interface{}{
			"nodes":           []map[string]string{},
			"availableBlocks": []map[string]string{},
		})
	}

	nodes := make([]map[string]string, 0, len(ctx.GetNodes()))
	for _, node := range ctx.GetNodes() {
		nodes = append(nodes, map[string]string{
			"id":    node.GetId(),
			"name":  node.GetName(),
			"label": node.GetLabel(),
			"type":  node.GetType(),
		})
	}

	blocks := make([]map[string]string, 0, len(ctx.GetAvailableBlocks()))
	for _, block := range ctx.GetAvailableBlocks() {
		blocks = append(blocks, map[string]string{
			"name":  block.GetName(),
			"label": block.GetLabel(),
			"type":  block.GetType(),
		})
	}

	return json.Marshal(map[string]interface{}{
		"nodes":           nodes,
		"availableBlocks": blocks,
	})
}

func parseOpenAICanvasPlan(raw string) (*openAICanvasPlan, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	plan := &openAICanvasPlan{}
	if err := json.Unmarshal([]byte(cleaned), plan); err == nil {
		return plan, nil
	}

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		candidate := cleaned[start : end+1]
		if err := json.Unmarshal([]byte(candidate), plan); err == nil {
			return plan, nil
		}
	}

	return nil, fmt.Errorf("invalid ai plan json")
}

func extractOpenAIText(result openAIResponsesResult) string {
	texts := make([]string, 0)
	for _, output := range result.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) == "" {
				continue
			}
			if content.Type == "" || content.Type == "output_text" || content.Type == "text" {
				texts = append(texts, content.Text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func buildCanvasSkillPromptContext(canvasContext *pb.CanvasAiContext) canvasSkillPromptContext {
	if canvasContext == nil {
		return canvasSkillPromptContext{
			PromptSection: "Component skill guidance:\n- None (no canvas context provided).",
		}
	}

	type blockRef struct {
		name string
		kind string
	}

	seen := map[string]struct{}{}
	blocks := make([]blockRef, 0, len(canvasContext.GetAvailableBlocks()))
	for _, block := range canvasContext.GetAvailableBlocks() {
		name := strings.TrimSpace(block.GetName())
		kind := strings.TrimSpace(block.GetType())
		if name == "" || (kind != "component" && kind != "trigger") {
			continue
		}

		key := kind + ":" + name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		blocks = append(blocks, blockRef{name: name, kind: kind})
	}

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].kind == blocks[j].kind {
			return blocks[i].name < blocks[j].name
		}
		return blocks[i].kind < blocks[j].kind
	})

	sectionLines := []string{"Component skill guidance:"}
	applied := make([]string, 0, len(blocks))
	missing := make([]string, 0)
	totalChars := 0

	for _, block := range blocks {
		content, sourcePath, ok := loadComponentSkillContent(block.name, block.kind)
		if !ok {
			missing = append(missing, fmt.Sprintf("%s (%s)", block.name, block.kind))
			continue
		}

		remaining := componentSkillMaxCharsTotal - totalChars
		if remaining <= 0 {
			break
		}

		content = strings.TrimSpace(content)
		if content == "" {
			missing = append(missing, fmt.Sprintf("%s (%s)", block.name, block.kind))
			continue
		}

		if len(content) > componentSkillMaxCharsPerBlock {
			content = content[:componentSkillMaxCharsPerBlock]
		}
		if len(content) > remaining {
			content = content[:remaining]
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		sectionLines = append(
			sectionLines,
			fmt.Sprintf("BEGIN_SKILL %s (%s) source=%s", block.name, block.kind, sourcePath),
			content,
			fmt.Sprintf("END_SKILL %s (%s)", block.name, block.kind),
		)

		totalChars += len(content)
		applied = append(applied, fmt.Sprintf("%s (%s)", block.name, block.kind))
	}

	if len(applied) == 0 {
		sectionLines = append(sectionLines, "- No component skill files found for currently available blocks.")
	}
	if len(missing) > 0 {
		sectionLines = append(sectionLines, fmt.Sprintf("Missing component skills: %d block(s).", len(missing)))
		missingPreview := missing
		if len(missingPreview) > componentSkillMissingPreviewLimit {
			missingPreview = missingPreview[:componentSkillMissingPreviewLimit]
		}
		sectionLines = append(sectionLines, "- Example missing blocks: "+strings.Join(missingPreview, ", "))
	}

	return canvasSkillPromptContext{
		PromptSection: strings.Join(sectionLines, "\n"),
		AppliedBlocks: applied,
		MissingBlocks: missing,
	}
}

func loadComponentSkillContent(blockName, blockType string) (string, string, bool) {
	for _, candidatePath := range candidateSkillPaths(blockName, blockType) {
		content, err := os.ReadFile(candidatePath)
		if err != nil {
			continue
		}

		trimmed := strings.TrimSpace(string(content))
		if trimmed == "" {
			continue
		}

		return trimmed, candidatePath, true
	}

	return "", "", false
}

func candidateSkillPaths(blockName, blockType string) []string {
	name := strings.TrimSpace(blockName)
	if name == "" {
		return []string{}
	}

	nameSlash := strings.ReplaceAll(name, ".", string(filepath.Separator))
	blockTypeDir := strings.TrimSpace(blockType)
	componentSkillsDir := canvasComponentSkillsDir

	paths := []string{
		filepath.Join(componentSkillsDir, name+".md"),
		filepath.Join(componentSkillsDir, nameSlash+".md"),
	}

	if blockTypeDir != "" {
		paths = append(paths,
			filepath.Join(componentSkillsDir, blockTypeDir, name+".md"),
			filepath.Join(componentSkillsDir, blockTypeDir, nameSlash+".md"),
		)
	}

	unique := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		unique = append(unique, p)
	}

	return unique
}

func sanitizeAssistantMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "I prepared a draft change set you can review and apply."
	}

	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?'
	})

	filteredParts := make([]string, 0, len(parts))
	for _, part := range parts {
		sentence := strings.TrimSpace(part)
		if sentence == "" {
			continue
		}

		lower := strings.ToLower(sentence)
		if strings.Contains(lower, "skill") || strings.Contains(lower, "guidance file") {
			continue
		}

		filteredParts = append(filteredParts, sentence)
	}

	if len(filteredParts) == 0 {
		return "I prepared a draft change set you can review and apply."
	}

	return strings.TrimSpace(strings.Join(filteredParts, ". ") + ".")
}
