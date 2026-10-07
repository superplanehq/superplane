import {
  canvasesDescribeRun,
  canvasesListRuns,
  canvasesReemitTriggerEvent,
  type CanvasesCanvasRun,
} from "@/api-client";
import { invalidateFactoryWorkOrderQueries } from "@/hooks/useFactoryWebsocket";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { findRunInListRunsResponse } from "@/pages/app/sidebarRunLookup";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { getRunStatus } from "@/ui/Runs/runPresentation";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

import type { SplitRunFixture, SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";

const RERUN_FAILURE_TOAST = "Failed to restart run";
const RERUN_SUCCESS_TOAST = "Run restarted";
const RUN_LOOKUP_PAGE_LIMIT = 25;
const RERUN_LOOKUP_DELAY_MS = 500;

export type StageAutomationRerunTarget = {
  sourcePhaseId: string;
  appId: string;
  runId: string;
};

type StartedStageAutomationRerun = StageAutomationRerunTarget & {
  eventId: string;
  triggerNodeId: string;
  createdRunId?: string;
  status: SplitRunPhaseStatus;
  startedAt: string;
};

type ReemittedStageAutomation = {
  canvasId: string;
  triggerNodeId: string;
  eventId: string;
};

export function useStageAutomationRerun(
  organizationId?: string,
  factoryId?: string,
  orderId?: string,
  knownRunIds: readonly string[] = [],
) {
  const queryClient = useQueryClient();
  const inFlight = useRef(false);
  const [pending, setPending] = useState(false);
  const [attempts, setAttempts] = useState<StartedStageAutomationRerun[]>([]);
  const watchers = useRef(new Map<string, AbortController>());
  const knownRunIdsRef = useRef(knownRunIds);
  const attemptsRef = useRef(attempts);
  const isMounted = useRef(true);
  knownRunIdsRef.current = knownRunIds;
  attemptsRef.current = attempts;

  useEffect(() => {
    const known = new Set(knownRunIds);
    for (const attempt of attemptsRef.current) {
      if (attempt.createdRunId && known.has(attempt.createdRunId)) {
        watchers.current.get(attempt.eventId)?.abort();
      }
    }
    setAttempts((current) => dropKnownReruns(current, known));
  }, [knownRunIds]);

  useEffect(() => {
    isMounted.current = true;
    const controllers = watchers.current;
    return () => {
      isMounted.current = false;
      for (const controller of controllers.values()) {
        controller.abort();
      }
      controllers.clear();
    };
  }, []);

  const rerun = useCallback(
    async (target: StageAutomationRerunTarget) => {
      if (inFlight.current) {
        return;
      }
      inFlight.current = true;
      setPending(true);
      try {
        const reemitted = await reemitStageAutomation(target, organizationId);
        await refreshStageAutomationQueries(queryClient, organizationId, factoryId, orderId);
        if (!isMounted.current) {
          return;
        }
        setAttempts((current) => appendAcceptedRerun(current, target, reemitted));
        watchAcceptedRerun({
          canvasId: target.appId,
          organizationId,
          reemitted,
          knownRunIds: knownRunIdsRef,
          watchers,
          setAttempts,
          isMounted,
        });
        showSuccessToast(RERUN_SUCCESS_TOAST);
      } catch (error) {
        console.error(RERUN_FAILURE_TOAST, error);
        showErrorToast(RERUN_FAILURE_TOAST);
      } finally {
        inFlight.current = false;
        if (isMounted.current) {
          setPending(false);
        }
      }
    },
    [factoryId, orderId, organizationId, queryClient],
  );

  return { rerun, pending, attempts };
}

export function fixtureWithStartedReruns(
  fixture: SplitRunFixture,
  attempts: StartedStageAutomationRerun[],
): SplitRunFixture {
  const phases = phasesWithStartedReruns(fixture.phases, attempts);
  return phases === fixture.phases ? fixture : { ...fixture, phases };
}

async function reemitStageAutomation(
  target: StageAutomationRerunTarget,
  organizationId: string | undefined,
): Promise<ReemittedStageAutomation> {
  const described = await canvasesDescribeRun(
    withOrganizationHeader({
      organizationId,
      path: { canvasId: target.appId, runId: target.runId },
    }),
  );
  const rootEvent = described.data?.run?.rootEvent;
  if (!rootEvent?.nodeId || !rootEvent.id) {
    throw new Error("Run root event is missing");
  }

  const response = await canvasesReemitTriggerEvent(
    withOrganizationHeader({
      organizationId,
      path: {
        canvasId: target.appId,
        nodeId: rootEvent.nodeId,
        eventId: rootEvent.id,
      },
    }),
  );
  const eventId = response.data?.eventId;
  if (!eventId) {
    throw new Error("Restarted event is missing");
  }

  return { canvasId: target.appId, triggerNodeId: rootEvent.nodeId, eventId };
}

function watchAcceptedRerun({
  canvasId,
  organizationId,
  reemitted,
  knownRunIds,
  watchers,
  setAttempts,
  isMounted,
}: {
  canvasId: string;
  organizationId: string | undefined;
  reemitted: ReemittedStageAutomation;
  knownRunIds: { current: readonly string[] };
  watchers: { current: Map<string, AbortController> };
  setAttempts: (update: (current: StartedStageAutomationRerun[]) => StartedStageAutomationRerun[]) => void;
  isMounted: { current: boolean };
}) {
  if (!isMounted.current) {
    return;
  }
  const controller = new AbortController();
  watchers.current.get(reemitted.eventId)?.abort();
  watchers.current.set(reemitted.eventId, controller);
  void followCreatedRun({ canvasId, organizationId, reemitted, knownRunIds, controller, watchers, setAttempts });
}

async function followCreatedRun({
  canvasId,
  organizationId,
  reemitted,
  knownRunIds,
  controller,
  watchers,
  setAttempts,
}: {
  canvasId: string;
  organizationId: string | undefined;
  reemitted: ReemittedStageAutomation;
  knownRunIds: { current: readonly string[] };
  controller: AbortController;
  watchers: { current: Map<string, AbortController> };
  setAttempts: (update: (current: StartedStageAutomationRerun[]) => StartedStageAutomationRerun[]) => void;
}) {
  const lookupEvent = createdRunLookupEvent(reemitted);
  let createdRunId: string | undefined;
  try {
    while (!controller.signal.aborted) {
      const run = createdRunId
        ? await readRunById(canvasId, organizationId, createdRunId)
        : await readCreatedRun(canvasId, organizationId, lookupEvent);
      if (controller.signal.aborted) {
        return;
      }
      if (run?.id) {
        createdRunId = run.id;
      }
      if (createdRunId && knownRunIds.current.includes(createdRunId)) {
        setAttempts((current) => current.filter((attempt) => attempt.eventId !== reemitted.eventId));
        return;
      }
      if (run?.id) {
        const status = phaseStatusFromCanvasRun(run);
        setAttempts((current) => applyCreatedRun(current, reemitted.eventId, run.id ?? "", status));
        if (isTerminalPhaseStatus(status)) {
          return;
        }
      }
      await wait(RERUN_LOOKUP_DELAY_MS, controller.signal);
    }
  } finally {
    if (watchers.current.get(reemitted.eventId) === controller) {
      watchers.current.delete(reemitted.eventId);
    }
  }
}

async function readRunById(
  canvasId: string,
  organizationId: string | undefined,
  runId: string,
): Promise<CanvasesCanvasRun | undefined> {
  try {
    const response = await canvasesDescribeRun(
      withOrganizationHeader({
        organizationId,
        path: { canvasId, runId },
      }),
    );
    return response.data?.run;
  } catch (error) {
    console.error("Failed to read restarted run", error);
    return undefined;
  }
}

async function readCreatedRun(
  canvasId: string,
  organizationId: string | undefined,
  event: SidebarEvent,
): Promise<CanvasesCanvasRun | undefined> {
  try {
    const response = await canvasesListRuns(
      withOrganizationHeader({
        organizationId,
        path: { canvasId },
        query: { limit: RUN_LOOKUP_PAGE_LIMIT },
      }),
    );
    return findRunInListRunsResponse(response.data?.runs ?? [], event)?.run;
  } catch (error) {
    console.error("Failed to find restarted run", error);
    return undefined;
  }
}

function createdRunLookupEvent(reemitted: ReemittedStageAutomation): SidebarEvent {
  return {
    id: reemitted.eventId,
    title: "Re-emitted event",
    state: "running",
    isOpen: false,
    nodeId: reemitted.triggerNodeId,
    triggerEventId: reemitted.eventId,
    kind: "trigger",
  };
}

function phaseStatusFromCanvasRun(run: CanvasesCanvasRun): SplitRunPhaseStatus {
  const status = getRunStatus(run);
  if (status === "failed" || status === "cancelled" || status === "passed") {
    return status;
  }
  return "running";
}

function isTerminalPhaseStatus(status: SplitRunPhaseStatus): boolean {
  return status === "failed" || status === "cancelled" || status === "passed";
}

function wait(ms: number, signal: AbortSignal): Promise<void> {
  if (signal.aborted) {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        resolve();
      },
      { once: true },
    );
  });
}

