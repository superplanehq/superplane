from __future__ import annotations

import asyncio
import os
import evals.evaluators as evals
import time

from pydantic_ai.usage import RunUsage
from pydantic_evals import Case, Dataset

from ai.agent import AgentDeps, build_agent, build_prompt
from ai.models import CanvasAnswer, CanvasQuestionRequest
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig
from evals.report import ReportBuilder

dataset = Dataset(
    cases=[
        Case(
            name="manual_run_then_two_noops",
            inputs=(
                "Build me a basic workflow that starts with a manual run and runs two noop actions"
            ),
            evaluators=[
              evals.CanvasHasTrigger("start"),
              evals.CanvasHasNode("noop", count=2),
              evals.CanvasTotalNodeCount(count=3),
            ],
        ),

        Case(
            name="github_and_slack",
            inputs=(
                "Listen to pull-request comments and send a slack message when "
                "a comment is made"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("slack.sendTextMessage", count=1),
                evals.CanvasTotalNodeCount(count=2),
            ],
        ),
        Case(
            name="github_issue_opened_to_discord",
            inputs=(
                "When a GitHub issue is opened, post a Discord message "
                "that includes the issue title"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onIssue"),
                evals.CanvasHasNode("discord.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=2),
                evals.NoDollarDataAsRoot(),
            ],
        ),
        Case(
            name="pr_comment_filter_slack_message_chain",
            inputs=(
                "When a GitHub PR receives a comment, run the filter component then Slack "
                "send text message; the Slack message body should contain the name of the PR and the time the filter node was executed"
            ),
            evaluators=[
                evals.CanvasHasTrigger("github.onPRComment"),
                evals.CanvasHasNode("filter"),
                evals.CanvasHasNode("slack.sendTextMessage"),
                evals.CanvasTotalNodeCount(count=3),
                evals.BracketSelectorsMatchCanvasNames(
                    scan_scope="all",
                    require_at_least_one_selector=True,
                    target_block_name="slack.sendTextMessage",
                ),
            ],
        ),
        Case(
            name="ephemeral_pr_preview_machines",
            inputs=(
                "Build a workflow that creates ephemeral preview machines for pull requests. "
                "On PR open, create infra and post the preview URL to the PR. "
                "On PR close or after 48 hours, tear it down."
            ),
            evaluators=[
              evals.CanvasHasTrigger("github.onPullRequest"),
              evals.CanvasHasNode("daytona.createRepositorySandbox"),
              evals.CanvasHasNode("wait"),
              evals.CanvasHasNode("daytona.deleteSandbox", count=2),
              evals.CanvasHasWorkflow("github.onPullRequest", "...", "daytona.createRepositorySandbox", "...", "wait", "...", "daytona.deleteSandbox"),
              evals.CanvasHasWorkflow("github.onPullRequest", "...", "readMemory", "...", "daytona.deleteSandbox"),
            ],
        ),
        Case(
            name="agent_labeled_issue_auto_resolve",
            inputs="Build a workflow that auto-resolves GitHub issues",
            evaluators=[
              evals.CanvasHasTrigger("github.onIssue"),
              evals.CanvasHasWorkflow("github.onIssue", "...", "github.createIssueComment", "...", "daytona.executeCode", "...", "github.createIssueComment"),
            ],
        ),
    ],
)

def load_env() -> dict[str, str]:
    return {
        "model": os.getenv("AI_MODEL", "").strip(),
        "base_url": "http://app:8000",
        "api_token": os.getenv("SUPERPLANE_API_TOKEN", "").strip(),
        "organization_id": os.getenv("EVAL_ORG_ID", "").strip(),
        "canvas_id": os.getenv("EVAL_CANVAS_ID", "").strip(),
    }

async def runner() -> None:
    env = load_env()

    deps = AgentDeps(
        client=SuperplaneClient(
            SuperplaneClientConfig(
                base_url=env["base_url"],
                api_token=env["api_token"],
                organization_id=env["organization_id"],
            )
        ),
        canvas_id=env["canvas_id"],
    )
    agent = build_agent(env["model"])
    # Keyed by case ``inputs`` string so usage matches when cases run concurrently.
    # Duplicate prompts across cases are not supported (detected under a lock).
    run_usages: dict[str, RunUsage] = {}
    usage_lock = asyncio.Lock()

    async def task(question: str) -> CanvasAnswer:
        payload = CanvasQuestionRequest(question=question, canvas_id=deps.canvas_id)
        result = await agent.run(build_prompt(payload), deps=deps)
        async with usage_lock:
            if question in run_usages:
                raise RuntimeError(
                    "Duplicate eval case inputs are not supported for usage correlation "
                    f"(collision on {question[:120]!r}…)"
                )
            run_usages[question] = result.usage()
        return result.output

    wall_start = time.perf_counter()
    report = await dataset.evaluate(task, progress=True)
    evaluate_wall_seconds = time.perf_counter() - wall_start
    ReportBuilder(
        report,
        model=env["model"],
        run_usages=run_usages,
        evaluate_wall_seconds=evaluate_wall_seconds,
    ).render()

def main() -> None:
    asyncio.run(runner())

if __name__ == "__main__":
    main()
