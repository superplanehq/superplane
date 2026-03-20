"""Pydantic Evals datasets and evaluators for the SuperPlane canvas agent."""

from ai.evals.basic_workflow import (
    MANUAL_RUN_TWO_NOOP_PROMPT,
    ManualRunTwoNoopTopologyEvaluator,
    build_manual_run_two_noop_dataset,
    evaluate_manual_run_two_noop_live,
    score_manual_run_two_noop_proposal,
)
from ai.evals.report_output import print_eval_report_plain
from ai.evals.stub_superplane_client import StubSuperplaneClient

__all__ = [
    "MANUAL_RUN_TWO_NOOP_PROMPT",
    "ManualRunTwoNoopTopologyEvaluator",
    "StubSuperplaneClient",
    "build_manual_run_two_noop_dataset",
    "evaluate_manual_run_two_noop_live",
    "print_eval_report_plain",
    "score_manual_run_two_noop_proposal",
]
