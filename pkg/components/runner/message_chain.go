package runner

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

type executionMessageChainBuilder interface {
	BuildExecutionMessageChain() (map[string]any, error)
}

func messageChainJSON(expressions core.ExpressionContext) (json.RawMessage, error) {
	if expressions == nil {
		return json.RawMessage("{}"), nil
	}

	builder, ok := expressions.(executionMessageChainBuilder)
	if !ok {
		return json.RawMessage("{}"), nil
	}

	chain, err := builder.BuildExecutionMessageChain()
	if err != nil {
		return nil, fmt.Errorf("build message chain: %w", err)
	}
	if chain == nil {
		return json.RawMessage("{}"), nil
	}

	raw, err := json.Marshal(chain)
	if err != nil {
		return nil, fmt.Errorf("marshal message chain: %w", err)
	}
	if !json.Valid(raw) {
		return json.RawMessage("{}"), nil
	}
	return raw, nil
}
