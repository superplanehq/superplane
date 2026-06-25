package components

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const oidcTokenParameterName = "SUPERPLANE_OIDC_TOKEN"

const (
	semaphoreOIDCTokenAudience = "superplane-ci"
	semaphoreOIDCTokenDuration = time.Hour

	semaphoreClaimOrgID        = "org_id"
	semaphoreClaimCanvasID     = "canvas_id"
	semaphoreClaimNodeID       = "node_id"
	semaphoreClaimExecutionID  = "execution_id"
	semaphoreClaimComponent    = "component"
	semaphoreClaimProjectID    = "project_id"
	semaphoreClaimPipelineFile = "pipeline_file"
	semaphoreClaimRef          = "ref"
	semaphoreClaimCommitSha    = "commit_sha"
)

func (r *RunWorkflow) signOIDCToken(ctx core.ExecutionContext, spec RunWorkflowSpec, metadata RunWorkflowNodeMetadata) (string, error) {
	if ctx.OIDC == nil {
		return "", fmt.Errorf("OIDC provider is not configured")
	}

	claims := map[string]any{
		semaphoreClaimOrgID:       ctx.OrganizationID,
		semaphoreClaimCanvasID:    ctx.WorkflowID,
		semaphoreClaimNodeID:      ctx.NodeID,
		semaphoreClaimExecutionID: ctx.ID.String(),
		semaphoreClaimComponent:   r.Name(),
	}

	if metadata.Project != nil && metadata.Project.ID != "" {
		claims[semaphoreClaimProjectID] = metadata.Project.ID
	}
	if spec.PipelineFile != "" {
		claims[semaphoreClaimPipelineFile] = spec.PipelineFile
	}
	if spec.Ref != "" {
		claims[semaphoreClaimRef] = spec.Ref
	}
	if spec.CommitSha != "" {
		claims[semaphoreClaimCommitSha] = spec.CommitSha
	}

	return ctx.OIDC.Sign(
		fmt.Sprintf("execution:%s", ctx.ID),
		semaphoreOIDCTokenDuration,
		semaphoreOIDCTokenAudience,
		claims,
	)
}
