package canvases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		"",
		"Current canvas context JSON:",
		string(canvasContextJSON),
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
