import { canvasesDescribeRun, canvasesListRuns, canvasesReemitTriggerEvent } from "@/api-client";
import { invalidateFactoryWorkOrderQueries } from "@/hooks/useFactoryWebsocket";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { findRunInListRunsResponse } from "@/pages/app/sidebarRunLookup";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { selectCreatedRerun } from "@/ui/CanvasPage/runInspectionRerunSelection";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";

import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "../splitRunMocks";

const RERUN_FAILURE_TOAST = "Failed to restart run";
const RERUN_SUCCESS_TOAST = "Run restarted";
const RUN_LOOKUP_PAGE_LIMIT = 25;

export type StageAutomationRerunTarget = {
  sourcePhaseId: string;
  appId: string;
  runId: string;
};

type StartedStageAutomationRerun = StageAutomationRerunTarget & {
  createdRunId: string;
  startedAt: string;
};

type ReemittedStageAutomation = {
  canvasId: string;
  triggerNodeId: string;
  eventId: string;
};

export function useStageAutomationRerun(organizationId?: string, factoryId?: string, orderId?: string) {
  const queryClient = useQueryClient();
  const inFlight = useRef(false);
  const [pending, setPending] = useState(false);
  const [attempts, setAttempts] = useState<StartedStageAutomationRerun[]>([]);

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
        const createdRunId = reemitted ? await findCreatedRun(target.appId, organizationId, reemitted) : undefined;
        if (createdRunId) {
          setAttempts((current) => appendStartedRerun(current, target, createdRunId));
        }
        showSuccessToast(RERUN_SUCCESS_TOAST);
      } catch (error) {
        console.error(RERUN_FAILURE_TOAST, error);
        showErrorToast(RERUN_FAILURE_TOAST);
      } finally {
        inFlight.current = false;
        setPending(false);
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
): Promise<ReemittedStageAutomation | undefined> {
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
    return undefined;
  }

  return { canvasId: target.appId, triggerNodeId: rootEvent.nodeId, eventId };
}

async function findCreatedRun(
  canvasId: string,
  organizationId: string | undefined,
  reemitted: ReemittedStageAutomation,
): Promise<string | undefined> {
  try {
    return await createdRunIdForEvent(canvasId, organizationId, reemitted);
  } catch (error) {
    console.error("Failed to find restarted run", error);
    return undefined;
  }
}

async function createdRunIdForEvent(
  canvasId: string,
  organizationId: string | undefined,
  reemitted: ReemittedStageAutomation,
): Promise<string | undefined> {
  let createdRunId: string | undefined;
  await selectCreatedRerun({
    eventId: reemitted.eventId,
    triggerNodeId: reemitted.triggerNodeId,
    fetchRunId: (event) => fetchCreatedRerunId(canvasId, organizationId, event),
    selectRun: (runId) => {
      createdRunId = runId;
    },
  });
  return createdRunId;
}

async function fetchCreatedRerunId(
  canvasId: string,
  organizationId: string | undefined,
  event: SidebarEvent,
): Promise<string | null> {
  const response = await canvasesListRuns(
    withOrganizationHeader({
      organizationId,
      path: { canvasId },
      query: { limit: RUN_LOOKUP_PAGE_LIMIT },
    }),
  );
  return findRunInListRunsResponse(response.data?.runs ?? [], event)?.runId ?? null;
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

function appendStartedRerun(
  attempts: StartedStageAutomationRerun[],
  target: StageAutomationRerunTarget,
  createdRunId: string,
): StartedStageAutomationRerun[] {
  if (attempts.some((attempt) => attempt.createdRunId === createdRunId)) {
    return attempts;
  }
  return [...attempts, { ...target, createdRunId, startedAt: new Date().toISOString() }];
}

function phasesWithStartedReruns(phases: SplitRunPhase[], attempts: StartedStageAutomationRerun[]): SplitRunPhase[] {
  if (attempts.length === 0) {
    return phases;
  }
  const knownRunIds = new Set(phases.flatMap((phase) => (phase.runId ? [phase.runId] : [])));
  const extras = attempts.flatMap((attempt) => {
    if (knownRunIds.has(attempt.createdRunId)) {
      return [];
    }
    const source = phases.find((phase) => phase.id === attempt.sourcePhaseId && phase.appId === attempt.appId);
    return source ? [startedRerunPhase(source, attempt)] : [];
  });
  return extras.length === 0 ? phases : [...phases, ...extras];
}

function startedRerunPhase(source: SplitRunPhase, attempt: StartedStageAutomationRerun): SplitRunPhase {
  const line: SplitRunStreamLine = {
    id: attempt.createdRunId,
    at: "",
    componentName: source.componentName,
    status: "running",
    duration: "",
    durationRunning: true,
    kind: "action",
    componentType: source.componentName,
    action: "running",
  };
  return {
    id: `rerun-${attempt.createdRunId}`,
    name: source.name,
    status: "running",
    duration: "",
    durationRunning: true,
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
