"""Parse argv/env for eval case selection and filter datasets by name."""

from __future__ import annotations

import argparse
import os
import sys
from collections.abc import Collection, Sequence
from typing import Any

from evals.case_names import eval_case_name
from evals.cases import dataset


def split_case_names(value: str | None) -> list[str] | None:
    if value is None:
        return None
    names = [part.strip() for part in value.split(",")]
    names = [n for n in names if n]
    return names if names else None


def select_cases(all_cases: Sequence[Any], selected: Collection[str] | None) -> list[Any]:
    if not selected:
        return list(all_cases)
    wanted = frozenset(selected)
    known = {eval_case_name(c, i) for i, c in enumerate(all_cases)}
    unknown = sorted(wanted - known)
    if unknown:
        available = "\n  ".join(sorted(known))
        sys.stderr.write(
            f"Unknown eval case name(s): {', '.join(unknown)}\nValid names:\n  {available}\n"
        )
        raise SystemExit(2)
    return [c for i, c in enumerate(all_cases) if eval_case_name(c, i) in wanted]


def _parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run SuperPlane canvas agent evals.")
    parser.add_argument(
        "--cases",
        metavar="NAMES",
        help="Comma-separated eval case names; overrides CASES when set.",
    )
    parser.add_argument(
        "--list-cases",
        action="store_true",
        help="Print eval case names and exit.",
    )
    return parser.parse_args(argv)


def case_filter(argv: list[str] | None = None) -> list[str] | None:
    """Resolve selected case names from argv and ``CASES`` env, or ``None`` to run all.

    Handles ``--list-cases`` (prints names and exits 0). Unknown ``--cases`` names exit 2.
    """
    args = _parse_args(argv if argv is not None else sys.argv[1:])
    if args.list_cases:
        for index, case in enumerate(dataset.cases):
            print(eval_case_name(case, index))
        raise SystemExit(0)
    selected = split_case_names(args.cases)
    if selected is None:
        selected = split_case_names(os.getenv("CASES"))
    return selected
