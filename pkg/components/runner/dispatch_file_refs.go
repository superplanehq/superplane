package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

// DispatchFileRewriter mints reachable GET URLs for sp-file refs at runner
// prompt-build time. Stored markdown stays unchanged.
func DispatchFileRewriter(exec core.ExecutionContext, timeoutSeconds int) func(string) (string, error) {
	rewriter := &dispatchFileRewriter{exec: exec, timeoutSeconds: timeoutSeconds}
	return rewriter.Rewrite
}

type dispatchFileRewriter struct {
	exec           core.ExecutionContext
	timeoutSeconds int
	once           sync.Once
	organizationID uuid.UUID
	factoryID      uuid.UUID
	workOrderID    uuid.UUID
	resolveErr     error
}

func (r *dispatchFileRewriter) Rewrite(text string) (string, error) {
	if len(blob.FileIDsInMarkdown(text)) == 0 {
		return text, nil
	}
	r.once.Do(r.resolve)
	if r.resolveErr != nil {
		return "", r.resolveErr
	}
	ctx := context.Background()
	rewritten, _, err := storedfiles.DescriptionForDispatch(
		ctx,
		database.DB(ctx),
		blob.Current(),
		r.organizationID,
		r.factoryID,
		r.workOrderID,
		text,
		blob.DispatchDownloadTTL(r.timeoutSeconds),
	)
	if err != nil {
		return "", fmt.Errorf("mint file URLs for the runner prompt: %w", err)
	}
	return rewritten, nil
}

func (r *dispatchFileRewriter) resolve() {
	organizationID, err := uuid.Parse(r.exec.OrganizationID)
	if err != nil {
		r.resolveErr = fmt.Errorf("invalid organization id %q: %w", r.exec.OrganizationID, err)
		return
	}
	canvasID, err := uuid.Parse(r.exec.WorkflowID)
	if err != nil {
		r.resolveErr = fmt.Errorf("invalid canvas id %q: %w", r.exec.WorkflowID, err)
		return
	}
	db := database.DB(context.Background())
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, canvasID)
	if err != nil {
		r.resolveErr = fmt.Errorf("load canvas for file dispatch: %w", err)
		return
	}
	if canvas.FactoryID == nil {
		r.resolveErr = fmt.Errorf("canvas %s is not owned by a factory", canvasID)
		return
	}
	r.organizationID = organizationID
	r.factoryID = *canvas.FactoryID
	execution, err := models.FindWorkOrderExecutionForRun(db, r.exec.RunID)
	if err == nil {
		r.workOrderID = execution.WorkOrderID
	}
}

// MintAgentStepFileRefs copies steps and rewrites prompt/command text with
// rewrite. Preview callers keep the original steps.
func MintAgentStepFileRefs(steps []AgentStep, rewrite func(string) (string, error)) ([]AgentStep, error) {
	if rewrite == nil {
		return steps, nil
	}
	minted := make([]AgentStep, len(steps))
	copy(minted, steps)
	for i := range minted {
		if minted[i].Prompt != nil {
			next, err := rewrite(*minted[i].Prompt)
			if err != nil {
				return nil, fmt.Errorf("mint file URLs in step %d prompt: %w", i+1, err)
			}
			prompt := next
			minted[i].Prompt = &prompt
		}
		if minted[i].Command != nil {
			next, err := rewrite(*minted[i].Command)
			if err != nil {
				return nil, fmt.Errorf("mint file URLs in step %d command: %w", i+1, err)
			}
			command := next
			minted[i].Command = &command
		}
	}
	return minted, nil
}

func MintStepsForRun(exec core.ExecutionContext, timeoutSeconds int, steps []AgentStep) ([]AgentStep, error) {
	return MintAgentStepFileRefs(steps, DispatchFileRewriter(exec, timeoutSeconds))
}
