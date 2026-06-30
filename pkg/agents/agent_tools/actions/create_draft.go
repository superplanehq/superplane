package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/models"
)

const createDraftActionName = "create_draft"

type createDraftAction struct{}

func (createDraftAction) Name() string {
	return createDraftActionName
}

func (createDraftAction) Execute(_ context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	draft, err := models.CreateDraftBranchFromLive(
		canvasID,
		uuid.MustParse(session.UserID),
		strings.TrimSpace(input.DisplayName),
		nil,
		nil,
	)
	if err != nil {
		return updateResult{}, fmt.Errorf("create draft: %w", err)
	}

	return updateResult{
		Action:    createDraftActionName,
		CanvasID:  session.CanvasID,
		VersionID: draft.ID.String(),
		Draft:     draftResult{VersionID: draft.ID.String(), DisplayName: draft.GitBranch, BranchName: draft.GitBranch},
		Summary:   summarizeCanvasVersion(nil, draft),
	}, nil
}
