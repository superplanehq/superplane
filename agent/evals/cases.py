from __future__ import annotations

from pydantic_evals import Case, Dataset

import evals.evaluators as evals

dataset = Dataset(
    evaluators=(
      evals.ToolCalled("get_canvas"),
      evals.ToolCalled("validate_proposal"),
    ),
    cases=[
        Case(
            name="manual_run_then_two_noops",
            inputs=(
                "Build me a basic workflow that starts with a manual run and runs two noop actions"
            ),
            evaluators=(
                evals.CanvasHasTrigger("start"),
                evals.CanvasHasNode("noop", count=2),
                evals.CanvasTotalNodeCount(count=3),
            ),
        ),
        Case(
            name="github_and_slack",
            inputs=(
                "Listen to pull-request comments and send a slack message when a comment is made"
            ),
            evaluators=(
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("slack.sendTextMessage", count=1),
                evals.CanvasTotalNodeCount(count=2),
            ),
        ),
        Case(
            name="github_issue_opened_to_discord",
            inputs=(
                "When a GitHub issue is opened, post a Discord message "
                "that includes the issue title"
            ),
            evaluators=(
                evals.CanvasHasTrigger("github.onIssue"),
                evals.CanvasHasNode("discord.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=2),
                evals.NoDollarDataAsRoot(),
            ),
        ),
        Case(
            name="pr_comment_filter_slack_message_chain",
            inputs=(
                "When a GitHub PR receives a comment, run the filter component then Slack "
                "send text message; the Slack message body should contain the name of the PR "
                "and the time the filter node was executed"
            ),
            evaluators=(
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("filter"),
                evals.CanvasHasNode("slack.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=3),
                evals.BracketSelectorsMatchCanvasNames(
                    scan_scope="all",
                    require_at_least_one_selector=True,
                    target_block_name="slack.sendTextMessage",
                ),
            ),
        ),
        Case(
            name="ephemeral_pr_preview_machines",
            inputs=(
                "Build a workflow that creates ephemeral preview machines for pull requests. "
                "On PR open, create infra and post the preview URL to the PR. "
                "On PR close or after 48 hours, tear it down."
            ),
            evaluators=(
                evals.CanvasHasTrigger("github.onPullRequest"),
                evals.CanvasHasNode("daytona.createRepositorySandbox"),
                evals.CanvasHasNode("wait"),
                evals.CanvasHasNode("daytona.deleteSandbox", count=2),
                evals.CanvasHasWorkflow(
                    "github.onPullRequest",
                    "...",
                    "daytona.createRepositorySandbox",
                    "...",
                    "wait",
                    "...",
                    "daytona.deleteSandbox",
                ),
                evals.CanvasHasWorkflow(
                    "github.onPullRequest",
                    "...",
                    "readMemory",
                    "...",
                    "daytona.deleteSandbox",
                ),
            ),
        ),
        Case(
            name="agent_labeled_issue_auto_resolve",
            inputs="Build a workflow that auto-resolves GitHub issues",
            evaluators=(
                evals.CanvasHasTrigger("github.onIssue"),
                evals.CanvasHasWorkflow(
                    "github.onIssue",
                    "...",
                    "github.createIssueComment",
                    "...",
                    "daytona.executeCode",
                    "...",
                    "github.createIssueComment",
                ),
            ),
        ),
        Case(
            name="github_pr_close_long_open_filter",
            inputs=(
                "When a GitHub pull request is closed, add a filter so the workflow only continues "
                "if that PR had been open for more than an hour."
            ),
            evaluators=(
                evals.CanvasHasTrigger("github.onPullRequest"),
                evals.CanvasHasNode("filter", count=1),
                evals.CanvasHasWorkflow(
                    "github.onPullRequest",
                    "...",
                    "filter",
                ),
                evals.ContainsDatetimeExpression(),
            ),
        ),
    ],
)
