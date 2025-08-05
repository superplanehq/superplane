package github

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/retry"
)

var inProgressRunStates = []string{"in_progress", "queued", "requested", "waiting", "pending"}

type GitHubExecutor struct {
	Resource integrations.Resource
	gh       *GitHubResourceManager
}

type ExecutorSpec struct {
	Workflow string            `json:"workflow"`
	Ref      string            `json:"ref"`
	Inputs   map[string]string `json:"inputs"`
}

func NewGitHubExecutor(resourceManager integrations.ResourceManager, resource integrations.Resource) (integrations.Executor, error) {
	gh, ok := resourceManager.(*GitHubResourceManager)
	if !ok {
		return nil, fmt.Errorf("invalid resource manager")
	}

	return &GitHubExecutor{
		gh:       gh,
		Resource: resource,
	}, nil
}

func (e *GitHubExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec ExecutorSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.Workflow == "" {
		return fmt.Errorf("workflow is required")
	}

	if spec.Ref == "" {
		return fmt.Errorf("ref is required")
	}

	return e.validateWorkflow(ctx, spec)
}

func (e *GitHubExecutor) validateWorkflow(ctx context.Context, spec ExecutorSpec) error {
	_, err := e.findWorkflow(ctx, spec.Workflow)
	if err != nil {
		return err
	}

	return nil
}

func (e *GitHubExecutor) findWorkflow(ctx context.Context, workflowName string) (*github.Workflow, error) {
	workflows, _, err := e.gh.client.Actions.ListWorkflows(ctx, e.gh.Owner, e.Resource.Name(), nil)
	if err != nil {
		return nil, fmt.Errorf("error listing workflows: %v", err)
	}

	for _, workflow := range workflows.Workflows {
		if workflow.GetName() == workflowName || workflow.GetPath() == workflowName {
			return workflow, nil
		}
	}

	return nil, fmt.Errorf("workflow %s not found in repository %s", workflowName, e.Resource.Name())
}

func (e *GitHubExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (integrations.StatefulResource, error) {
	var spec ExecutorSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	return e.triggerWorkflow(spec, parameters)
}

func (e *GitHubExecutor) triggerWorkflow(spec ExecutorSpec, parameters executors.ExecutionParameters) (integrations.StatefulResource, error) {
	workflow, err := e.findWorkflow(context.Background(), spec.Workflow)
	if err != nil {
		return nil, err
	}

	_, err = e.gh.client.Actions.CreateWorkflowDispatchEventByID(
		context.Background(),
		e.gh.Owner,
		e.Resource.Name(),
		*workflow.ID,
		github.CreateWorkflowDispatchEventRequest{
			Ref:    spec.Ref,
			Inputs: e.buildWorkflowInputs(spec.Inputs, parameters),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("error triggering workflow: %v", err)
	}

	//
	// GitHub doesn't expose the run ID from the dispatch call, so we need to find it.
	//
	workflowRun, err := e.findTriggeredWorkflowRun(*workflow.ID, spec.Ref)
	if err != nil {
		return nil, fmt.Errorf("error finding triggered workflow run: %v", err)
	}

	return &WorkflowRun{
		ID:         workflowRun.GetID(),
		Status:     workflowRun.GetStatus(),
		Conclusion: workflowRun.GetConclusion(),
	}, nil
}

func (e *GitHubExecutor) findTriggeredWorkflowRun(workflowID int64, ref string) (*github.WorkflowRun, error) {
	var run *github.WorkflowRun

	//
	// We need to use a creation time filter to ensure we only get the run we just triggered.
	// See: https://docs.github.com/en/search-github/getting-started-with-searching-on-github/understanding-the-search-syntax
	//
	creationTimeFilter := fmt.Sprintf(
		"%s..%s",
		time.Now().Add(-time.Minute).Format(time.RFC3339),
		time.Now().Add(time.Minute).Format(time.RFC3339),
	)

	err := retry.WithConstantWait(func() error {
		runs, _, err := e.gh.client.Actions.ListWorkflowRunsByID(
			context.Background(),
			e.gh.Owner,
			e.Resource.Name(),
			workflowID,
			&github.ListWorkflowRunsOptions{
				Branch:  ref,
				Event:   "workflow_dispatch",
				Created: creationTimeFilter,
				ListOptions: github.ListOptions{
					PerPage: 10,
				},
			},
		)

		if err != nil {
			return fmt.Errorf("error listing workflow runs: %v", err)
		}

		if len(runs.WorkflowRuns) == 0 {
			return fmt.Errorf("Empty list of workflow_dispatch runs found for workflow %d with filter %s", workflowID, creationTimeFilter)
		}

		for _, r := range runs.WorkflowRuns {
			if slices.Contains(inProgressRunStates, r.GetStatus()) {
				run = r
				return nil
			}
		}

		return fmt.Errorf("workflow run not found")
	}, retry.Options{
		Task:         "Find triggered workflow run",
		MaxAttempts:  15,
		Wait:         2 * time.Second,
		InitialDelay: 2 * time.Second,
		Verbose:      false,
	})

	if err != nil {
		return nil, err
	}

	return run, nil
}

func (e *GitHubExecutor) buildWorkflowInputs(fromSpec map[string]string, fromExecution executors.ExecutionParameters) map[string]interface{} {
	inputs := make(map[string]any)
	for k, v := range fromSpec {
		inputs[k] = v
	}

	inputs["superplane_execution_id"] = fromExecution.ExecutionID
	return inputs
}