async function refreshStageAutomationQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string | undefined,
  factoryId: string | undefined,
  orderId: string | undefined,
) {
  await queryClient.invalidateQueries({ queryKey: ["canvases"] });
  if (!organizationId || !factoryId) {
    return;
  }
  invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
}

function appendAcceptedRerun(
  attempts: StartedStageAutomationRerun[],
  target: StageAutomationRerunTarget,
  reemitted: ReemittedStageAutomation,
): StartedStageAutomationRerun[] {
  if (attempts.some((attempt) => attempt.eventId === reemitted.eventId)) {
    return attempts;
  }
  return [
    ...attempts,
    {
      ...target,
      sourcePhaseId: stableSourcePhaseId(target.sourcePhaseId, attempts),
      eventId: reemitted.eventId,
      triggerNodeId: reemitted.triggerNodeId,
      status: "running",
      startedAt: new Date().toISOString(),
    },
  ];
}

function stableSourcePhaseId(sourcePhaseId: string, attempts: StartedStageAutomationRerun[]): string {
  const seen = new Set<string>();
  let current = sourcePhaseId;
  while (!seen.has(current)) {
    seen.add(current);
    const parent = attempts.find((attempt) => temporaryRerunPhaseId(attempt.eventId) === current);
    if (!parent || parent.sourcePhaseId === current) {
      return current;
    }
    current = parent.sourcePhaseId;
  }
  return sourcePhaseId;
}

