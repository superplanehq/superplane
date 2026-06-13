import type { InfiniteData } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type {
  CanvasesCanvasEvent,
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
} from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import {
  executionToRef,
  parseRunsFiltersFromQueryKey,
  runMatchesFilters,
  upsertExecutionIntoInfiniteEventsData,
  upsertExecutionIntoInfiniteRunsData,
  upsertRootEventIntoInfiniteData,
  upsertRunIntoInfiniteData,
  type InfiniteEventsPage,
  type InfiniteRunsPage,
} from "./canvasInfiniteCache";

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "canvas-1",
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    createdAt: "2026-06-01T12:00:00.000Z",
    updatedAt: "2026-06-01T12:00:00.000Z",
    rootEvent: { id: "event-1", nodeId: "trigger-1", createdAt: "2026-06-01T12:00:00.000Z" },
    executions: [],
    ...overrides,
  };
}

function makeInfiniteRunsData(runs: CanvasesCanvasRun[], totalCount = runs.length): InfiniteData<InfiniteRunsPage> {
  return {
    pages: [{ runs, totalCount, hasNextPage: false }],
    pageParams: [undefined],
  };
}

function makeEvent(overrides: Partial<CanvasesCanvasEventWithExecutions> = {}): CanvasesCanvasEventWithExecutions {
  return {
    id: "event-1",
    nodeId: "trigger-1",
    createdAt: "2026-06-01T12:00:00.000Z",
    executions: [],
    ...overrides,
  };
}

function makeInfiniteEventsData(
  events: CanvasesCanvasEventWithExecutions[],
  totalCount = events.length,
): InfiniteData<InfiniteEventsPage> {
  return {
    pages: [{ events, totalCount, hasNextPage: false }],
    pageParams: [undefined],
  };
}

describe("parseRunsFiltersFromQueryKey", () => {
  it("parses state and result filters from infinite run query keys", () => {
    const queryKey = canvasKeys.infiniteRuns("canvas-1", {
      states: ["STATE_STARTED"],
      results: ["RESULT_FAILED", "RESULT_CANCELLED"],
    });

    expect(parseRunsFiltersFromQueryKey(queryKey)).toEqual({
      states: ["STATE_STARTED"],
      results: ["RESULT_FAILED", "RESULT_CANCELLED"],
    });
  });

  it("returns empty filters for unfiltered infinite run query keys", () => {
    expect(parseRunsFiltersFromQueryKey(canvasKeys.infiniteRuns("canvas-1"))).toEqual({});
  });
});

describe("runMatchesFilters", () => {
  it("matches runs against state and result filters", () => {
    const run = makeRun({ state: "STATE_FINISHED", result: "RESULT_PASSED" });

    expect(runMatchesFilters(run, {})).toBe(true);
    expect(runMatchesFilters(run, { states: ["STATE_FINISHED"] })).toBe(true);
    expect(runMatchesFilters(run, { results: ["RESULT_PASSED"] })).toBe(true);
    expect(runMatchesFilters(run, { states: ["STATE_STARTED"] })).toBe(false);
    expect(runMatchesFilters(run, { results: ["RESULT_FAILED"] })).toBe(false);
  });
});

describe("upsertRunIntoInfiniteData", () => {
  it("inserts a matching run at the top and bumps totalCount", () => {
    const existing = makeRun({ id: "run-old", createdAt: "2026-06-01T11:00:00.000Z" });
    const incoming = makeRun({ id: "run-new", createdAt: "2026-06-01T12:00:00.000Z" });
    const old = makeInfiniteRunsData([existing], 1);

    const next = upsertRunIntoInfiniteData(old, incoming, {})!;

    expect(next.pages[0]?.runs?.map((run) => run.id)).toEqual(["run-new", "run-old"]);
    expect(next.pages[0]?.totalCount).toBe(2);
  });

  it("updates an existing run when the incoming payload is newer", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        state: "STATE_STARTED",
        updatedAt: "2026-06-01T12:00:00.000Z",
      }),
    ]);
    const incoming = makeRun({
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoInfiniteData(old, incoming, {})!;

    expect(next.pages[0]?.runs?.[0]?.state).toBe("STATE_FINISHED");
    expect(next.pages[0]?.runs?.[0]?.result).toBe("RESULT_PASSED");
    expect(next.pages[0]?.totalCount).toBe(1);
  });

  it("rejects stale run_started updates after run_finished", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: "2026-06-01T12:01:00.000Z",
      }),
    ]);
    const incoming = makeRun({
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoInfiniteData(old, incoming, {});

    expect(next).toBe(old);
  });

  it("removes a run that no longer matches the active filter", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        state: "STATE_STARTED",
        updatedAt: "2026-06-01T12:00:00.000Z",
      }),
    ]);
    const incoming = makeRun({
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoInfiniteData(old, incoming, { states: ["STATE_STARTED"] })!;

    expect(next.pages[0]?.runs).toEqual([]);
    expect(next.pages[0]?.totalCount).toBe(0);
  });
});

