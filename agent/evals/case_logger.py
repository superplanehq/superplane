from __future__ import annotations

import asyncio
import time
from pathlib import Path
from typing import TextIO


class CaseLogger:
    def __init__(self, run_id: str, case_names: list[str]) -> None:
        self._lock = asyncio.Lock()
        self._case_started_at_monotonic: dict[str, float] = {}
        output_dir = Path("/app/tmp/agent/evals")
        display_output_dir = Path("tmp/agent/evals")
        output_dir.mkdir(parents=True, exist_ok=True)
        run_suffix = run_id.split("_")[-1] if "_" in run_id else run_id

        self._files_by_case_name: dict[str, TextIO] = {}
        self._display_path_by_case_name: dict[str, str] = {}
        for index, case_name in enumerate(case_names, start=1):
            log_id = f"{run_suffix}-{index:02d}"
            filename = f"{log_id}.log"
            output_path = output_dir / filename
            file = output_path.open("w", encoding="utf-8")
            file.write(f"case={case_name}\n")
            file.flush()
            self._files_by_case_name[case_name] = file
            self._display_path_by_case_name[case_name] = str(display_output_dir / filename)

    @property
    def display_paths_by_case_name(self) -> dict[str, str]:
        return dict(self._display_path_by_case_name)

    async def log_case(self, case_name: str, line: str) -> None:
        file = self._files_by_case_name.get(case_name)
        if file is None:
            raise RuntimeError(f"No log file configured for case {case_name!r}")
        lines = line.splitlines() or [line]
        async with self._lock:
            now = time.perf_counter()
            started_at = self._case_started_at_monotonic.get(case_name)
            if started_at is None:
                self._case_started_at_monotonic[case_name] = now
                elapsed_seconds = 0.0
            else:
                elapsed_seconds = now - started_at

            elapsed_timestamp = _format_elapsed(elapsed_seconds)
            file.write(f"[{elapsed_timestamp}] {lines[0]}\n")
            for continuation in lines[1:]:
                file.write(f"{continuation}\n")
            file.flush()

    def close(self) -> None:
        for file in self._files_by_case_name.values():
            file.close()


def _format_elapsed(total_seconds: float) -> str:
    if total_seconds < 0:
        total_seconds = 0.0
    whole_seconds = int(total_seconds)
    millis = int(round((total_seconds - whole_seconds) * 1000))
    if millis == 1000:
        whole_seconds += 1
        millis = 0
    minutes, seconds = divmod(whole_seconds, 60)
    hours, minutes = divmod(minutes, 60)
    return f"{hours:02d}:{minutes:02d}:{seconds:02d}.{millis:03d}"
