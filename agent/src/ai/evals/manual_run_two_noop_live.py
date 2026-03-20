"""CLI: manual-run + two-noop eval — real LLM, Superplane API stubbed by default.

Default uses ``StubSuperplaneClient`` (catalog: ``start``, ``noop``). Set provider keys
(e.g. ``OPENAI_API_KEY``).

Optional ``--real-superplane`` uses HTTP + ``SUPERPLANE_*`` and ``CANVAS_ID``.

Example::

    uv run python -m ai.evals.manual_run_two_noop_live --model gpt-4o-mini
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


async def _async_main(*, model: str, canvas_id: str, use_real_superplane: bool) -> None:
    if use_real_superplane:
        client: SuperplaneClient = SuperplaneClient(
            SuperplaneClientConfig(
                base_url=_require_env("SUPERPLANE_BASE_URL"),
                api_token=_require_env("SUPERPLANE_API_TOKEN"),
                organization_id=_require_env("SUPERPLANE_ORG_ID"),
            )
        )
    else:
        client = StubSuperplaneClient()

    deps = AgentDeps(client=client, canvas_id=canvas_id)
    report = await evaluate_manual_run_two_noop_live(model=model, deps=deps, progress=True)
    print_eval_report_plain(report, include_input=True, include_durations=True)


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Eval: real LLM proposes manual run + two noops; Superplane stubbed by default."
        ),
    )
    parser.add_argument(
        "--model",
        default=(os.getenv("AI_MODEL") or "gpt-4o-mini").strip(),
        help="Provider model id (default: AI_MODEL env or gpt-4o-mini).",
    )
    parser.add_argument(
        "--canvas-id",
        default=(os.getenv("CANVAS_ID") or "").strip(),
        help="Canvas id for deps (default: CANVAS_ID or eval-stub-canvas when API is stubbed).",
    )
    parser.add_argument(
        "--real-superplane",
        action="store_true",
        help="Call real Superplane HTTP API (requires SUPERPLANE_* and CANVAS_ID).",
    )
    args = parser.parse_args()

    use_real = args.real_superplane
    canvas_id = args.canvas_id or ("eval-stub-canvas" if not use_real else "")
    if not canvas_id:
        print(
            "error: set CANVAS_ID or --canvas-id when using --real-superplane",
            file=sys.stderr,
        )
        sys.exit(1)

    if args.model == "test":
        print("error: pass a real --model / AI_MODEL (not the stub 'test')", file=sys.stderr)
        sys.exit(1)

    asyncio.run(_async_main(model=args.model, canvas_id=canvas_id, use_real_superplane=use_real))


if __name__ == "__main__":
    main()
