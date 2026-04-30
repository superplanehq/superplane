from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any, cast

from pydantic_ai.usage import RunUsage
from pydantic_evals.reporting import EvaluationReport, ReportCase, ReportCaseFailure

from ai.jsonutil import to_jsonable
from ai.llm_cost_estimate import (
    estimate_cost_usd_for_model,
    matched_claude_pricing_key,
    pricing_reference_url,
)

type _ReportRow = ReportCase[Any, Any, Any] | ReportCaseFailure[Any, Any, Any]


class ReportBuilder:
    def __init__(
        self,
        report: EvaluationReport,
        *,
        model: str,
        run_usages: dict[str, RunUsage],
        evaluate_wall_seconds: float,
        case_names: list[str],
        interaction_log_paths_by_case_name: dict[str, str] | None = None,
    ) -> None:
        self.report = report
        self.model = model
        self.run_usages = run_usages
        self.evaluate_wall_seconds = evaluate_wall_seconds
        self.case_names = case_names
        self.interaction_log_paths_by_case_name = interaction_log_paths_by_case_name or {}

    def _ordered_report_rows(self) -> list[_ReportRow]:
        """Match dataset order; pydantic-evals splits successes vs task exceptions."""
        by_name: dict[str, _ReportRow] = {}
        for ok_row in self.report.cases:
            if ok_row.name in by_name:
                raise RuntimeError(f"duplicate successful report case name {ok_row.name!r}")
            by_name[ok_row.name] = cast(_ReportRow, ok_row)
        for fail_row in self.report.failures:
            if fail_row.name in by_name:
                raise RuntimeError(f"eval report case name collision: {fail_row.name!r}")
            by_name[fail_row.name] = cast(_ReportRow, fail_row)

        report_names = frozenset(by_name)
        dataset_names = frozenset(self.case_names)
        if report_names != dataset_names:
            raise RuntimeError(
                "eval case name mismatch between dataset and report: "
                f"only_in_report={sorted(report_names - dataset_names)!r} "
                f"only_in_dataset={sorted(dataset_names - report_names)!r}"
            )

        return [by_name[name] for name in self.case_names]

    def render(self) -> None:
        display_output_dir = Path("tmp/agent/evals")
        output_dir = Path("/app/tmp/agent/evals")
        output_dir.mkdir(parents=True, exist_ok=True)

        ordered_rows = self._ordered_report_rows()
        ok_count = len(self.report.cases)
        if len(self.run_usages) != ok_count:
            raise RuntimeError(
                f"usage/case count mismatch: {len(self.run_usages)} usage keys vs "
                f"{ok_count} successful report cases (failures do not record usage; "
                f"duplicate inputs or missing usage on success?)"
            )

        task_time_sum_seconds = 0.0
        case_count_with_duration = 0
        total_assertions = 0
        passed_assertions = 0
        usage_by_case: list[dict[str, Any]] = []
        cost_per_case: list[dict[str, Any]] = []
        total_cost_usd: float | None = None
        pricing_match = matched_claude_pricing_key(self.model)

        print()
        print()

        for i, case_result in enumerate(ordered_rows):
            case_name = case_result.name
            safe_case_name = re.sub(r"[^A-Za-z0-9_.-]", "_", case_name)

            if isinstance(case_result, ReportCaseFailure):
                serialized_output = {
                    "__task_failed__": True,
                    "error_message": case_result.error_message,
                    "error_stacktrace": case_result.error_stacktrace,
                }
                case_input = case_result.inputs
                assertion_values = []
                duration_seconds = None
                run_usage: RunUsage | None = None
                case_cost = None
                usage_by_case.append({"name": case_name, "usage": None, "task_failed": True})
            else:
                serialized_output = self._serialize_output(case_result.output)
                case_input = getattr(case_result, "inputs", getattr(case_result, "input", "-"))
                assertion_values = self._get_assertion_values(case_result)
                duration_seconds = self._duration_seconds(case_result)

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

            # /app/tmp/... inside Docker; tmp/... on host for the display path.
            filename = output_dir / f"{safe_case_name}.json"
            display_filename = display_output_dir / f"{safe_case_name}.json"

            with filename.open("w", encoding="utf-8") as file:
                json.dump(serialized_output, file, indent=2, default=str)

            print(f"{case_name} {self._format_duration(case_result)}")
            print(f"  input:        {case_input}")
            print(f"  output:       {display_filename}")
            print(f"  log:          {self.interaction_log_paths_by_case_name.get(case_name)}")
            if isinstance(case_result, ReportCaseFailure):
                print("  toolCalls:    - (task failed before completion)")
                print("  inputTokens:  -")
                print("  outputTokens: -")
                print("  cacheRead:    -")
                print("  cacheWrite:   -")
                print("  cost:         -")
                print(f"  error:        {case_result.error_message}")
            else:
                assert run_usage is not None
                print(f"  toolCalls:    {run_usage.tool_calls}")
                print(f"  inputTokens:  {run_usage.input_tokens}")
                print(f"  outputTokens: {run_usage.output_tokens}")
                print(f"  cacheRead:    {run_usage.cache_read_tokens}")
                print(f"  cacheWrite:   {run_usage.cache_write_tokens}")
                print(f"  cost:         {self._format_cost(case_cost)}")

            print("  assertions:")
            if isinstance(case_result, ReportCaseFailure):
                assertion_lines: list[str] = []
            else:
                assertion_lines = self._format_assertion_lines(case_result)
            if not assertion_lines:
                print("    - none")
            for assertion_line in assertion_lines:
                print(f"    - {assertion_line}")

            total_assertions += len(assertion_values)
            passed_assertions += sum(
                1 for assertion in assertion_values if bool(getattr(assertion, "value", False))
            )
            if duration_seconds is not None:
                task_time_sum_seconds += duration_seconds
                case_count_with_duration += 1

            if i < len(ordered_rows) - 1:
                print()
                print()

        print()
        print()

        total_usage = RunUsage()
        for usage in self.run_usages.values():
            total_usage = total_usage + usage

        summary_path = output_dir / "usage_summary.json"
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
            "logs_by_case": self.interaction_log_paths_by_case_name,
        }

        print("================================================")
        print("")

        with summary_path.open("w", encoding="utf-8") as summary_file:
            json.dump(summary_payload, summary_file, indent=2)

        time = task_time_sum_seconds + self.evaluate_wall_seconds

        print(f"totalTime:    {time:.1f}s ")
        print(f"totalCost:    {self._format_cost(total_cost_usd)} ")
        print(f"toolCalls:    {total_usage.tool_calls}")
        print(f"inputTokens:  {total_usage.input_tokens}")
        print(f"outputTokens: {total_usage.output_tokens}")

        print("")

        print(f"{passed_assertions}/{total_assertions} assertions passed")

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
            status = "passed" if passed else "failed"
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

    def _format_cost(self, value: float | None) -> str:
        if value is None:
            return "-"
        return f"${value:.4f}"
