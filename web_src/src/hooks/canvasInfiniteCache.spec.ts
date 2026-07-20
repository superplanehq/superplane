import type { InfiniteData } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import {
  executionToRef,
  parseRunsFiltersFromQueryKey,
  runMatchesFilters,
  upsertExecutionIntoInfiniteRunsData,
  upsertRunIntoDescribeRunData,
  upsertRunIntoInfiniteData,
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

  it("preserves existing run details when a live update omits them", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        rootEvent: { id: "event-1", nodeId: "trigger-1", createdAt: "2026-06-01T12:00:00.000Z" },
        executions: [
          {
            id: "execution-1",
            nodeId: "node-1",
            state: "STATE_FINISHED",
            updatedAt: "2026-06-01T12:00:30.000Z",
          },
        ],
        queueItems: [
          {
            id: "queue-item-1",
            nodeId: "node-1",
            createdAt: "2026-06-01T12:00:45.000Z",
          },
        ],
        updatedAt: "2026-06-01T12:00:30.000Z",
      }),
    ]);
    const incoming: CanvasesCanvasRun = {
      id: "run-1",
      canvasId: "canvas-1",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    };

    const next = upsertRunIntoInfiniteData(old, incoming, {})!;
    const updatedRun = next.pages[0]?.runs?.[0];

    expect(updatedRun?.state).toBe("STATE_FINISHED");
    expect(updatedRun?.result).toBe("RESULT_PASSED");
    expect(updatedRun?.rootEvent?.id).toBe("event-1");
    expect(updatedRun?.executions?.map((execution) => execution.id)).toEqual(["execution-1"]);
    expect(updatedRun?.queueItems?.map((queueItem) => queueItem.id)).toEqual(["queue-item-1"]);
  });

  it("clears existing queue items when a refreshed run has none", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        queueItems: [
          {
            id: "queue-item-1",
            nodeId: "node-1",
            createdAt: "2026-06-01T12:00:45.000Z",
          },
        ],
        updatedAt: "2026-06-01T12:00:30.000Z",
      }),
    ]);
    const incoming = makeRun({
      queueItems: [],
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoInfiniteData(old, incoming, {})!;

    expect(next.pages[0]?.runs?.[0]?.queueItems).toEqual([]);
  });

  it("sorts newly inserted live runs by updatedAt when createdAt is unavailable", () => {
    const existing = makeRun({
      id: "run-old",
      createdAt: "2026-06-01T11:00:00.000Z",
      updatedAt: "2026-06-01T11:00:00.000Z",
    });
    const incoming = makeRun({
      id: "run-new",
      createdAt: undefined,
      updatedAt: "2026-06-01T12:00:00.000Z",
    });
    const old = makeInfiniteRunsData([existing], 1);

    const next = upsertRunIntoInfiniteData(old, incoming, {})!;

    expect(next.pages[0]?.runs?.map((run) => run.id)).toEqual(["run-new", "run-old"]);
    expect(next.pages[0]?.totalCount).toBe(2);
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

describe("upsertRunIntoDescribeRunData", () => {
  it("updates describe-run cache when the incoming payload is newer", () => {
    const current = {
      run: makeRun({
        state: "STATE_STARTED",
        queueItems: [
          {
            id: "queue-item-1",
            nodeId: "node-1",
            createdAt: "2026-06-01T12:00:45.000Z",
          },
        ],
        updatedAt: "2026-06-01T12:00:00.000Z",
      }),
    };
    const incoming = makeRun({
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoDescribeRunData(current, incoming);

    expect(next?.run?.state).toBe("STATE_FINISHED");
    expect(next?.run?.result).toBe("RESULT_PASSED");
    expect(next?.run?.queueItems?.map((queueItem) => queueItem.id)).toEqual(["queue-item-1"]);
  });

  it("clears describe-run queue items when the incoming payload has none", () => {
    const current = {
      run: makeRun({
        queueItems: [
          {
            id: "queue-item-1",
            nodeId: "node-1",
            createdAt: "2026-06-01T12:00:45.000Z",
          },
        ],
        updatedAt: "2026-06-01T12:00:00.000Z",
      }),
    };
    const incoming = makeRun({
      queueItems: [],
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoDescribeRunData(current, incoming);

    expect(next?.run?.queueItems).toEqual([]);
  });

  it("rejects stale run_started updates after run_finished", () => {
    const current = {
      run: makeRun({
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: "2026-06-01T12:01:00.000Z",
      }),
    };
    const incoming = makeRun({
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const next = upsertRunIntoDescribeRunData(current, incoming);

    expect(next).toBe(current);
    expect(next?.run?.state).toBe("STATE_FINISHED");
  });

  it("seeds describe-run cache when no entry exists yet", () => {
    const incoming = makeRun({
      state: "STATE_STARTED",
      updatedAt: "2026-06-01T12:00:00.000Z",
    });

    const next = upsertRunIntoDescribeRunData(undefined, incoming);

    expect(next).toEqual({ run: incoming });
  });
});

describe("execution cache patching", () => {
  it("maps executions to refs and upserts them into matching runs", () => {
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

    const runsOld = makeInfiniteRunsData([makeRun({ rootEvent: { id: "event-1", nodeId: "trigger-1" } })]);
    const runsNext = upsertExecutionIntoInfiniteRunsData(runsOld, execution)!;
    expect(runsNext.pages[0]?.runs?.[0]?.executions?.[0]?.state).toBe("STATE_FINISHED");
  });

  it("ignores stale execution updates", () => {
    const old = makeInfiniteRunsData([
      makeRun({
        rootEvent: { id: "event-1", nodeId: "trigger-1" },
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

    const next = upsertExecutionIntoInfiniteRunsData(old, incoming);

    expect(next?.pages[0]?.runs?.[0]?.executions?.[0]?.state).toBe("STATE_FINISHED");
  });
});
