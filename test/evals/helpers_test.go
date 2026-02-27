package evals

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"os"
	"strings"
	"testing"
	"time"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	envRunEvals   = "OPENAI_EVALS"
	envOpenAIKey  = "OPENAI_API_KEY"
	envEvalTrials = "OPENAI_EVAL_ATTEMPTS"
	defaultTimeout = 45 * time.Second
	defaultTrials  = 2
)

type evalNode struct {
	ID              string
	Name            string
	Label           string
	Type            string
	BlockName       string
	IntegrationName string
	Config          map[string]any
}

type evalBlock struct {
	Name  string
	Label string
	Type  string
}

type evalScenario struct {
	Prompt string
	Blocks []evalBlock
	Nodes  []evalNode
}

func runEval(
	t *testing.T,
	scenario evalScenario,
) *canvasesactions.EvalPlan {
	t.Helper()

	if os.Getenv(envRunEvals) != "1" {
		t.Skipf("%s is not enabled", envRunEvals)
	}

	apiKey := os.Getenv(envOpenAIKey)
	if apiKey == "" {
		t.Skipf("%s is not set", envOpenAIKey)
	}

	ctxBlocks := make([]*pb.CanvasAiBlockContext, 0, len(scenario.Blocks))
	for _, b := range scenario.Blocks {
		ctxBlocks = append(ctxBlocks, &pb.CanvasAiBlockContext{
			Name:  b.Name,
			Label: b.Label,
			Type:  b.Type,
		})
	}

	ctxNodes := make([]*pb.CanvasAiNodeContext, 0, len(scenario.Nodes))
	for _, n := range scenario.Nodes {
		cfg, err := structpb.NewStruct(n.Config)
		if err != nil {
			t.Fatalf("invalid node config for %s: %v", n.ID, err)
		}

		ctxNodes = append(ctxNodes, &pb.CanvasAiNodeContext{
			Id:              n.ID,
			Name:            n.Name,
			Label:           n.Label,
			Type:            n.Type,
			BlockName:       n.BlockName,
			IntegrationName: n.IntegrationName,
			Configuration:   cfg,
		})
	}

	req := &pb.SendAiMessageRequest{
		Prompt: scenario.Prompt,
		CanvasContext: &pb.CanvasAiContext{
			Nodes:           ctxNodes,
			AvailableBlocks: ctxBlocks,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	plan, err := canvasesactions.GenerateCanvasAIPlanForEval(ctx, apiKey, req)
	if err != nil {
		t.Fatalf("openai eval planning failed: %v", err)
	}

	if len(plan.Operations) == 0 {
		t.Fatalf("expected at least one operation, got none (assistantMessage=%q)", plan.AssistantMessage)
	}

	return plan
}

func runEvalWithInvariant(
	t *testing.T,
	scenario evalScenario,
	check func(plan *canvasesactions.EvalPlan) error,
) *canvasesactions.EvalPlan {
	t.Helper()

	trials := defaultTrials
	if raw := strings.TrimSpace(os.Getenv(envEvalTrials)); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			t.Fatalf("%s must be a positive integer, got %q", envEvalTrials, raw)
		}
		trials = parsed
	}

	var lastPlan *canvasesactions.EvalPlan
	var lastErr error
	for i := 0; i < trials; i++ {
		plan := runEval(t, scenario)
		if err := check(plan); err == nil {
			return plan
		} else {
			lastPlan = plan
			lastErr = err
		}
	}

	if lastErr == nil {
		t.Fatalf("invariant failed without explicit error")
	}

	failWithPlan(t, fmt.Sprintf("invariant failed after %d attempts: %v", trials, lastErr), lastPlan)
	return nil
}

func operationConfig(operation map[string]any) map[string]any {
	raw, ok := operation["configuration"]
	if !ok {
		return nil
	}

	config, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	return config
}

func operationType(operation map[string]any) string {
	raw, ok := operation["type"]
	if !ok {
		return ""
	}

	opType, _ := raw.(string)
	return opType
}

func addNodeBlockName(operation map[string]any) string {
	blockName, _ := operation["blockName"].(string)
	if blockName != "" {
		return blockName
	}

	node, ok := operation["node"].(map[string]any)
	if !ok {
		return ""
	}

	blockName, _ := node["blockName"].(string)
	return blockName
}

func hasAskFollowupQuestion(operations []map[string]any) bool {
	for _, op := range operations {
		if operationType(op) == "ask_followup_question" {
			return true
		}
	}
	return false
}

func actionList(configuration map[string]any) []string {
	if configuration == nil {
		return nil
	}

	raw, ok := configuration["actions"]
	if !ok {
		return nil
	}

	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	actions := make([]string, 0, len(items))
	for _, item := range items {
		action, ok := item.(string)
		if !ok || action == "" {
			continue
		}
		actions = append(actions, action)
	}

	return actions
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func failWithPlan(t *testing.T, message string, plan *canvasesactions.EvalPlan) {
	t.Helper()

	if plan == nil {
		t.Fatalf("%s\nplan=nil", message)
	}

	operationsJSON, err := json.MarshalIndent(plan.Operations, "", "  ")
	if err != nil {
		t.Fatalf("%s\nassistantMessage=%q\noperationsMarshalError=%v", message, plan.AssistantMessage, err)
	}

	t.Fatalf("%s\nassistantMessage=%q\noperations=%s", message, plan.AssistantMessage, string(operationsJSON))
}
