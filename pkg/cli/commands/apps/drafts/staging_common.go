package drafts

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func resolveStagingDraftTarget(ctx core.CommandContext) (draftID, appID string, err error) {
	if len(ctx.Args) < 1 || len(ctx.Args) > 2 {
		return "", "", fmt.Errorf("usage: superplane apps drafts staging <commit|reset> <draft-id> [app]")
	}

	draftID = strings.TrimSpace(ctx.Args[0])
	if draftID == "" {
		return "", "", fmt.Errorf("draft id is required")
	}

	if len(ctx.Args) == 2 {
		appID, err = common.ResolveAppNameOrIDArg(ctx, strings.TrimSpace(ctx.Args[1]))
		if err != nil {
			return "", "", err
		}
	} else if active := strings.TrimSpace(ctx.Config.GetActiveApp()); active != "" {
		appID = active
	} else {
		appID, err = common.ResolveAppForDraft(ctx, draftID)
		if err != nil {
			return "", "", err
		}
	}

	if _, err := common.ResolveDraftVersionID(ctx, appID, common.DraftResolveOptions{
		DraftID:     draftID,
		UseDraft:    true,
		AllowCreate: false,
	}); err != nil {
		return "", "", err
	}

	return draftID, appID, nil
}
