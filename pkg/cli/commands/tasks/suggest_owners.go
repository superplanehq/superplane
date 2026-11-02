package tasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type taskSuggestOwnersCommand struct {
	workspace *string
	taskID    *string
	paths     *[]string
}

func (c *taskSuggestOwnersCommand) Execute(ctx core.CommandContext) error {
	rawTaskID := strings.TrimSpace(stringValue(c.taskID))
	if rawTaskID == "" {
		return fmt.Errorf("--task is required")
	}

	workspaceID, err := resolveWorkspace(ctx, c.workspace)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(rawTaskID)
	if err != nil {
		return err
	}

	body := openapi_client.NewFactoriesSuggestTaskOwnersBody()
	body.SetPaths(suggestOwnerPaths(c.paths))

	response, _, err := ctx.API.FactoryAPI.
		FactoriesSuggestTaskOwners(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderSuggestOwners(stdout, response)
	})
}

func suggestOwnerPaths(paths *[]string) []string {
	if paths == nil {
		return []string{}
	}
	cleaned := make([]string, 0, len(*paths))
	for _, path := range *paths {
		if trimmed := strings.TrimSpace(path); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func renderSuggestOwners(stdout io.Writer, response *openapi_client.FactoriesSuggestTaskOwnersResponse) error {
	if err := writeMemberLines(stdout, "Current owners", response.GetCurrentOwners()); err != nil {
		return err
	}
	if err := writeCodeOwnerLines(stdout, response.GetCodeOwners()); err != nil {
		return err
	}
	if err := writeMemberLines(stdout, "Members", response.GetMembers()); err != nil {
		return err
	}
	if note := strings.TrimSpace(response.GetHistoryNote()); note != "" {
		if _, err := fmt.Fprintf(stdout, "%s\n", note); err != nil {
			return err
		}
	}
	return nil
}

func writeMemberLines(stdout io.Writer, title string, members []openapi_client.FactoriesTaskOwnerMember) error {
	if _, err := fmt.Fprintf(stdout, "%s:\n", title); err != nil {
		return err
	}
	if len(members) == 0 {
		_, err := fmt.Fprintln(stdout, "  None")
		return err
	}
	for _, member := range members {
		if _, err := fmt.Fprintf(stdout, "  %s, %s\n", ownerLabel(member.GetName(), member.GetGithubLogin(), member.GetEmail(), member.GetUserId()), openTaskCount(member.GetOpenTasks())); err != nil {
			return err
		}
	}
	return nil
}

func writeCodeOwnerLines(stdout io.Writer, owners []openapi_client.FactoriesTaskCodeOwner) error {
	if _, err := fmt.Fprintln(stdout, "Code owners:"); err != nil {
		return err
	}
	if len(owners) == 0 {
		_, err := fmt.Fprintln(stdout, "  None")
		return err
	}
	for _, owner := range owners {
		if _, err := fmt.Fprintf(
			stdout,
			"  %s, %s, %s\n",
			ownerLabel(owner.GetName(), owner.GetGithubLogin(), "", owner.GetUserId()),
			commitCount(owner.GetCommits()),
			openTaskCount(owner.GetOpenTasks()),
		); err != nil {
			return err
		}
	}
	return nil
}

func ownerLabel(name, login, email, userID string) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = strings.TrimSpace(userID)
	}
	if login = strings.TrimSpace(login); login != "" {
		return fmt.Sprintf("%s (%s)", label, login)
	}
	if email = strings.TrimSpace(email); email != "" {
		return fmt.Sprintf("%s <%s>", label, email)
	}
	return label
}

func openTaskCount(count int32) string {
	if count == 1 {
		return "1 open task"
	}
	return fmt.Sprintf("%d open tasks", count)
}

func commitCount(count int32) string {
	if count == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", count)
}
