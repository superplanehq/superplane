from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any

from pydantic_ai.usage import RunUsage
from pydantic_evals.reporting import EvaluationReport
from rich.console import Console

from ai.jsonutil import to_jsonable


class ReportBuilder:
    def __init__(
        self,
        report: EvaluationReport,
        *,
        model: str,
        run_usages: list[RunUsage],
    ) -> None:
        self.report = report
        self.model = model
        self.run_usages = run_usages
        self.console = Console()

    def render(self) -> None:
        display_output_dir = Path("agent/tmp/eval_outputs")
        output_dir = Path("/app/tmp/eval_outputs")
        output_dir.mkdir(parents=True, exist_ok=True)
        if len(self.run_usages) != len(self.report.cases):
            raise RuntimeError(
                f"usage/case count mismatch: {len(self.run_usages)} usages vs "
                f"{len(self.report.cases)} report cases"
            )

        total_duration = 0.0
        case_count_with_duration = 0
        total_assertions = 0
        passed_assertions = 0
        usage_by_case: list[dict[str, Any]] = []

        self.console.print()
        self.console.print()

        for i, case_result in enumerate(self.report.cases):
            case_name = getattr(case_result, "name", None) or f"case_{i}"
            safe_case_name = re.sub(r"[^A-Za-z0-9_.-]", "_", case_name)

            serialized_output = self._serialize_output(case_result.output)
            case_input = getattr(case_result, "inputs", getattr(case_result, "input", "-"))
            assertion_values = self._get_assertion_values(case_result)
            duration_seconds = self._duration_seconds(case_result)

            # /app/tmp/... inside Docker; agent/tmp/... on host for the display path.
            filename = output_dir / f"{safe_case_name}.json"
            display_filename = display_output_dir / f"{safe_case_name}.json"

            with filename.open("w", encoding="utf-8") as file:
                json.dump(serialized_output, file, indent=2, default=str)

            run_usage = self.run_usages[i]
            usage_json = to_jsonable(run_usage)
            usage_by_case.append({"name": case_name, "usage": usage_json})

            self.console.print(f"{case_name} {self._format_duration(case_result)}")
            self.console.print(f"  input: {case_input}")
            self.console.print(f"  output: {display_filename}")
            self.console.print(f"  usage: {self._format_usage_line(run_usage)}")
            self.console.print("  assertions:")
            assertion_lines = self._format_assertion_lines(case_result)
            if not assertion_lines:
                self.console.print("    - none")
            for assertion_line in assertion_lines:
                self.console.print(f"    - {assertion_line}")

            total_assertions += len(assertion_values)
            passed_assertions += sum(
                1
                for assertion in assertion_values
                if bool(getattr(assertion, "value", False))
            )
            if duration_seconds is not None:
                total_duration += duration_seconds
                case_count_with_duration += 1

            if i < len(self.report.cases) - 1:
                self.console.print()
                self.console.print()

        self.console.print()
        self.console.print()

        total_usage = RunUsage()
        for usage in self.run_usages:
            total_usage = total_usage + usage

        summary_path = output_dir / "usage_summary.json"
        display_summary = display_output_dir / "usage_summary.json"
        summary_payload: dict[str, Any] = {
            "model": self.model,
            "cases": usage_by_case,
            "total": to_jsonable(total_usage),
        }
        with summary_path.open("w", encoding="utf-8") as summary_file:
            json.dump(summary_payload, summary_file, indent=2)

        self.console.print(f"duration: {total_duration:.1f}s")
        self.console.print(f"results: {passed_assertions}/{total_assertions}")
        self.console.print(f"usage (total): {self._format_usage_line(total_usage)}")
        self.console.print(f"usage summary: {display_summary}")

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

    def _format_usage_line(self, usage: RunUsage) -> str:
        parts = [
            f"requests={usage.requests}",
            f"tool_calls={usage.tool_calls}",
            f"in={usage.input_tokens}",
            f"out={usage.output_tokens}",
        ]
        if usage.cache_read_tokens or usage.cache_write_tokens:
            parts.append(f"cache_r={usage.cache_read_tokens}")
            parts.append(f"cache_w={usage.cache_write_tokens}")
        if usage.details:
            parts.append(f"details={usage.details}")
        return " ".join(parts)

  