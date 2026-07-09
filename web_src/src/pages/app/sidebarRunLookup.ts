import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasRun, CanvasesListRunsResponse } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { upsertRunIntoInfiniteData, type InfiniteRunsPage } from "@/hooks/canvasInfiniteCache";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { findRunIdForSidebarEvent, getSidebarEventExecutionId, getSidebarEventRootEventId } from "@/pages/app/utils";

export type RunLookupIndex = {
  byRootEventId: Map<string, string>;
  byExecutionId: Map<string, string>;
};

export const EMPTY_RUN_LOOKUP_INDEX: RunLookupIndex = {
  byRootEventId: new Map(),
  byExecutionId: new Map(),
};

export function buildRunLookupFingerprint(runs: CanvasesCanvasRun[]): string {
  const parts: string[] = [];

  for (const run of runs) {
    if (!run.id) {
      continue;
    }

    const executionIds = (run.executions ?? [])
      .map((execution) => execution.id)
      .filter(Boolean)
      .sort()
      .join(",");

    parts.push(`${run.id}|${run.rootEvent?.id ?? ""}|${executionIds}`);
  }

  parts.sort();
  return parts.join(";");
}

export function buildRunLookupFingerprintFromSources(options: {
  primaryRuns: CanvasesCanvasRun[];
  pages: Array<CanvasesListRunsResponse | undefined>;
}): string {
  const cachedRuns = collectCachedCanvasRuns(options);
  return buildRunLookupFingerprint(cachedRuns);
}

export function buildRunLookupIndex(runs: CanvasesCanvasRun[]): RunLookupIndex {
  const byRootEventId = new Map<string, string>();
  const byExecutionId = new Map<string, string>();

  for (const run of runs) {
    if (!run.id) {
      continue;
    }

    const rootEventId = run.rootEvent?.id;
    if (rootEventId) {
      byRootEventId.set(rootEventId, run.id);
    }

    for (const execution of run.executions ?? []) {
      if (execution.id) {
        byExecutionId.set(execution.id, run.id);
      }
    }
  }

  return { byRootEventId, byExecutionId };
}

export function findRunIdInLookupIndex(index: RunLookupIndex, event: SidebarEvent): string | null {
  const executionId = getSidebarEventExecutionId(event);
  if (executionId) {
    const runId = index.byExecutionId.get(executionId);
    if (runId) {
      return runId;
    }
  }

  const rootEventId = getSidebarEventRootEventId(event);
  if (rootEventId) {
    return index.byRootEventId.get(rootEventId) ?? null;
  }

  return null;
}

export function findLatestRunIdForNode(runs: CanvasesCanvasRun[], nodeId: string): string | null {
  let latestRunId: string | null = null;
  let latestTimestamp = Number.NEGATIVE_INFINITY;

  for (const run of runs) {
    if (!run.id || !runIncludesNode(run, nodeId)) {
      continue;
    }

    const timestamp = getRunTimestamp(run);
    if (latestRunId === null || timestamp > latestTimestamp) {
      latestRunId = run.id;
      latestTimestamp = timestamp;
    }
  }

  return latestRunId;
}

export function getSidebarEventLookupKey(event: SidebarEvent): string | null {
  return getSidebarEventRootEventId(event) ?? getSidebarEventExecutionId(event) ?? null;
}

export function collectCachedCanvasRuns(options: {
  primaryRuns: CanvasesCanvasRun[];
  pages: Array<CanvasesListRunsResponse | undefined>;
}): CanvasesCanvasRun[] {
  const seen = new Set<string>();
  const runs: CanvasesCanvasRun[] = [];

  const addRun = (run: CanvasesCanvasRun | undefined) => {
    if (!run?.id || seen.has(run.id)) {
      return;
    }

    seen.add(run.id);
    runs.push(run);
  };

  for (const run of options.primaryRuns) {
    addRun(run);
  }

  for (const page of options.pages) {
    for (const run of page?.runs ?? []) {
      addRun(run);
    }
  }

  return runs;
}

export function resolveRunIdsForSidebarEvents(
  index: RunLookupIndex,
  events: SidebarEvent[],
): Map<string, string | null> {
  const runIdsByEventId = new Map<string, string | null>();

  for (const event of events) {
    runIdsByEventId.set(event.id, findRunIdInLookupIndex(index, event));
  }

  return runIdsByEventId;
}

export function seedRunInInfiniteRunsCache(queryClient: QueryClient, canvasId: string, run: CanvasesCanvasRun): void {
  queryClient.setQueryData(canvasKeys.infiniteRuns(canvasId, {}), (current) =>
    upsertRunIntoInfiniteData(current as { pages: InfiniteRunsPage[]; pageParams: unknown[] } | undefined, run, {}),
  );
}

export function findRunInListRunsResponse(
  runs: CanvasesCanvasRun[],
  event: SidebarEvent,
): { runId: string; run: CanvasesCanvasRun } | null {
  const runId = findRunIdForSidebarEvent(runs, event);
  if (!runId) {
    return null;
  }

  const run = runs.find((candidate) => candidate.id === runId);
  if (!run) {
    return { runId, run: { id: runId } };
  }

  return { runId, run };
}

export function shouldContinueRunLookupPagination(options: {
  pageRuns: CanvasesCanvasRun[];
  loadedCount: number;
  response: CanvasesListRunsResponse | undefined;
}): boolean {
  const { pageRuns, loadedCount, response } = options;

  if (pageRuns.length === 0 || !response?.lastTimestamp) {
    return false;
  }

  const totalCount = response.totalCount;
  if (typeof totalCount === "number" && totalCount > 0 && loadedCount >= totalCount) {
    return false;
  }

  if (response.hasNextPage === false) {
    return false;
  }

  return true;
}

function runIncludesNode(run: CanvasesCanvasRun, nodeId: string): boolean {
  if (run.rootEvent?.nodeId === nodeId) {
    return true;
  }

  return (run.executions ?? []).some((execution) => execution.nodeId === nodeId);
}

function getRunTimestamp(run: CanvasesCanvasRun): number {
  return (
    parseTimestamp(run.createdAt) ||
    parseTimestamp(run.updatedAt) ||
    parseTimestamp(run.finishedAt) ||
    Number.NEGATIVE_INFINITY
  );
}

function parseTimestamp(value: string | undefined): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
}
