import type { InfiniteData } from "@tanstack/react-query";
import type {
  CanvasesCanvasEvent,
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  CanvasesCanvasRunResult,
  CanvasesCanvasRunState,
} from "@/api-client";
import type { CanvasRunsFilters } from "./useCanvasData";

export type InfiniteRunsPage = {
  runs?: CanvasesCanvasRun[];
  totalCount?: number;
  hasNextPage?: boolean;
  lastTimestamp?: string;
};

export type InfiniteEventsPage = {
  events?: CanvasesCanvasEventWithExecutions[];
  totalCount?: number;
  hasNextPage?: boolean;
  lastTimestamp?: string;
};

const RUN_STATE_ORDER: Record<CanvasesCanvasRunState, number> = {
  STATE_UNKNOWN: 0,
  STATE_STARTED: 1,
  STATE_FINISHED: 2,
};

const EXECUTION_STATE_ORDER: Record<string, number> = {
  STATE_UNKNOWN: 0,
  STATE_PENDING: 1,
  STATE_STARTED: 2,
  STATE_FINISHED: 3,
};

export function parseRunsFiltersFromQueryKey(queryKey: readonly unknown[]): CanvasRunsFilters {
  const infiniteIndex = queryKey.indexOf("infinite");
  if (infiniteIndex === -1) {
    return {};
  }

  const filters: CanvasRunsFilters = {};
  let index = infiniteIndex + 1;

  if (queryKey[index] === "states") {
    index += 1;
    const states: CanvasesCanvasRunState[] = [];
    while (index < queryKey.length && queryKey[index] !== "results") {
      states.push(queryKey[index] as CanvasesCanvasRunState);
      index += 1;
    }
    if (states.length > 0) {
      filters.states = states;
    }
  }

  if (queryKey[index] === "results") {
    index += 1;
    const results: CanvasesCanvasRunResult[] = [];
    while (index < queryKey.length) {
      results.push(queryKey[index] as CanvasesCanvasRunResult);
      index += 1;
    }
    if (results.length > 0) {
      filters.results = results;
    }
  }

  return filters;
}

export function runMatchesFilters(run: CanvasesCanvasRun, filters: CanvasRunsFilters): boolean {
  if (filters.states?.length && (!run.state || !filters.states.includes(run.state))) {
    return false;
  }

  if (filters.results?.length && (!run.result || !filters.results.includes(run.result))) {
    return false;
  }

  return true;
}

function parseTimestamp(value: string | undefined): number {
  if (!value) {
    return 0;
  }

  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? 0 : parsed;
}

function shouldAcceptRunUpdate(existing: CanvasesCanvasRun, incoming: CanvasesCanvasRun): boolean {
  const existingUpdatedAt = parseTimestamp(existing.updatedAt);
  const incomingUpdatedAt = parseTimestamp(incoming.updatedAt);

  if (incomingUpdatedAt < existingUpdatedAt) {
    return false;
  }

  if (incomingUpdatedAt > existingUpdatedAt) {
    return true;
  }

  const existingState = RUN_STATE_ORDER[existing.state ?? "STATE_UNKNOWN"] ?? 0;
  const incomingState = RUN_STATE_ORDER[incoming.state ?? "STATE_UNKNOWN"] ?? 0;
  return incomingState >= existingState;
}

function bumpTotalCountOnAllPages<T extends { totalCount?: number }>(pages: T[], delta: number): void {
  for (const page of pages) {
    page.totalCount = Math.max(0, (page.totalCount ?? 0) + delta);
  }
}

function findRunLocation(pages: InfiniteRunsPage[], runId: string): { pageIndex: number; runIndex: number } | null {
  for (let pageIndex = 0; pageIndex < pages.length; pageIndex += 1) {
    const runIndex = pages[pageIndex].runs?.findIndex((run) => run.id === runId) ?? -1;
    if (runIndex !== -1) {
      return { pageIndex, runIndex };
    }
  }

  return null;
}