function temporaryRerunPhaseId(eventId: string): string {
  return `rerun-${eventId}`;
}

function applyCreatedRun(
  attempts: StartedStageAutomationRerun[],
  eventId: string,
  createdRunId: string,
  status: SplitRunPhaseStatus,
): StartedStageAutomationRerun[] {
  let changed = false;
  const next = attempts.map((attempt) => {
    if (attempt.eventId !== eventId) {
      return attempt;
    }
    if (attempt.createdRunId === createdRunId && attempt.status === status) {
      return attempt;
    }
    changed = true;
    return { ...attempt, createdRunId, status };
  });
  return changed ? next : attempts;
}

function dropKnownReruns(
  attempts: StartedStageAutomationRerun[],
  knownRunIds: ReadonlySet<string>,
): StartedStageAutomationRerun[] {
  const next = attempts.filter((attempt) => !attempt.createdRunId || !knownRunIds.has(attempt.createdRunId));
  return next.length === attempts.length ? attempts : next;
}

function phasesWithStartedReruns(phases: SplitRunPhase[], attempts: StartedStageAutomationRerun[]): SplitRunPhase[] {
  if (attempts.length === 0) {
    return phases;
  }
  const knownRunIds = new Set(phases.flatMap((phase) => (phase.runId ? [phase.runId] : [])));
  const extras: SplitRunPhase[] = [];
  for (const attempt of attempts) {
    if (attempt.createdRunId && knownRunIds.has(attempt.createdRunId)) {
      continue;
    }
    const source = [...phases, ...extras].find(
      (phase) => phase.id === attempt.sourcePhaseId && phase.appId === attempt.appId,
    );
    if (!source) {
      continue;
    }
    extras.push(startedRerunPhase(source, attempt));
  }
  return extras.length === 0 ? phases : [...phases, ...extras];
}

function startedRerunPhase(source: SplitRunPhase, attempt: StartedStageAutomationRerun): SplitRunPhase {
  const running = attempt.status === "running";
  const line: SplitRunStreamLine = {
    id: attempt.eventId,
    at: "",
    componentName: source.componentName,
    status: attempt.status,
    duration: "",
    durationRunning: running,
    kind: "action",
    componentType: source.componentName,
    action: running ? "running" : attempt.status,
  };
  return {
    id: temporaryRerunPhaseId(attempt.eventId),
    name: source.name,
    status: attempt.status,
    duration: "",
    durationRunning: running,
    startedAt: attempt.startedAt,
    componentName: source.componentName,
    artifacts: [],
    stream: [line],
    canvasSteps: [],
    appId: source.appId,
    runId: attempt.createdRunId,
    columnKey: source.columnKey,
    canvasKey: source.canvasKey,
    triggerName: source.triggerName,
    pullRequestActivity: source.pullRequestActivity,
  };
}
