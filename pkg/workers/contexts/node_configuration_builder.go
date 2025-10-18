package contexts

import (
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var expressionRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

type NodeConfigurationBuilder struct {
	tx                  *gorm.DB
	workflowID          uuid.UUID
	previousExecutionID *uuid.UUID
	rootEventID         *uuid.UUID
	input               any
	parentBlueprintNode *models.WorkflowNode
}

func NewNodeConfigurationBuilder(tx *gorm.DB, workflowID uuid.UUID) *NodeConfigurationBuilder {
	return &NodeConfigurationBuilder{
		tx:         tx,
		workflowID: workflowID,
	}
}

func (b *NodeConfigurationBuilder) ForBlueprintNode(parentBlueprintNode *models.WorkflowNode) *NodeConfigurationBuilder {
	b.parentBlueprintNode = parentBlueprintNode
	return b
}

func (b *NodeConfigurationBuilder) WithRootEvent(rootEventID *uuid.UUID) *NodeConfigurationBuilder {
	b.rootEventID = rootEventID
	return b
}

func (b *NodeConfigurationBuilder) WithPreviousExecution(previousExecutionID *uuid.UUID) *NodeConfigurationBuilder {
	b.previousExecutionID = previousExecutionID
	return b
}

func (b *NodeConfigurationBuilder) WithInput(input any) *NodeConfigurationBuilder {
	b.input = input
	return b
}

func (b *NodeConfigurationBuilder) Build(configuration map[string]any) (map[string]any, error) {
	resolved, err := b.resolve(configuration)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

func (b *NodeConfigurationBuilder) resolve(configuration map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(configuration))

	for k, v := range configuration {
		resolved, err := b.resolveValue(v)
		if err != nil {
			return nil, fmt.Errorf("error resolving field %s: %w", k, err)
		}
		result[k] = resolved
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) resolveValue(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return b.ResolveExpression(v)

	case map[string]any:
		return b.resolve(v)

	case map[string]string:
		anyMap := make(map[string]any, len(v))
		for key, value := range v {
			anyMap[key] = value
		}

		return b.resolve(anyMap)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			resolved, err := b.resolveValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return v, nil
	}
}

func (b *NodeConfigurationBuilder) ResolveExpression(expression string) (any, error) {
	if !expressionRegex.MatchString(expression) {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := b.resolveExpression(matches[1])
		if e != nil {
			err = e
			return ""
		}

		return fmt.Sprintf("%v", value)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) resolveExpression(expression string) (any, error) {
	env := map[string]any{"$": b.input}

	if b.parentBlueprintNode != nil {
		env["config"] = b.parentBlueprintNode.Configuration.Data()
	}

	exprOptions := []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),

		//
		// Access data from any node in the chain with chain("<NODE ID>").*
		//
		expr.Function("chain", func(params ...any) (any, error) {
			nodeID, ok := params[0].(string)
			if !ok {
				return nil, fmt.Errorf("bad parameter")
			}

			if b.previousExecutionID == nil {
				return nil, fmt.Errorf("no previous execution")
			}

			execution, err := b.findExecutionInChain(nodeID)
			if err != nil {
				return nil, err
			}

			events, err := execution.GetOutputs()
			if err != nil {
				return nil, err
			}

			outputs := make(map[string][]any)
			for _, event := range events {
				outputs[event.Channel] = append(outputs[event.Channel], event.Data.Data())
			}

			return outputs, nil
		}),
	}

	//
	// Access the data from the root event with root().*
	// Only available on workflow-level nodes.
	//
	if b.parentBlueprintNode == nil {
		exprOptions = append(exprOptions, expr.Function("root", func(params ...any) (any, error) {
			if b.rootEventID == nil {
				return nil, fmt.Errorf("no root event found")
			}

			e, err := models.FindWorkflowEvent(*b.rootEventID)
			if err != nil {
				return nil, err
			}

			return e.Data.Data(), nil
		}))
	}

	vm, err := expr.Compile(expression, exprOptions...)
	if err != nil {
		return "", err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return "", fmt.Errorf("expression evaluation failed: %w", err)
	}

	return output, nil
}

func (b *NodeConfigurationBuilder) findExecutionInChain(nodeID string) (*models.WorkflowNodeExecution, error) {
	executionsInChain, err := b.listExecutionsInChain()
	if err != nil {
		return nil, err
	}

	i := slices.IndexFunc(executionsInChain, func(execution models.WorkflowNodeExecution) bool {
		return execution.NodeID == nodeID
	})

	if i == -1 {
		return nil, fmt.Errorf("node %s not found in execution chain", nodeID)
	}

	return &executionsInChain[i], nil
}

func (b *NodeConfigurationBuilder) listExecutionsInChain() ([]models.WorkflowNodeExecution, error) {
	var executions []models.WorkflowNodeExecution

	err := b.tx.Raw(`
		WITH RECURSIVE execution_chain AS (
			SELECT
				id,
				workflow_id,
				node_id,
				root_event_id,
				event_id,
				previous_execution_id,
				parent_execution_id,
				blueprint_id,
				state,
				result,
				result_reason,
				result_message,
				metadata,
				configuration,
				created_at,
				updated_at
			FROM workflow_node_executions
			WHERE id = ? AND workflow_id = ?

			UNION ALL

			-- Recursive case: Get the previous execution
			SELECT
				wne.id,
				wne.workflow_id,
				wne.node_id,
				wne.root_event_id,
				wne.event_id,
				wne.previous_execution_id,
				wne.parent_execution_id,
				wne.blueprint_id,
				wne.state,
				wne.result,
				wne.result_reason,
				wne.result_message,
				wne.metadata,
				wne.configuration,
				wne.created_at,
				wne.updated_at
			FROM workflow_node_executions wne
			INNER JOIN execution_chain ec ON wne.id = ec.previous_execution_id
			WHERE wne.workflow_id = ?
		)
		SELECT *
		FROM execution_chain
		ORDER BY created_at DESC;
	`, b.previousExecutionID, b.workflowID, b.workflowID).Scan(&executions).Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}