function insertRunSorted(runs: CanvasesCanvasRun[], run: CanvasesCanvasRun): CanvasesCanvasRun[] {
  const runCreatedAt = parseTimestamp(run.createdAt);
  const insertIndex = runs.findIndex((existingRun) => parseTimestamp(existingRun.createdAt) < runCreatedAt);
  if (insertIndex === -1) {
    return [...runs, run];
  }

  return [...runs.slice(0, insertIndex), run, ...runs.slice(insertIndex)];
}

export function upsertRunIntoInfiniteData(
  old: InfiniteData<InfiniteRunsPage> | undefined,
  run: CanvasesCanvasRun,
  filters: CanvasRunsFilters,
): InfiniteData<InfiniteRunsPage> | undefined {
  if (!old) {
    return old;
  }

  const pages = old.pages.map((page) => ({
    ...page,
    runs: page.runs ? [...page.runs] : [],
  }));
  const location = findRunLocation(pages, run.id ?? "");
  const matches = runMatchesFilters(run, filters);

  if (matches) {
    if (location) {
      const existing = pages[location.pageIndex].runs![location.runIndex];
      if (!shouldAcceptRunUpdate(existing, run)) {
        return old;
      }

      pages[location.pageIndex].runs![location.runIndex] = run;
      return { ...old, pages };
    }

    if (pages.length === 0) {
      return {
        ...old,
        pages: [{ runs: [run], totalCount: 1, hasNextPage: false }],
      };
    }

    pages[0].runs = insertRunSorted(pages[0].runs ?? [], run);
    bumpTotalCountOnAllPages(pages, 1);
    return { ...old, pages };
  }

  if (!location) {
    return old;
  }

  pages[location.pageIndex].runs!.splice(location.runIndex, 1);
  bumpTotalCountOnAllPages(pages, -1);
  return { ...old, pages };
}

export function canvasEventToEventWithExecutions(event: CanvasesCanvasEvent): CanvasesCanvasEventWithExecutions {
  return {
    id: event.id,
    canvasId: event.canvasId,
    nodeId: event.nodeId,
    channel: event.channel,
    customName: event.customName,
    data: event.data,
    createdAt: event.createdAt,
    executions: [],
  };
}

function findEventLocation(
  pages: InfiniteEventsPage[],
  eventId: string,
): { pageIndex: number; eventIndex: number } | null {
  for (let pageIndex = 0; pageIndex < pages.length; pageIndex += 1) {
    const eventIndex = pages[pageIndex].events?.findIndex((event) => event.id === eventId) ?? -1;
    if (eventIndex !== -1) {
      return { pageIndex, eventIndex };
    }
  }

  return null;
}

export function upsertRootEventIntoInfiniteData(
  old: InfiniteData<InfiniteEventsPage> | undefined,
  event: CanvasesCanvasEvent,
): InfiniteData<InfiniteEventsPage> | undefined {
  if (!old || !event.id) {
    return old;
  }

  const mappedEvent = canvasEventToEventWithExecutions(event);
  const pages = old.pages.map((page) => ({
    ...page,
    events: page.events ? [...page.events] : [],
  }));
  const location = findEventLocation(pages, event.id);

  if (location) {
    const existing = pages[location.pageIndex].events![location.eventIndex];
    pages[location.pageIndex].events![location.eventIndex] = {
      ...mappedEvent,
      executions: existing.executions ?? [],
    };
    return { ...old, pages };
  }

  if (pages.length === 0) {
    return {
      ...old,
      pages: [{ events: [mappedEvent], totalCount: 1, hasNextPage: false }],
    };
  }

  pages[0].events = [mappedEvent, ...(pages[0].events ?? [])];
  bumpTotalCountOnAllPages(pages, 1);
  return { ...old, pages };
}

