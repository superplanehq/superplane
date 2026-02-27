package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

// EvalPlan is the minimal AI planning result used by eval tests.
type EvalPlan struct {
	AssistantMessage string
	Operations       []map[string]any
}

// GenerateCanvasAIPlanForEval runs the same planning path used in production,
// but without organization settings lookup, so test/evals can call OpenAI directly.
func GenerateCanvasAIPlanForEval(
	ctx context.Context,
	apiKey string,
	req *pb.SendAiMessageRequest,
) (*EvalPlan, error) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		return nil, err
	}

	plan, err := generateCanvasAIPlan(ctx, reg, apiKey, req)
	if err != nil {
		return nil, err
	}

	return &EvalPlan{
		AssistantMessage: plan.AssistantMessage,
		Operations:       plan.Operations,
	}, nil
}
