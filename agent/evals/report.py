from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any

from pydantic_ai.usage import RunUsage
from pydantic_evals.reporting import EvaluationReport
from rich.console import Console

from ai.jsonutil import to_jsonable
from ai.llm_cost_estimate import (
    estimate_cost_usd_for_model,
    matched_claude_pricing_key,
    pricing_reference_url,
)


class ReportBuilder:
    def __init__(
        self,
        report: EvaluationReport,
        *,
        model: str,
        run_usages: dict[str, RunUsage],
        evaluate_wall_seconds: float,
    ) -> None:
        self.report = report
        self.model = model
        self.run_usages = run_usages
        self.evaluate_wall_seconds = evaluate_wall_seconds
        self.console = Console()

    def render(self) -> None:
        display_output_dir = Path("agent/tmp/eval_outputs")
        output_dir = Path("/app/tmp/eval_outputs")
        output_dir.mkdir(parents=True, exist_ok=True)
        if len(self.run_usages) != len(self.report.cases):
            raise RuntimeError(
                f"usage/case count mismatch: {len(self.run_usages)} usage keys vs "
                f"{len(self.report.cases)} report cases (duplicate inputs or missing usage?)"
            )

        task_time_sum_seconds = 0.0
        case_count_with_duration = 0
        total_assertions = 0
        passed_assertions = 0
        usage_by_case: list[dict[str, Any]] = []
        cost_per_case: list[dict[str, Any]] = []
        total_cost_usd: float | None = None
        pricing_match = matched_claude_pricing_key(self.model)

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

            question = self._inputs_key(case_result, case_index=i)
            try:
                run_usage = self.run_usages[question]
            except KeyError as error:
                raise RuntimeError(
                    f"No usage for case {case_name!r}; inputs not found in run_usages keys"
                ) from error
            usage_json = to_jsonable(run_usage)
            usage_by_case.append({"name": case_name, "usage": usage_json})

            case_cost = estimate_cost_usd_for_model(self.model, run_usage)
            if case_cost is not None:
                cost_per_case.append({"name": case_name, "usd": round(case_cost, 6)})
                if total_cost_usd is None:
                    total_cost_usd = 0.0
                total_cost_usd += case_cost

            self.console.print(f"{case_name} {self._format_duration(case_result)}")
            self.console.print(f"  input:        {case_input}")
            self.console.print(f"  output:       {display_filename}")
            self.console.print(f"  toolCalls:    {run_usage.tool_calls}")
            self.console.print(f"  inputTokens:  {run_usage.input_tokens}")
            self.console.print(f"  outputTokens: {run_usage.output_tokens}")
            self.console.print(f"  cost:         ${case_cost:.4f}")

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
                task_time_sum_seconds += duration_seconds
                case_count_with_duration += 1

            if i < len(self.report.cases) - 1:
                self.console.print()
                self.console.print()

        self.console.print()
        self.console.print()

        total_usage = RunUsage()
        for usage in self.run_usages.values():
            total_usage = total_usage + usage

        summary_path = output_dir / "usage_summary.json"
        display_summary = display_output_dir / "usage_summary.json"
        summary_payload: dict[str, Any] = {
            "model": self.model,
            "cases": usage_by_case,
            "total": to_jsonable(total_usage),
            "durations": {
                "task_time_sum_seconds": round(task_time_sum_seconds, 3),
                "wall_time_seconds": round(self.evaluate_wall_seconds, 3),
                "note": (
                    "task_time_sum is pydantic-evals per-case task time added up "
                    "(overlaps when cases run in parallel). wall_time is perf_counter "
                    "around dataset.evaluate only (excludes report I/O)."
                ),
            },
            "estimated_cost_usd": self._cost_summary_json(
                pricing_match=pricing_match,
                per_case=cost_per_case,
                total=total_cost_usd,
            ),
        }

        self.console.print("================================================")
        self.console.print("")

        with summary_path.open("w", encoding="utf-8") as summary_file:
            json.dump(summary_payload, summary_file, indent=2)

        time = task_time_sum_seconds + self.evaluate_wall_seconds

        self.console.print(f"totalTime:    {time:.1f}s ")
        self.console.print(f"totalCost:    ${total_cost_usd:.4f} ")
        self.console.print(f"toolCalls:    {total_usage.tool_calls}")
        self.console.print(f"inputTokens:  {total_usage.input_tokens}")
        self.console.print(f"outputTokens: {total_usage.output_tokens}")

        self.console.print("")

        self.console.print(f"{passed_assertions}/{total_assertions} assertions passed")

    def _inputs_key(self, case_result: Any, *, case_index: int) -> str:
        raw = getattr(case_result, "inputs", getattr(case_result, "input", None))
        if raw is None:
            raise RuntimeError(f"case {case_index} has no inputs; cannot correlate usage")
        if not isinstance(raw, str):
            raise RuntimeError(
                f"case {case_index}: usage correlation requires str inputs, "
                f"got {type(raw).__name__}"
            )
        return raw

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

    def _cost_summary_json(
        self,
        *,
        pricing_match: str | None,
        per_case: list[dict[str, Any]],
        total: float | None,
    ) -> dict[str, Any]:
        base: dict[str, Any] = {
            "reference": pricing_reference_url(),
            "claude_pricing_match": pricing_match,
            "disclaimer": (
                "Approximate Anthropic Claude 4.5/4.6 list rates (base input/output) plus "
                "0.1×/1.25× on cache read/write tokens when present; excludes other Claude "
                "versions, batch, long-context premiums, and non-Claude models."
            ),
        }
        if total is not None:
            base["per_case"] = per_case
            base["total"] = round(total, 6)
        else:
            base["per_case"] = []
            base["total"] = None
        return base

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

  