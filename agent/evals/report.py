from __future__ import annotations

import json
import os
import re
from pathlib import Path
from typing import Any

from pydantic_evals.reporting import EvaluationReport
from rich.console import Console


class ReportBuilder:
    def __init__(self, report: EvaluationReport) -> None:
        self.report = report
        self.console = Console()

    def render(self) -> None:
        output_dir = Path("tmp/eval_outputs")
        output_dir.mkdir(parents=True, exist_ok=True)
        total_duration = 0.0
        case_count_with_duration = 0
        total_assertions = 0
        passed_assertions = 0

        self.console.print()
        self.console.print()

        for i, case_result in enumerate(self.report.cases):
            case_name = getattr(case_result, "name", None) or f"case_{i}"
            safe_case_name = re.sub(r"[^A-Za-z0-9_.-]", "_", case_name)
            filename = output_dir / f"{safe_case_name}.json"
            serialized_output = self._serialize_output(case_result.output)
            case_input = getattr(case_result, "inputs", getattr(case_result, "input", "-"))
            assertion_values = self._get_assertion_values(case_result)
            duration_seconds = self._duration_seconds(case_result)

            with filename.open("w", encoding="utf-8") as file:
                json.dump(serialized_output, file, indent=2, default=str)

            self.console.print(f"{case_name} {self._format_duration(case_result)}")
            self.console.print(f"  input: {case_input}")
            self.console.print(f"  output: {filename}")
            self.console.print(f"  assertions: {self._format_assertions_inline(case_result)}")

            total_assertions += len(assertion_values)
            passed_assertions += sum(1 for assertion in assertion_values if bool(getattr(assertion, "value", False)))
            if duration_seconds is not None:
                total_duration += duration_seconds
                case_count_with_duration += 1

            if i < len(self.report.cases) - 1:
                self.console.print()
                self.console.print()

        self.console.print()
        self.console.print()

        self.console.print(f"duration: {total_duration:.1f}s")
        self.console.print(f"results: {passed_assertions}/{total_assertions}")

    def _serialize_output(self, output: Any) -> Any:
        if hasattr(output, "model_dump"):
            return output.model_dump()
        if hasattr(output, "dict"):
            return output.dict()
        return output

    def _get_assertion_values(self, case_result: Any) -> list[Any]:
        assertions = getattr(case_result, "assertions", None)
        if assertions is None:
            return []
        if isinstance(assertions, dict):
            return list(assertions.values())
        try:
            return list(assertions)
        except TypeError:
            return []

    def _format_assertion_lines(self, case_result: Any) -> list[str]:
        lines: list[str] = []
        for assertion in self._get_assertion_values(case_result):
            name = getattr(assertion, "name", "assertion")
            passed = bool(getattr(assertion, "value", False))
            reason = getattr(assertion, "reason", None)
            status = "[green]passed[/]" if passed else "[red]failed[/]"
            line = f"{name}: {status}"
            if reason:
                line = f"{line} - {reason}"
            lines.append(line)
        return lines

    def _format_assertions_inline(self, case_result: Any) -> str:
        assertion_lines = self._format_assertion_lines(case_result)
        if not assertion_lines:
            return "-"
        return "; ".join(assertion_lines)

    def _duration_seconds(self, case_result: Any) -> float | None:
        duration = getattr(case_result, "task_duration", None)
        if duration is None:
            duration = getattr(case_result, "duration", None)
        if duration is None:
            return None
        if hasattr(duration, "total_seconds"):
            return float(duration.total_seconds())
        if isinstance(duration, (int, float)):
            return float(duration)
        return None

    def _format_duration(self, case_result: Any) -> str:
        duration_seconds = self._duration_seconds(case_result)
        if duration_seconds is None:
            return "-"
        return f"{duration_seconds:.1f}s"

  