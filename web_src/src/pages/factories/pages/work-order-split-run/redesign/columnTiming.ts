import type { SplitRunFixture, SplitRunPhase, SplitRunPhaseStatus } from "../splitRunMocks";
import { consoleColumnIdForStage, isConsoleCreationStage, type ConsoleColumnId } from "./automationsViewModel";

export type ColumnTimingId = "intake" | ConsoleColumnId;

export type ColumnTiming = {
  enteredAt: string;
  /** Dwell in this column. Absent for Intake and Done. */
  durationMs?: number;
};

const COLUMN_ORDER: ConsoleColumnId[] = ["backlog", "implement", "verify", "done"];

const NEXT_COLUMN: Record<ConsoleColumnId, ConsoleColumnId | undefined> = {
  backlog: "implement",
  implement: "verify",
  verify: "done",
  done: undefined,
};

/** Arrival and dwell for each reached console column. */
export function timingsForConsoleColumns(
  fixture: SplitRunFixture,
  nowMs = Date.now(),
): Partial<Record<ColumnTimingId, ColumnTiming>> {
  const enteredMs = enteredMsByColumn(fixture);
  const intakeEntered = enteredMs.intake;
  const result: Partial<Record<ColumnTimingId, ColumnTiming>> = {};
  if (intakeEntered != null) {
    result.intake = { enteredAt: new Date(intakeEntered).toISOString() };
  }
  for (const column of COLUMN_ORDER) {
    const entered = enteredMs[column];
    if (entered == null) {
      continue;
    }
    const timing: ColumnTiming = { enteredAt: new Date(entered).toISOString() };
    if (column !== "done") {
      const next = NEXT_COLUMN[column];
      const nextEntered = next ? enteredMs[next] : undefined;
      const end = nextEntered ?? dwellEndMs(column, fixture, nowMs);
      const durationMs = Math.max(0, end - entered);
      if (durationMs > 0) {
        timing.durationMs = durationMs;
      }
    }
    result[column] = timing;
  }
  return result;
}

function enteredMsByColumn(fixture: SplitRunFixture): Partial<Record<ColumnTimingId, number>> {
  const entered: Partial<Record<ColumnTimingId, number>> = {};
  const creation = fixture.phases.find((phase) => isConsoleCreationStage(phase) && phase.status !== "pending");
  const intakeEntered = parseTime(creation?.startedAt);
  if (intakeEntered != null) {
    entered.intake = intakeEntered;
    entered.backlog = intakeEntered;
  }
  for (const phase of fixture.phases) {
    const started = parseTime(phase.startedAt);
    if (started == null) {
      continue;
    }
    const column = timingColumnForPhase(phase);
    const current = entered[column];
    if (current == null || started < current) {
      entered[column] = started;
    }
  }
  inferVerifyFromImplement(entered, fixture);
  return entered;
}

/**
 * Board Verify is often a wait after Implement passed, before a Verify
 * automation starts. Use Implement's finish as arrival in that case.
 */
function inferVerifyFromImplement(entered: Partial<Record<ColumnTimingId, number>>, fixture: SplitRunFixture): void {
  if (entered.done != null) {
    return;
  }
  const implementEnd = endedMsForColumn(fixture.phases, "implement");
  const leftImplement = fixture.currentStepIndex >= 1 || implementPassedAndIdle(fixture.phases);
  if (implementEnd != null) {
    if (entered.verify == null && !leftImplement) {
      return;
    }
    if (entered.verify == null || implementEnd < entered.verify) {
      entered.verify = implementEnd;
    }
    return;
  }
  if (entered.verify == null && leftImplement) {
    entered.verify = entered.implement;
  }
}

function implementPassedAndIdle(phases: SplitRunPhase[]): boolean {
  const implementPhases = phases.filter((phase) => timingColumnForPhase(phase) === "implement");
  if (implementPhases.length === 0) {
    return false;
  }
  if (implementPhases.some((phase) => phase.status === "running" || phase.status === "pending")) {
    return false;
  }
  return implementPhases.some((phase) => phase.status === "passed");
}

function endedMsForColumn(phases: SplitRunPhase[], column: ConsoleColumnId): number | undefined {
  const times = phases
    .filter((phase) => timingColumnForPhase(phase) === column)
    .map((phase) => parseTime(phase.endedAt))
    .filter((value): value is number => value != null);
  return times.length ? Math.max(...times) : undefined;
}

function dwellEndMs(column: ConsoleColumnId, fixture: SplitRunFixture, nowMs: number): number {
  if (isOpenLineStatus(fixture.lineStatus)) {
    return nowMs;
  }
  return endedMsForColumn(fixture.phases, column) ?? latestEndedMs(fixture.phases) ?? nowMs;
}

function isOpenLineStatus(status: SplitRunPhaseStatus): boolean {
  return status === "running" || status === "waiting" || status === "pending";
}

function latestEndedMs(phases: SplitRunPhase[]): number | undefined {
  const times = phases.map((phase) => parseTime(phase.endedAt)).filter((value): value is number => value != null);
  return times.length ? Math.max(...times) : undefined;
}

function timingColumnForPhase(phase: SplitRunPhase): ConsoleColumnId {
  if (isConsoleCreationStage(phase)) {
    return "backlog";
  }
  return consoleColumnIdForStage({
    id: phase.id,
    name: phase.name,
    columnKey: phase.columnKey,
    appId: phase.appId,
    componentName: phase.componentName,
    pullRequestActivity: phase.pullRequestActivity,
  });
}

function parseTime(value?: string): number | undefined {
  const ms = Date.parse(value ?? "");
  return Number.isFinite(ms) ? ms : undefined;
}
