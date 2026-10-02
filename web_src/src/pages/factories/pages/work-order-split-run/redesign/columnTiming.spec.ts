import { describe, expect, it } from "bun:test";

import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import type { SplitRunPhase, SplitRunPhaseStatus } from "../splitRunMocks";
import { timingsForConsoleColumns } from "./columnTiming";

const INTAKE_AT = "2026-09-30T10:00:00.000Z";
const IMPLEMENT_AT = "2026-09-30T11:00:00.000Z";
const VERIFY_AT = "2026-09-30T12:30:00.000Z";
const DONE_AT = "2026-09-30T13:00:00.000Z";
const NOW = Date.parse("2026-09-30T13:15:00.000Z");

function phase(id: string, name: string, startedAt?: string, status: SplitRunPhaseStatus = "passed"): SplitRunPhase {
  return {
    id,
    name,
    status,
    duration: "",
    startedAt,
    componentName: name,
    artifacts: [],
    stream: [],
    canvasSteps: [],
  };
}

function timingsFor(...phases: SplitRunPhase[]) {
  return timingsForConsoleColumns({ ...SPLIT_RUN_RUNNING, phases }, NOW);
}

describe("timingsForConsoleColumns", () => {
  it("stamps Intake and Done without dwell, and measures the columns between them", () => {
    const timings = timingsFor(
      phase("backlog", "Backlog", INTAKE_AT),
      phase("implement", "Implement", IMPLEMENT_AT),
      phase("verify", "Verify", VERIFY_AT),
      phase("done-closure", "Done", DONE_AT),
    );

    expect(timings.intake).toEqual({ enteredAt: INTAKE_AT });
    expect(timings.backlog).toEqual({ enteredAt: INTAKE_AT, durationMs: 60 * 60 * 1000 });
    expect(timings.implement).toEqual({ enteredAt: IMPLEMENT_AT, durationMs: 90 * 60 * 1000 });
    expect(timings.verify).toEqual({ enteredAt: VERIFY_AT, durationMs: 30 * 60 * 1000 });
    expect(timings.done).toEqual({ enteredAt: DONE_AT });
  });

  it("uses now for the current column and hides columns the task has not entered", () => {
    const timings = timingsFor(
      phase("backlog", "Backlog", INTAKE_AT),
      phase("implement", "Implement", IMPLEMENT_AT, "running"),
    );

    expect(timings.implement).toEqual({ enteredAt: IMPLEMENT_AT, durationMs: NOW - Date.parse(IMPLEMENT_AT) });
    expect(timings.verify).toBeUndefined();
    expect(timings.done).toBeUndefined();
  });

  it("stamps Verify after Implement passed while the dispatch is still active", () => {
    const implement = { ...phase("implement", "Implement", IMPLEMENT_AT), endedAt: VERIFY_AT };
    const timings = timingsFor(phase("backlog", "Backlog", INTAKE_AT), implement);

    expect(timings.implement).toEqual({
      enteredAt: IMPLEMENT_AT,
      durationMs: Date.parse(VERIFY_AT) - Date.parse(IMPLEMENT_AT),
    });
    expect(timings.verify).toEqual({ enteredAt: VERIFY_AT, durationMs: NOW - Date.parse(VERIFY_AT) });
  });

  it("counts a pending Verify step so hover shows arrival and time so far", () => {
    const timings = timingsFor(
      phase("backlog", "Backlog", INTAKE_AT),
      phase("implement", "Implement", IMPLEMENT_AT),
      phase("verify-1", "Verify", VERIFY_AT, "pending"),
    );

    expect(timings.verify).toEqual({ enteredAt: VERIFY_AT, durationMs: NOW - Date.parse(VERIFY_AT) });
  });

  it("stamps Verify from Implement finish when the task waits in that column", () => {
    const implement = { ...phase("implement", "Implement", IMPLEMENT_AT), endedAt: VERIFY_AT };
    const timings = timingsForConsoleColumns(
      {
        ...SPLIT_RUN_RUNNING,
        lineStatus: "waiting",
        currentStepIndex: 0,
        phases: [phase("backlog", "Backlog", INTAKE_AT), implement],
      },
      NOW,
    );

    expect(timings.implement).toEqual({
      enteredAt: IMPLEMENT_AT,
      durationMs: Date.parse(VERIFY_AT) - Date.parse(IMPLEMENT_AT),
    });
    expect(timings.verify).toEqual({ enteredAt: VERIFY_AT, durationMs: NOW - Date.parse(VERIFY_AT) });
  });

  it("uses the earliest run in a column, including a history run", () => {
    const timings = timingsFor(
      phase("backlog", "Backlog", INTAKE_AT),
      { ...phase("implement-old", "Implement", IMPLEMENT_AT, "failed"), historyRun: true },
      phase("implement", "Implement", VERIFY_AT, "running"),
    );

    expect(timings.implement?.enteredAt).toBe(IMPLEMENT_AT);
    expect(timings.implement?.durationMs).toBe(NOW - Date.parse(IMPLEMENT_AT));
  });

  it("stops dwell at the last ended time when the line is closed", () => {
    const implement = { ...phase("implement", "Implement", IMPLEMENT_AT, "failed"), endedAt: VERIFY_AT };
    const timings = timingsForConsoleColumns(
      {
        ...SPLIT_RUN_RUNNING,
        lineStatus: "failed",
        currentStepIndex: 0,
        phases: [phase("backlog", "Backlog", INTAKE_AT), implement],
      },
      NOW,
    );

    expect(timings.implement).toEqual({
      enteredAt: IMPLEMENT_AT,
      durationMs: Date.parse(VERIFY_AT) - Date.parse(IMPLEMENT_AT),
    });
    expect(timings.verify).toBeUndefined();
  });

  it("maps a named implement step to Implement even when stepIndex is 1", () => {
    const timings = timingsFor(phase("backlog", "Backlog", INTAKE_AT), {
      ...phase("open-pr", "Open pull request", IMPLEMENT_AT, "running"),
      stepIndex: 1,
    });

    expect(timings.implement).toEqual({ enteredAt: IMPLEMENT_AT, durationMs: NOW - Date.parse(IMPLEMENT_AT) });
    expect(timings.verify).toBeUndefined();
  });
});
