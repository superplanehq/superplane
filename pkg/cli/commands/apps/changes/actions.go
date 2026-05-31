package changes

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type ActionCommand struct {
	action openapi_client.CanvasesActOnCanvasChangeRequestRequestAction
}

func (c *ActionCommand) Execute(ctx core.CommandContext) error {
	changeRequestID, canvasTarget, err := parseCanvasChangeRequestTargetArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesActOnCanvasChangeRequestBody{}
	body.SetAction(c.action)

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesActOnCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if response.ChangeRequest == nil {
		return nil
	}

	changeRequest := *response.ChangeRequest
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(changeRequest)
	}

	return renderCanvasChangeRequestSummaryText(ctx, eventLabelForChangeRequestAction(c.action), changeRequest)
}

func parseCanvasChangeRequestTargetArgs(args []string) (string, string, error) {
	if len(args) < 1 || len(args) > 2 {
		return "", "", fmt.Errorf("expected <change-request-id> [name-or-id]")
	}

	changeRequestID := strings.TrimSpace(args[0])
	if changeRequestID == "" {
		return "", "", fmt.Errorf("<change-request-id> is required")
	}
	if _, err := uuid.Parse(changeRequestID); err != nil {
		return "", "", fmt.Errorf("invalid change request id %q", changeRequestID)
	}

	canvasTarget := ""
	if len(args) == 2 {
		canvasTarget = strings.TrimSpace(args[1])
	}

	return changeRequestID, canvasTarget, nil
}

func resolveCanvasTargetFromOptionalArg(ctx core.CommandContext, target string) (string, error) {
	return common.ResolveAppNameOrIDArg(ctx, target)
}

func loadCanvasForChangeRequestResolve(filePath string) (openapi_client.CanvasesCanvas, error) {
	// #nosec
	data, err := os.ReadFile(filePath)
	if err != nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("failed to read resource file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if kind != models.CanvasKind {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("unsupported resource kind %q for resolve", kind)
	}

	resource, err := models.ParseCanvas(data)
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}

	return models.CanvasFromCanvas(*resource), nil
}

func renderCanvasChangeRequestText(ctx core.CommandContext, changeRequest openapi_client.CanvasesCanvasChangeRequest) error {
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := changeRequest.GetMetadata()
		diff := changeRequest.GetDiff()

		title := "-"
		if metadata.HasTitle() && strings.TrimSpace(metadata.GetTitle()) != "" {
			title = metadata.GetTitle()
		}

		description := "-"
		if metadata.HasDescription() && strings.TrimSpace(metadata.GetDescription()) != "" {
			description = metadata.GetDescription()
		}

		publishedAt := "-"
		if metadata.HasPublishedAt() {
			publishedAt = metadata.GetPublishedAt().Format(time.RFC3339)
		}

		_, _ = fmt.Fprintf(stdout, "ID: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "App: %s\n", metadata.GetCanvasId())
		_, _ = fmt.Fprintf(stdout, "Version: %s\n", metadata.GetVersionId())
		_, _ = fmt.Fprintf(stdout, "Based On Version: %s\n", metadata.GetBasedOnVersionId())
		_, _ = fmt.Fprintf(stdout, "Status: %s\n", metadata.GetStatus())
		_, _ = fmt.Fprintf(stdout, "Is Conflicted: %t\n", metadata.GetIsConflicted())
		_, _ = fmt.Fprintf(stdout, "Changed Nodes: %d\n", len(diff.GetChangedNodeIds()))
		_, _ = fmt.Fprintf(stdout, "Conflicting Nodes: %d\n", len(diff.GetConflictingNodeIds()))
		_, _ = fmt.Fprintf(stdout, "Title: %s\n", title)
		_, _ = fmt.Fprintf(stdout, "Description: %s\n", description)
		_, _ = fmt.Fprintf(stdout, "Published At: %s\n", publishedAt)
		_, _ = fmt.Fprintf(stdout, "Created At: %s\n", formatTimeOrDash(metadata.GetCreatedAt(), metadata.HasCreatedAt()))
		_, _ = fmt.Fprintf(stdout, "Updated At: %s\n", formatTimeOrDash(metadata.GetUpdatedAt(), metadata.HasUpdatedAt()))

		approvals := changeRequest.GetApprovals()
		_, _ = fmt.Fprintf(stdout, "Approvals: %d\n", len(approvals))
		for index, approval := range approvals {
			actor := approval.GetActor()
			actorName := actor.GetName()
			if strings.TrimSpace(actorName) == "" {
				actorName = actor.GetId()
			}
			if strings.TrimSpace(actorName) == "" {
				actorName = "unknown"
			}

			approverScope := "any user"
			approver := approval.GetApprover()
			if approver.GetType() == openapi_client.CHANGEMANAGEMENTAPPROVERTYPE_TYPE_USER {
				approverScope = "user " + approver.GetUserId()
			} else if approver.GetType() == openapi_client.CHANGEMANAGEMENTAPPROVERTYPE_TYPE_ROLE {
				approverScope = "role " + approver.GetRoleName()
			}

			state := strings.ToLower(strings.TrimPrefix(string(approval.GetState()), "STATE_"))
			createdAt := formatTimeOrDash(approval.GetCreatedAt(), approval.HasCreatedAt())
			invalidatedAt := formatTimeOrDash(approval.GetInvalidatedAt(), approval.HasInvalidatedAt())
			_, _ = fmt.Fprintf(
				stdout,
				"  - %d. %s by %s (%s) at %s",
				index+1,
				state,
				actorName,
				approverScope,
				createdAt,
			)
			if approval.HasInvalidatedAt() {
				_, _ = fmt.Fprintf(stdout, " [invalidated at %s]", invalidatedAt)
			}
			_, _ = fmt.Fprintln(stdout, "")
		}

		_, err := fmt.Fprintln(stdout)
		return err
	})
}

func renderCanvasChangeRequestSummaryText(
	ctx core.CommandContext,
	event string,
	changeRequest openapi_client.CanvasesCanvasChangeRequest,
) error {
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := changeRequest.GetMetadata()
		_, _ = fmt.Fprintf(stdout, "Change request %s: %s\n", event, metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Status: %s\n", metadata.GetStatus())
		_, _ = fmt.Fprintf(stdout, "Is Conflicted: %t\n", metadata.GetIsConflicted())
		_, err := fmt.Fprintf(stdout, "Version: %s\n", metadata.GetVersionId())
		return err
	})
}

func formatTimeOrDash(value time.Time, hasValue bool) string {
	if !hasValue {
		return "-"
	}

	return value.Format(time.RFC3339)
}

func eventLabelForChangeRequestAction(action openapi_client.CanvasesActOnCanvasChangeRequestRequestAction) string {
	switch action {
	case openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_APPROVE:
		return "approved"
	case openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_UNAPPROVE:
		return "unapproved"
	case openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REJECT:
		return "rejected"
	case openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REOPEN:
		return "reopened"
	case openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_PUBLISH:
		return "published"
	default:
		return strings.ToLower(string(action))
	}
}
