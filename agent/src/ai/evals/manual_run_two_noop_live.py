"""CLI: run the manual-run + two-noop Pydantic Eval with a live LLM and Superplane tools.

Real API: set SUPERPLANE_BASE_URL, SUPERPLANE_API_TOKEN, SUPERPLANE_ORG_ID, CANVAS_ID (or flags),
and provider keys (e.g. OPENAI_API_KEY).

Stub API: pass ``--stub-api`` or set ``EVAL_STUB_SUPERPLANE=1`` — no Superplane env required;
catalog is fixed to trigger ``start`` and component ``noop``. Canvas id defaults to
``eval-stub-canvas``.

Example::

    uv run python -m ai.evals.manual_run_two_noop_live --stub-api --model gpt-4o-mini
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys

from ai.agent import AgentDeps
from ai.evals.basic_workflow import evaluate_manual_run_two_noop_live
from ai.evals.report_output import print_eval_report_plain
from ai.evals.stub_superplane_client import StubSuperplaneClient
from ai.superplane_client import SuperplaneClient, SuperplaneClientConfig


def _require_env(name: str) -> str:
    value = (os.getenv(name) or "").strip()
    if not value:
        print(f"error: missing environment variable {name}", file=sys.stderr)
        sys.exit(1)
    return value


def _stub_api_enabled(flag: bool) -> bool:
    if flag:
        return True
    return (os.getenv("EVAL_STUB_SUPERPLANE") or "").strip().lower() in {"1", "true", "yes", "on"}


async def _async_main(
    *,
    model: str,
    canvas_id: str,
    stub_api: bool,
    report_format: str,
    rich_include_output: bool,
) -> None:
    if stub_api:
        client: SuperplaneClient = StubSuperplaneClient()
    else:
        client = SuperplaneClient(
            SuperplaneClientConfig(
                base_url=_require_env("SUPERPLANE_BASE_URL"),
                api_token=_require_env("SUPERPLANE_API_TOKEN"),
                organization_id=_require_env("SUPERPLANE_ORG_ID"),
            )
        )
    deps = AgentDeps(client=client, canvas_id=canvas_id)
    report = await evaluate_manual_run_two_noop_live(model=model, deps=deps, progress=True)
    if report_format == "rich":
        report.print(
            include_input=True,
            include_output=rich_include_output,
            include_durations=True,
            include_analyses=False,
        )
    else:
        print_eval_report_plain(report, include_input=True, include_durations=True)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Live eval: manual run plus two noops (Pydantic Evals).",
    )
    parser.add_argument(
        "--model",
        default=(os.getenv("AI_MODEL") or "gpt-4o-mini").strip(),
        help="Provider model id (default: AI_MODEL env or gpt-4o-mini).",
    )
    parser.add_argument(
        "--canvas-id",
        default=(os.getenv("CANVAS_ID") or "").strip(),
        help="Canvas UUID (default: CANVAS_ID env; stub mode default: eval-stub-canvas).",
    )
    parser.add_argument(
        "--stub-api",
        action="store_true",
        help="In-memory Superplane API (no network). Or set EVAL_STUB_SUPERPLANE=1.",
    )
    parser.add_argument(
        "--format",
        choices=("plain", "rich"),
        default="plain",
        help="plain: line-oriented summary (default). rich: Pydantic Evals table.",
    )
    parser.add_argument(
        "--rich-full",
        action="store_true",
        help="With --format rich, include the model output column (often very wide).",
    )
    args = parser.parse_args()
    stub_api = _stub_api_enabled(args.stub_api)
    canvas_id = args.canvas_id if args.canvas_id else ("eval-stub-canvas" if stub_api else "")
    if not canvas_id:
        print("error: pass --canvas-id or set CANVAS_ID (not using --stub-api)", file=sys.stderr)
        sys.exit(1)
    if args.model == "test":
        print("error: pass a real --model / AI_MODEL (not the stub 'test')", file=sys.stderr)
        sys.exit(1)
    asyncio.run(
        _async_main(
            model=args.model,
            canvas_id=canvas_id,
            stub_api=stub_api,
            report_format=args.format,
            rich_include_output=args.rich_full,
        )
    )


if __name__ == "__main__":
    main()