describe("upsertRootEventIntoInfiniteData", () => {
  it("maps root events and inserts them at the top of page 0", () => {
    const old = makeInfiniteEventsData([makeEvent({ id: "event-old", createdAt: "2026-06-01T11:00:00.000Z" })], 1);
    const incoming: CanvasesCanvasEvent = {
      id: "event-new",
      nodeId: "trigger-1",
      root: true,
      createdAt: "2026-06-01T12:00:00.000Z",
      customName: "Deploy",
    };

    const next = upsertRootEventIntoInfiniteData(old, incoming)!;

    expect(next.pages[0]?.events?.[0]).toMatchObject({
      id: "event-new",
      customName: "Deploy",
      executions: [],
    });
    expect(next.pages[0]?.totalCount).toBe(2);
  });

  it("preserves existing executions when the same root event is upserted again", () => {
    const old = makeInfiniteEventsData([
      makeEvent({
        executions: [{ id: "execution-1", nodeId: "node-1", state: "STATE_STARTED" }],
      }),
    ]);
    const incoming: CanvasesCanvasEvent = {
      id: "event-1",
      nodeId: "trigger-1",
      root: true,
      createdAt: "2026-06-01T12:00:00.000Z",
    };

    const next = upsertRootEventIntoInfiniteData(old, incoming)!;

    expect(next.pages[0]?.events?.[0]?.executions).toEqual([
      { id: "execution-1", nodeId: "node-1", state: "STATE_STARTED" },
    ]);
  });
});

describe("execution cache patching", () => {
  it("maps executions to refs and upserts them into matching events and runs", () => {
    const execution: CanvasesCanvasNodeExecution = {
      id: "execution-1",
      nodeId: "node-1",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: "2026-06-01T12:00:00.000Z",
      updatedAt: "2026-06-01T12:01:00.000Z",
      rootEvent: { id: "event-1", nodeId: "trigger-1", createdAt: "2026-06-01T12:00:00.000Z" },
    };

    expect(executionToRef(execution)).toEqual({
      id: "execution-1",
      nodeId: "node-1",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: undefined,
      resultMessage: undefined,
      createdAt: "2026-06-01T12:00:00.000Z",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const eventsOld = makeInfiniteEventsData([makeEvent({ id: "event-1" })]);
    const eventsNext = upsertExecutionIntoInfiniteEventsData(eventsOld, execution)!;
    expect(eventsNext.pages[0]?.events?.[0]?.executions?.[0]?.state).toBe("STATE_FINISHED");

    const runsOld = makeInfiniteRunsData([makeRun({ rootEvent: { id: "event-1", nodeId: "trigger-1" } })]);
    const runsNext = upsertExecutionIntoInfiniteRunsData(runsOld, execution)!;
    expect(runsNext.pages[0]?.runs?.[0]?.executions?.[0]?.state).toBe("STATE_FINISHED");
  });

  it("ignores stale execution updates", () => {
    const old = makeInfiniteEventsData([
      makeEvent({
        executions: [
          {
            id: "execution-1",
            nodeId: "node-1",
            state: "STATE_FINISHED",
            updatedAt: "2026-06-01T12:02:00.000Z",
          },
        ],
      }),
    ]);
    const incoming: CanvasesCanvasNodeExecution = {
      id: "execution-1",
      nodeId: "node-1",
      state: "STATE_STARTED",
      updatedAt: "2026-06-01T12:01:00.000Z",
      rootEvent: { id: "event-1", nodeId: "trigger-1" },
    };

    const next = upsertExecutionIntoInfiniteEventsData(old, incoming);

    expect(next?.pages[0]?.events?.[0]?.executions?.[0]?.state).toBe("STATE_FINISHED");
  });
});
