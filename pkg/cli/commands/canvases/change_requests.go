package canvases

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type changeRequestListCommand struct {
	statusFilter *string
	onlyMine     *bool
	query        *string
	limit        *int64
	before       *string
}

func (c *changeRequestListCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("list accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasChangeRequestAPI.
		CanvasesListCanvasChangeRequests(ctx.Context, canvasID)

	if c.statusFilter != nil {
		statusFilter := strings.TrimSpace(*c.statusFilter)
		if statusFilter != "" {
			request = request.StatusFilter(statusFilter)
		}
	}
	if c.onlyMine != nil {
		request = request.OnlyMine(*c.onlyMine)
	}
	if c.query != nil {
		query := strings.TrimSpace(*c.query)
		if query != "" {
			request = request.Query(query)
		}
	}
	if c.limit != nil && *c.limit > 0 {
		request = request.Limit(*c.limit)
	}
	if c.before != nil {
		beforeRaw := strings.TrimSpace(*c.before)
		if beforeRaw != "" {
			beforeTime, parseErr := time.Parse(time.RFC3339, beforeRaw)
			if parseErr != nil {
				return fmt.Errorf("invalid --before value %q: expected RFC3339 timestamp", beforeRaw)
			}
			request = request.Before(beforeTime)
		}
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	changeRequests := response.GetChangeRequests()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(changeRequests)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tSTATUS\tCONFLICTED\tCHANGED_NODES\tCONFLICTING_NODES\tTITLE\tUPDATED_AT")

		for _, changeRequest := range changeRequests {
			metadata := changeRequest.GetMetadata()
			diff := changeRequest.GetDiff()

			title := "-"
			if metadata.HasTitle() && strings.TrimSpace(metadata.GetTitle()) != "" {
				title = metadata.GetTitle()
			}

			updatedAt := ""
			if metadata.HasUpdatedAt() {
				updatedAt = metadata.GetUpdatedAt().Format(time.RFC3339)
			}

			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%t\t%d\t%d\t%s\t%s\n",
				metadata.GetId(),
				metadata.GetStatus(),
				metadata.GetIsConflicted(),
				len(diff.GetChangedNodeIds()),
				len(diff.GetConflictingNodeIds()),
				title,
				updatedAt,
			)
		}

		return writer.Flush()
	})
}

type changeRequestGetCommand struct{}

func (c *changeRequestGetCommand) Execute(ctx core.CommandContext) error {
	changeRequestID, canvasTarget, err := parseCanvasChangeRequestTargetArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesDescribeCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
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

	return renderCanvasChangeRequestText(ctx, changeRequest)
}

type changeRequestCreateCommand struct {
	versionID   *string
	title       *string
	description *string
}

func (c *changeRequestCreateCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("create accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	versionID := ""
	if c.versionID != nil {
		versionID = strings.TrimSpace(*c.versionID)
	}

	if versionID == "" {
		versioningContext, err := resolveCanvasVersioningContext(ctx, canvasID)
		if err != nil {
			return err
		}
		if !versioningContext.versioningEnabled {
			return fmt.Errorf("effective canvas versioning is disabled for this canvas")
		}

		versionID, err = findCurrentUserDraftVersionID(ctx, canvasID)
		if err != nil {
			return err
		}
		if versionID == "" {
			return fmt.Errorf("no draft version found; run `superplane canvases update %s --draft ...` first", canvasID)
		}
	}

	body := openapi_client.CanvasesCreateCanvasChangeRequestBody{}
	body.SetVersionId(versionID)

	if c.title != nil {
		trimmedTitle := strings.TrimSpace(*c.title)
		if trimmedTitle != "" {
			body.SetTitle(trimmedTitle)
		}
	}
	if c.description != nil {
		trimmedDescription := strings.TrimSpace(*c.description)
		if trimmedDescription != "" {
			body.SetDescription(trimmedDescription)
		}
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
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

	return renderCanvasChangeRequestSummaryText(ctx, "created", changeRequest)
}

type changeRequestActionCommand struct {
	action openapi_client.ActOnCanvasChangeRequestRequestAction
}

func (c *changeRequestActionCommand) Execute(ctx core.CommandContext) error {
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

type changeRequestResolveCommand struct {
	file            *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *changeRequestResolveCommand) Execute(ctx core.CommandContext) error {
	changeRequestID, canvasTarget, err := parseCanvasChangeRequestTargetArgs(ctx.Args)
	if err != nil {
		return err
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, canvasTarget)
	if err != nil {
		return err
	}

	filePath := ""
	if c.file != nil {
		filePath = strings.TrimSpace(*c.file)
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	canvas, err := loadCanvasForChangeRequestResolve(filePath)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesResolveCanvasChangeRequestBody{}
	body.SetCanvas(canvas)

	autoLayoutValue := ""
	if c.autoLayout != nil {
		autoLayoutValue = strings.TrimSpace(*c.autoLayout)
	}
	autoLayoutScopeValue := ""
	if c.autoLayoutScope != nil {
		autoLayoutScopeValue = strings.TrimSpace(*c.autoLayoutScope)
	}
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}

	if autoLayoutFlagsWereSet(ctx) {
		if autoLayoutValue == "" && (autoLayoutScopeValue != "" || len(autoLayoutNodeIDs) > 0) {
			return fmt.Errorf("--auto-layout is required when using --auto-layout-scope or --auto-layout-node")
		}

		if autoLayoutValue != "" {
			autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
			if parseErr != nil {
				return parseErr
			}
			body.SetAutoLayout(*autoLayout)
		}
	}

	response, _, err := ctx.API.CanvasChangeRequestAPI.
		CanvasesResolveCanvasChangeRequest(ctx.Context, canvasID, changeRequestID).
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

	return renderCanvasChangeRequestSummaryText(ctx, "resolved", changeRequest)
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
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" && ctx.Config != nil {
		trimmedTarget = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}
	if trimmedTarget == "" {
		return "", fmt.Errorf("<name-or-id> is required (or set an active canvas)")
	}

	return findCanvasID(ctx, ctx.API, trimmedTarget)
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
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", metadata.GetCanvasId())
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
			if approver.GetType() == openapi_client.CANVASESCANVASCHANGEREQUESTAPPROVERTYPE_TYPE_USER {
				approverScope = "user " + approver.GetUserId()
			} else if approver.GetType() == openapi_client.CANVASESCANVASCHANGEREQUESTAPPROVERTYPE_TYPE_ROLE {
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

func eventLabelForChangeRequestAction(action openapi_client.ActOnCanvasChangeRequestRequestAction) string {
	switch action {
	case openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_APPROVE:
		return "approved"
	case openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_UNAPPROVE:
		return "unapproved"
	case openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REJECT:
		return "rejected"
	case openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REOPEN:
		return "reopened"
	case openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_PUBLISH:
		return "published"
	default:
		return strings.ToLower(string(action))
	}
}
