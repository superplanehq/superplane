package github

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
)

type GitHubOIDCVerifier struct{}

func (h *GitHubOIDCVerifier) Verify(ctx context.Context, verifier *crypto.OIDCVerifier, token string, options integrations.VerifyTokenOptions) error {
	//
	// Verify token signature, subject and audience.
	//
	idToken, err := verifier.Verify(ctx, "https://token.actions.githubusercontent.com", "superplane", token)
	if err != nil {
		return fmt.Errorf("error verifying token: %v", err)
	}

	//
	// Verify that token is for a specific workflow_dispatch run in the correct repository.
	//
	var claims struct {
		EventName    string `json:"event_name"`
		RepositoryID string `json:"repository_id"`
		RunID        string `json:"run_id"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("error parsing claims: %v", err)
	}

	if claims.EventName != "workflow_dispatch" {
		return fmt.Errorf("run is not from a workflow_dispatch event: %s", claims.EventName)
	}

	if options.ChildResource != claims.RunID {
		return fmt.Errorf("invalid run ID: got %s, expected %s", claims.RunID, options.ChildResource)
	}

	if options.ParentResource != claims.RepositoryID {
		return fmt.Errorf("invalid repository: got %s, expected %s", claims.RepositoryID, options.ParentResource)
	}

	return nil
}
