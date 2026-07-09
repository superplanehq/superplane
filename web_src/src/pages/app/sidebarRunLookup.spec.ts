import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import {
  buildRunLookupIndex,
  buildRunLookupFingerprint,
  collectCachedCanvasRuns,
  findLatestRunIdForNode,
  findRunIdInLookupIndex,
  getSidebarEventLookupKey,
  resolveRunIdsForSidebarEvents,
  shouldContinueRunLookupPagination,
} from "./sidebarRunLookup";

const executionEvent = {
  id: "execution-1",
  title: "Execution",
  state: "success",
  isOpen: false,
  kind: "execution",
  executionId: "execution-1",
  triggerEventId: "root-1",
} satisfies SidebarEvent;

const triggerEvent = {
  id: "root-1",
  title: "Trigger",
  state: "success",
  isOpen: false,
  kind: "trigger",
} satisfies SidebarEvent;

const runs: CanvasesCanvasRun[] = [
  {
    id: "run-1",
    rootEvent: { id: "root-1", nodeId: "trigger-1" },
    executions: [{ id: "execution-1", nodeId: "node-1" }],
  },
];

describe("sidebarRunLookup", () => {
  it("builds lookup indexes for root events and executions", () => {
    const index = buildRunLookupIndex(runs);

    expect(findRunIdInLookupIndex(index, triggerEvent)).toBe("run-1");
    expect(findRunIdInLookupIndex(index, executionEvent)).toBe("run-1");
  });

  it("dedupes cached runs from multiple pages", () => {
    const collected = collectCachedCanvasRuns({
      primaryRuns: runs,
      pages: [{ runs }, { runs: [{ id: "run-2", rootEvent: { id: "root-2" } }] }],
    });

    expect(collected).toHaveLength(2);
  });

  it("resolves run ids for visible sidebar events in one pass", () => {
    const index = buildRunLookupIndex(runs);
    const resolved = resolveRunIdsForSidebarEvents(index, [triggerEvent, executionEvent]);

    expect(resolved.get("root-1")).toBe("run-1");
    expect(resolved.get("execution-1")).toBe("run-1");
  });

  it("uses a stable lookup key for fetch dedupe", () => {
    expect(getSidebarEventLookupKey(triggerEvent)).toBe("root-1");
    expect(getSidebarEventLookupKey(executionEvent)).toBe("root-1");
  });

  it("finds the latest cached run that includes a node", () => {
    const latestRunId = findLatestRunIdForNode(
      [
        {
          id: "older-run",
          rootEvent: { id: "older-root", nodeId: "trigger-1" },
          createdAt: "2026-07-07T10:00:00Z",
        },
        {
          id: "latest-run",
          executions: [{ id: "execution-2", nodeId: "action-1" }],
          createdAt: "2026-07-07T10:05:00Z",
        },
        {
          id: "other-node-run",
          executions: [{ id: "execution-3", nodeId: "action-2" }],
          createdAt: "2026-07-07T10:10:00Z",
        },
      ],
      "action-1",
    );

    expect(latestRunId).toBe("latest-run");
  });

  it("matches trigger nodes through the run root event", () => {
    expect(findLatestRunIdForNode(runs, "trigger-1")).toBe("run-1");
  });

  it("ignores run state-only updates in the lookup fingerprint", () => {
    const started = [{ id: "run-1", rootEvent: { id: "root-1" } }] satisfies CanvasesCanvasRun[];
    const finished = [
      { id: "run-1", state: "STATE_FINISHED" as const, rootEvent: { id: "root-1" } },
    ] satisfies CanvasesCanvasRun[];

    expect(buildRunLookupFingerprint(started)).toBe(buildRunLookupFingerprint(finished));
  });

  it("continues pagination while the API reports more pages", () => {
    expect(
      shouldContinueRunLookupPagination({
        pageRuns: [{ id: "run-1" }],
        loadedCount: 25,
        response: {
          totalCount: 100,
          hasNextPage: true,
          lastTimestamp: "2026-02-06T15:00:00.000Z",
        },
      }),
    ).toBe(true);
  });

  it("stops pagination when all runs are loaded", () => {
    expect(
      shouldContinueRunLookupPagination({
        pageRuns: [{ id: "run-1" }],
        loadedCount: 100,
        response: {
          totalCount: 100,
          hasNextPage: true,
          lastTimestamp: "2026-02-06T15:00:00.000Z",
        },
      }),
    ).toBe(false);
  });

  it("stops pagination when the API reports no next page", () => {
    expect(
      shouldContinueRunLookupPagination({
        pageRuns: [{ id: "run-1" }],
        loadedCount: 25,
        response: {
          totalCount: 100,
          hasNextPage: false,
          lastTimestamp: "2026-02-06T15:00:00.000Z",
        },
      }),
    ).toBe(false);
  });
});