export function executionToRef(execution: CanvasesCanvasNodeExecution): CanvasesCanvasNodeExecutionRef {
  return {
    id: execution.id,
    nodeId: execution.nodeId,
    state: execution.state,
    result: execution.result,
    resultReason: execution.resultReason,
    resultMessage: execution.resultMessage,
    createdAt: execution.createdAt,
    updatedAt: execution.updatedAt,
  };
}

function shouldAcceptExecutionRefUpdate(
  existing: CanvasesCanvasNodeExecutionRef,
  incoming: CanvasesCanvasNodeExecutionRef,
): boolean {
  const existingUpdatedAt = parseTimestamp(existing.updatedAt);
  const incomingUpdatedAt = parseTimestamp(incoming.updatedAt);

  if (incomingUpdatedAt < existingUpdatedAt) {
    return false;
  }

  if (incomingUpdatedAt > existingUpdatedAt) {
    return true;
  }

  const existingState = EXECUTION_STATE_ORDER[existing.state ?? "STATE_UNKNOWN"] ?? 0;
  const incomingState = EXECUTION_STATE_ORDER[incoming.state ?? "STATE_UNKNOWN"] ?? 0;
  return incomingState >= existingState;
}

export function upsertExecutionRef(
  executions: CanvasesCanvasNodeExecutionRef[],
  incoming: CanvasesCanvasNodeExecutionRef,
) {
  if (!incoming.id) {
    return executions;
  }

  const index = executions.findIndex((execution) => execution.id === incoming.id);
  if (index === -1) {
    return [...executions, incoming];
  }

  if (!shouldAcceptExecutionRefUpdate(executions[index], incoming)) {
    return executions;
  }

  const next = executions.slice();
  next[index] = incoming;
  return next;
}

export function upsertExecutionIntoInfiniteEventsData(
  old: InfiniteData<InfiniteEventsPage> | undefined,
  execution: CanvasesCanvasNodeExecution,
): InfiniteData<InfiniteEventsPage> | undefined {
  const rootEventId = execution.rootEvent?.id;
  if (!old || !rootEventId) {
    return old;
  }

  const executionRef = executionToRef(execution);
  const pages = old.pages.map((page) => ({
    ...page,
    events: page.events ? [...page.events] : [],
  }));

  for (const page of pages) {
    const eventIndex = page.events?.findIndex((event) => event.id === rootEventId) ?? -1;
    if (eventIndex === -1) {
      continue;
    }

    page.events![eventIndex] = {
      ...page.events![eventIndex],
      executions: upsertExecutionRef(page.events![eventIndex].executions ?? [], executionRef),
    };
    return { ...old, pages };
  }

  return old;
}

function findRunByRootEventId(
  pages: InfiniteRunsPage[],
  rootEventId: string,
): { pageIndex: number; runIndex: number } | null {
  for (let pageIndex = 0; pageIndex < pages.length; pageIndex += 1) {
    const runIndex = pages[pageIndex].runs?.findIndex((run) => run.rootEvent?.id === rootEventId) ?? -1;
    if (runIndex !== -1) {
      return { pageIndex, runIndex };
    }
  }

  return null;
}

export function upsertExecutionIntoInfiniteRunsData(
  old: InfiniteData<InfiniteRunsPage> | undefined,
  execution: CanvasesCanvasNodeExecution,
): InfiniteData<InfiniteRunsPage> | undefined {
  const rootEventId = execution.rootEvent?.id;
  if (!old || !rootEventId) {
    return old;
  }

  const executionRef = executionToRef(execution);
  const pages = old.pages.map((page) => ({
    ...page,
    runs: page.runs ? [...page.runs] : [],
  }));
  const location = findRunByRootEventId(pages, rootEventId);
  if (!location) {
    return old;
  }

  const run = pages[location.pageIndex].runs![location.runIndex];
  pages[location.pageIndex].runs![location.runIndex] = {
    ...run,
    executions: upsertExecutionRef(run.executions ?? [], executionRef),
  };
  return { ...old, pages };
}
