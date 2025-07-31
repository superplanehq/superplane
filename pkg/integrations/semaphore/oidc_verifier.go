package semaphore

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
)

type SemaphoreOIDCVerifier struct{}

func (h *SemaphoreOIDCVerifier) Verify(ctx context.Context, verifier *crypto.OIDCVerifier, token string, options integrations.VerifyTokenOptions) error {
	//
	// Verify token signature, subject and audience.
	//
	idToken, err := verifier.Verify(ctx, options.IntegrationURL, options.IntegrationURL, token)
	if err != nil {
		return fmt.Errorf("error verifying token: %v", err)
	}

	//
	// Verify that token is for the correct project and workflow.
	//
	var claims struct {
		WorkflowID string `json:"wf_id"`
		ProjectID  string `json:"prj_id"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("error parsing claims: %v", err)
	}

	if claims.WorkflowID != options.ChildResource {
		return fmt.Errorf("invalid workflow ID: %s", claims.WorkflowID)
	}

	if claims.ProjectID != options.ParentResource {
		return fmt.Errorf("invalid project ID: %s", claims.ProjectID)
	}

	return nil
}
