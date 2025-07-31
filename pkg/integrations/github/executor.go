package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
)

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
	owner, repo, err := parseRepoName(e.Resource.Name())
	if err != nil {
		return fmt.Errorf("error parsing repository name: %v", err)
	}

	_, err = e.findWorkflow(ctx, owner, repo, spec.Workflow)
	if err != nil {
		return err
	}

	return nil
}

func (e *GitHubExecutor) findWorkflow(ctx context.Context, owner, repo string, workflowName string) (*github.Workflow, error) {
	workflows, _, err := e.gh.client.Actions.ListWorkflows(ctx, owner, repo, nil)
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
	owner, repo, err := parseRepoName(e.Resource.Name())
	if err != nil {
		return nil, fmt.Errorf("error parsing repository name: %v", err)
	}

	workflow, err := e.findWorkflow(context.Background(), owner, repo, spec.Workflow)
	if err != nil {
		return nil, err
	}

	_, err = e.gh.client.Actions.CreateWorkflowDispatchEventByID(
		context.Background(),
		owner,
		repo,
		*workflow.ID,
		github.CreateWorkflowDispatchEventRequest{
			Ref:    spec.Ref,
			Inputs: e.buildWorkflowInputs(spec.Inputs, parameters),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("error triggering workflow: %v", err)
	}

	// After triggering, we need to find the actual workflow run that was created
	// Since GitHub doesn't return the run ID from the dispatch call, we look for the most recent run
	workflowRun, err := e.findTriggeredWorkflowRun(owner, repo, *workflow.ID, spec.Ref)
	if err != nil {
		return nil, fmt.Errorf("error finding triggered workflow run: %v", err)
	}

	return &WorkflowRun{
		ID:         workflowRun.GetID(),
		Status:     workflowRun.GetStatus(),
		Conclusion: workflowRun.GetConclusion(),
		Repository: e.Resource.Name(),
	}, nil
}

func (e *GitHubExecutor) findTriggeredWorkflowRun(owner, repo string, workflowID int64, ref string) (*github.WorkflowRun, error) {
	//
	// TODO: we should add multiple tries here.
	//
	// Wait a short time for the workflow to be created
	time.Sleep(5 * time.Second)

	// List recent workflow runs for this workflow
	runs, _, err := e.gh.client.Actions.ListWorkflowRunsByID(
		context.Background(),
		owner,
		repo,
		workflowID,
		&github.ListWorkflowRunsOptions{
			Branch: ref,
			ListOptions: github.ListOptions{
				PerPage: 1,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing workflow runs: %v", err)
	}

	if len(runs.WorkflowRuns) == 0 {
		return nil, fmt.Errorf("no workflow runs found for workflow %d", workflowID)
	}

	return runs.WorkflowRuns[0], nil
}

func (e *GitHubExecutor) buildWorkflowInputs(fromSpec map[string]string, fromExecution executors.ExecutionParameters) map[string]interface{} {
	inputs := make(map[string]any)

	for k, v := range fromSpec {
		inputs[k] = v
	}

	inputs["superplane_stage_id"] = fromExecution.StageID
	inputs["superplane_execution_id"] = fromExecution.ExecutionID

	if fromExecution.Token != "" {
		inputs["superplane_execution_token"] = fromExecution.Token
	}

	return inputs
}
