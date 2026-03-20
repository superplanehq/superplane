"""Readable eval report output (Pydantic Evals ``print``/``render`` are Rich tables)."""

from __future__ import annotations

import sys
from typing import Any, TextIO

from pydantic_evals.reporting import EvaluationReport


def print_eval_report_plain(
    report: EvaluationReport[Any, Any, Any],
    *,
    file: TextIO | None = None,
    include_input: bool = True,
    include_durations: bool = True,
) -> None:
    """Line-oriented summary: case name, scores, optional input and timings."""
    out = file or sys.stdout
    print(f"Eval: {report.name}", file=out)
    if report.failures:
        print(f"Failures: {len(report.failures)}", file=out)
        for failure in report.failures:
            print(f"  - {failure.name}: {failure.error_message}", file=out)

    for case in report.cases:
        print(f"\ncase: {case.name}", file=out)
        if include_input:
            inp = case.inputs
            if isinstance(inp, str):
                if "\n" in inp:
                    print("  input:", file=out)
                    for line in inp.splitlines():
                        print(f"    {line}", file=out)
                else:
                    print(f"  input: {inp}", file=out)
            else:
                print(f"  input: {inp!r}", file=out)

        if case.scores:
            for name, result in case.scores.items():
                reason = f" ({result.reason})" if result.reason else ""
                print(f"  score {name}: {result.value}{reason}", file=out)
        else:
            print("  scores: (none)", file=out)

        if include_durations:
            print(f"  duration task: {case.task_duration * 1000:.1f} ms", file=out)
            print(f"  duration total: {case.total_duration * 1000:.1f} ms", file=out)
