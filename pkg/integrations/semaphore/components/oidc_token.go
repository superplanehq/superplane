package components

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/ciauth"
	"github.com/superplanehq/superplane/pkg/core"
)

const oidcTokenParameterName = "SUPERPLANE_OIDC_TOKEN"

const (
	semaphoreClaimProjectID    = "project_id"
	semaphoreClaimPipelineFile = "pipeline_file"
	semaphoreClaimRef          = "ref"
	semaphoreClaimCommitSha    = "commit_sha"
)

const (
	semaphoreOIDCTokenAudience = ciauth.ExecutionTokenAudience
	semaphoreOIDCTokenDuration = time.Hour
)

func (r *RunWorkflow) signOIDCToken(ctx core.ExecutionContext, spec RunWorkflowSpec, metadata RunWorkflowNodeMetadata) (string, error) {
	if ctx.OIDC == nil {
		return "", fmt.Errorf("OIDC provider is not configured")
	}

	claims := map[string]any{
		ciauth.ClaimOrgID:       ctx.OrganizationID,
		ciauth.ClaimCanvasID:    ctx.WorkflowID,
		ciauth.ClaimNodeID:      ctx.NodeID,
		ciauth.ClaimExecutionID: ctx.ID.String(),
		ciauth.ClaimComponent:   r.Name(),
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
