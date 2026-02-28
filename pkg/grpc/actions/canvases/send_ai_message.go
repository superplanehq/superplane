package canvases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	canvasAIOpenAIModel                  = "gpt-5.3-codex"
	canvasAIAssociatedEncryptionDataName = "agent_mode_openai_api_key"
	componentSkillMaxCharsPerBlock       = 3000
	componentSkillMaxCharsTotal          = 14000
	componentSkillMissingPreviewLimit    = 10
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

	prompt := strings.Join([]string{
		"You are an AI planner for a workflow canvas editor.",
		"Return strict JSON only with this schema:",
		`{"assistantMessage":"string","operations":[{"type":"add_node","nodeKey":"optional-string","blockName":"required-block-name","nodeName":"optional","configuration":{"optional":"object"},"position":{"x":123,"y":456},"source":{"nodeKey":"optional","nodeId":"optional","nodeName":"optional","handleId":"optional-or-null"}},{"type":"connect_nodes","source":{"nodeKey":"optional","nodeId":"optional","nodeName":"optional","handleId":"optional-or-null"},"target":{"nodeKey":"optional","nodeId":"optional","nodeName":"optional"}},{"type":"update_node_config","target":{"nodeKey":"optional","nodeId":"optional","nodeName":"optional"},"configuration":{"required":"object"},"nodeName":"optional"},{"type":"delete_node","target":{"nodeKey":"optional","nodeId":"optional","nodeName":"optional"}}]}`,
		"Rules:",
		"- Use only blockName values present in availableBlocks.",
		"- Prefer add_node with nodeKey so follow-up connect/update operations can reference new nodes.",
		"- Keep operations minimal and valid.",
		"- Never invent component names or use components not listed in availableBlocks.",
		"- First inspect existing nodes and prefer updating/reusing/reconnecting them before asking follow-up questions.",
		"- If parts of the request are ambiguous, make reasonable assumptions and still return best-effort operations when there is a safe place to apply them.",
		"- Ask a clarifying question and return operations as [] only when you cannot safely map the request to availableBlocks or cannot identify any valid target/location in the current canvas.",
		"- Prefer a left-to-right horizontal flow.",
		"- Use delete_node when the user explicitly asks to remove/delete a node.",
		"- For add_node, include position when possible.",
		"- Use at least 420px horizontal spacing between sequential nodes to avoid overlap.",
		"- Keep nodes in the same path on the same y lane when possible.",
		"- For branches, use vertical lane offsets of at least 220px.",
		"- If you used assumptions, mention them briefly in assistantMessage while still returning operations.",
		"- If component skill guidance is provided below, treat it as the source of truth for those blocks.",
		"- Never mention skills, skill files, or internal guidance sources in assistantMessage.",
		"- Data-flow expression rules (SuperPlane message chain):",
		"- Access upstream node payloads with explicit node-name lookups such as $[\"Node Name\"].data.field.",
		"- For expression-capable string fields, wrap expressions with handlebars: {{ ... }}.",
		"- For embedded string interpolation, use literal text plus handlebars (example: root@{{ $[\"Create Hetzner Machine\"].data.ipv4 }}).",
		"- previous() means immediate upstream only; use previous(<depth>) only when depth-based access is explicitly intended.",
		"- root() refers to the root trigger event payload.",
		"- Use memory.find(\"namespace\", {\"field\": value}) to filter memory rows by exact key/value matches.",
		"- Use memory.findFirst(\"namespace\", {\"field\": value}) to get the first matching memory row (or nil).",
		"- Never use non-SuperPlane syntaxes like {{steps.create_hetzner.ipv4}} or other steps.* references.",
		"- When configuring fields like SSH host/IP, identify the actual producer node in the run chain and reference that node by name instead of assuming previous().",
		"",
		"Current canvas context JSON:",
		string(canvasContextJSON),
		"",
		skillContext.PromptSection,
		"",
		"User request:",
		req.GetPrompt(),
	}, "\n")

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
		return &openAICanvasPlan{
			AssistantMessage: "I couldn't produce executable operations for that request. Please rephrase with specific component names or desired flow.",
			Operations:       []map[string]interface{}{},
		}, nil
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
	baseDir := filepath.Join("templates", "skills")
	blockTypeDir := strings.TrimSpace(blockType)

	paths := []string{
		filepath.Join(baseDir, name+".md"),
		filepath.Join(baseDir, nameSlash+".md"),
	}

	if blockTypeDir != "" {
		paths = append(paths,
			filepath.Join(baseDir, blockTypeDir, name+".md"),
			filepath.Join(baseDir, blockTypeDir, nameSlash+".md"),
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
