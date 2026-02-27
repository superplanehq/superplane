package evals

import (
	"fmt"
	"testing"

	canvasesactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
)

func TestGraphRewiringEval(t *testing.T) {
	runEvalWithInvariant(t, evalScenario{
		Prompt: `Insert setData between existing createSandbox and setup nodes.
There is already a direct connection createSandbox -> setup.
Rewire to createSandbox -> setData -> setup.`,
		Blocks: []evalBlock{
			{Name: "setData", Type: "component"},
			{Name: "daytona.createSandbox", Type: "component"},
			{Name: "daytona.executeCommand", Type: "component"},
		},
		Nodes: []evalNode{
			{
				ID:              "node-create",
				Name:            "Create Sandbox",
				Label:           "Create Sandbox",
				Type:            "component",
				BlockName:       "daytona.createSandbox",
				IntegrationName: "daytona",
				Config: map[string]any{
					"sandboxName": "my-sandbox",
				},
			},
			{
				ID:              "node-setup",
				Name:            "Setup and Run App",
				Label:           "Setup and Run App",
				Type:            "component",
				BlockName:       "daytona.executeCommand",
				IntegrationName: "daytona",
				Config: map[string]any{
					"command": "npm start",
				},
			},
		},
	}, func(plan *canvasesactions.EvalPlan) error {
		for _, op := range plan.Operations {
			if operationType(op) == "delete_edge" {
				return nil
			}
		}

		return fmt.Errorf("expected delete_edge operation when inserting node between connected nodes")
	})
}

