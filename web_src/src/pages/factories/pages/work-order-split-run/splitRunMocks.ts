import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
  FactoriesWorkOrderExecution,
} from "@/api-client";
import { UNKNOWN_ORG_USER_NAME, getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";
import { workOrderOwnerDisplay } from "../../lib/workOrderCreator";
import { latestDispatchForLine } from "../../lib/workOrderNumberResolution";
import { clockLabel, providerForName } from "./splitRunFormat";
import { SPLIT_RUN_RUNNING } from "./splitRunRunningFixture";
import {
  costUsdForDisplay,
  durationForExecution,
  elapsedForDisplay,
  lineStatusForDisplay,
  startedLabelForOrder,
  tokensLabelForDisplay,
} from "./splitRunWorkOrderDisplay";

import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import { PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import type {
  RunOverlayProvider,
  RunOverlayStep,
  RunOverlayStepStatus,
} from "../work-order-run-overlay/workOrderRunOverlayMocks";

export type SplitRunPhaseId = string;

export type SplitRunPhaseStatus = "passed" | "running" | "pending" | "waiting" | "failed";

export type SplitRunStreamEvent = "started" | "expanded" | "completed";

export interface SplitRunStreamLine {
  id: string;
  nodeId?: string;
  event?: SplitRunStreamEvent;
  at: string;
  componentName: string;
  status: SplitRunPhaseStatus;
  duration?: string;
  detail?: string;
  artifact?: FactoriesWorkOrderArtifact;
  /** Agent transcript line. No checkmark. */
  note?: boolean;
}

export interface SplitRunPhase {
  id: SplitRunPhaseId;
  name: string;
  status: SplitRunPhaseStatus;
  duration: string;
  /** Component that ran or is running in this phase. */
  componentName: string;
  artifacts: FactoriesWorkOrderArtifact[];
  stream: SplitRunStreamLine[];
  canvasSteps: RunOverlayStep[];
  appId?: string;
  runId?: string;
}

export type SplitRunFooterTone = "waiting" | "draft" | "failed";

export type SplitRunBoardColumn = "backlog" | "plan" | "implement" | "verify" | "done";

export interface SplitRunFixture {
  title: string;
  owner: OrgUserDisplay;
  elapsed: string;
  startedLabel: string;
  costUsd: string;
  tokensLabel: string;
  lineName: string;
  lineStatus: SplitRunPhaseStatus;
  currentPhaseId: SplitRunPhaseId;
  phases: SplitRunPhase[];
  waitingNotes: WorkOrderStatusNotePresentation[];
  checks: WorkOrderCheckPresentation[];
  footerTone?: SplitRunFooterTone;
}

export { SPLIT_RUN_RUNNING };

const UNKNOWN_OWNER: OrgUserDisplay = {
  id: "unknown",
  name: UNKNOWN_ORG_USER_NAME,
  initials: getUserInitials(UNKNOWN_ORG_USER_NAME) || "U",
};

const DRAFT_NEXT_STEP: WorkOrderStatusNotePresentation = {
  key: "start-plan",
  headline: "Start the next stage",
  text: "This work order is a draft. Dispatch it to the Plan and Implement line to start Plan.",
  cta: { label: "Dispatch" },
};

const IMPLEMENT_FAILED_NOTE: WorkOrderStatusNotePresentation = {
  key: "implement-failed",
  headline: "Implement did not pass",
  text: "Backend tests failed on the reconciliation worker. The implement step stopped. Open the run to see the diagnosis, then dispatch the line again.",
  cta: { label: "Open failed run", href: "https://superplanehq.semaphoreci.com/" },
  source: { name: "Refund Implementer" },
};

export function phaseById(fixture: SplitRunFixture, id: SplitRunPhaseId): SplitRunPhase {
  const phase = fixture.phases.find((entry) => entry.id === id);
  if (!phase) {
    throw new Error(`Unknown split-run phase: ${id}`);
  }
  return phase;
}

export function splitRunStatusLabel(status: SplitRunPhaseStatus): string {
  if (status === "passed") return "Completed";
  if (status === "running") return "Running";
  if (status === "waiting") return "Needs attention";
  if (status === "failed") return "Failed";
  return "Pending";
}

const PR_CLOSURE_APP_NAME = "PR Closure";

export function splitRunFixtureForWorkOrder(
  order?: FactoriesWorkOrder,
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null },
): SplitRunFixture {
  if (!order) {
    return SPLIT_RUN_RUNNING;
  }
  return mappedWorkOrderFixture(order, options);
}

function mappedWorkOrderFixture(
  order: FactoriesWorkOrder,
  options?: { checks?: FactoriesWorkOrderCheck[]; lineId?: string | null },
): SplitRunFixture {
  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order, options?.lineId);
  const current = pickCurrentExecution(executions);
  const phases = executions.map((execution) => executionToPhase(execution));
  return {
    title: order.title ?? "Work order",
    owner: workOrderOwnerDisplay(order, UNKNOWN_OWNER),
    elapsed: elapsedForDisplay(displayStatus, order),
    startedLabel: startedLabelForOrder(order),
    costUsd: costUsdForDisplay(order),
    tokensLabel: tokensLabelForDisplay(order),
    lineName: latestDispatchForLine(order, options?.lineId)?.line?.name ?? SPLIT_RUN_RUNNING.lineName,
    lineStatus: lineStatusForDisplay(displayStatus),
    currentPhaseId: current ? phaseIdForExecution(current) : (phases[0]?.id ?? ""),
    phases,
    ...reviewSurfaces(order, displayStatus, options?.checks, options?.lineId),
  };
}

function reviewSurfaces(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  apiChecks?: FactoriesWorkOrderCheck[],
  lineId?: string | null,
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footerTone"> {
  const executions = latestDispatchExecutions(order, lineId);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const implementFailed = column === "implement" && current?.result === "RESULT_FAILED";
  const showChecks = column === "verify" || column === "done";
  const checks = showChecks ? presentWorkOrderChecks(apiChecks ?? []) : [];

  if (displayStatus === "draft") {
    return { waitingNotes: [DRAFT_NEXT_STEP], checks: [], footerTone: "draft" };
  }
  if (implementFailed) {
    return { waitingNotes: [IMPLEMENT_FAILED_NOTE], checks: [], footerTone: "failed" };
  }
  if (column === "implement" && (displayStatus === "waiting" || current?.state === "STATE_PENDING")) {
    return {
      waitingNotes: presentWorkOrderStatusNotes(order.statusNotes, displayStatus),
      checks: [],
      footerTone: "waiting",
    };
  }
  return { waitingNotes: [], checks };
}

function boardColumnFor(current: FactoriesWorkOrderExecution | undefined, executionCount: number): SplitRunBoardColumn {
  if (!current || executionCount === 0) {
    return "backlog";
  }
  const step = (current.step ?? "").toLowerCase();
  const index = current.stepIndex ?? -1;
  if (step.includes("done") || index >= 3) {
    return "done";
  }
  if (step.includes("verify") || index === 2) {
    return "verify";
  }
  if (step.includes("implement") || index === 1) {
    return "implement";
  }
  if (step.includes("plan") || index === 0) {
    return "plan";
  }
  return "backlog";
}

function executionToPhase(execution: FactoriesWorkOrderExecution): SplitRunPhase {
  const status = statusForExecution(execution);
  const name = execution.step ?? "Step";
  const componentName = execution.run?.appName ?? name;
  const duration = durationForExecution(execution, status);
  const line: SplitRunStreamLine = {
    id: execution.id ?? name,
    at: clockLabel(execution.updatedAt ?? execution.createdAt),
    componentName,
    status,
    duration,
    detail: execution.step,
  };
  return {
    id: phaseIdForExecution(execution),
    name,
    status,
    duration,
    componentName,
    artifacts: isPrClosureRun(execution) ? [PR_CLOSURE_PR_ARTIFACT] : [],
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, providerForName(componentName))],
    appId: execution.run?.appId,
    runId: execution.run?.id,
  };
}

function streamLineToCanvasStep(line: SplitRunStreamLine, provider: RunOverlayProvider): RunOverlayStep {
  return {
    id: line.id,
    title: line.componentName,
    componentName: line.componentName,
    provider,
    status: canvasStatus(line.status),
    detail: line.detail,
    duration: line.duration === "—" ? undefined : line.duration,
  };
}

function canvasStatus(status: SplitRunPhaseStatus): RunOverlayStepStatus {
  if (status === "waiting") return "pending";
  return status;
}

function statusForExecution(execution: FactoriesWorkOrderExecution): SplitRunPhaseStatus {
  if (execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING") {
    return "running";
  }
  if (execution.state === "STATE_PENDING") {
    return "waiting";
  }
  if (execution.result === "RESULT_FAILED") {
    return "failed";
  }
  if (execution.result === "RESULT_PASSED") {
    return "passed";
  }
  return "waiting";
}

function phaseIdForExecution(execution: FactoriesWorkOrderExecution): string {
  const name = (execution.step ?? "step").toLowerCase().replace(/[^a-z0-9]+/g, "-");
  return `${name}-${execution.stepIndex ?? 0}`;
}

function latestDispatchExecutions(order: FactoriesWorkOrder, lineId?: string | null): FactoriesWorkOrderExecution[] {
  return latestDispatchForLine(order, lineId)?.stepExecutions ?? [];
}

function pickCurrentExecution(executions: FactoriesWorkOrderExecution[]): FactoriesWorkOrderExecution | undefined {
  const active = executions.filter(
    (execution) =>
      execution.state === "STATE_STARTED" ||
      execution.state === "STATE_PENDING" ||
      execution.state === "STATE_CANCELLING" ||
      execution.result === "RESULT_FAILED",
  );
  const pool = active.length > 0 ? active : executions;
  return pool.reduce<FactoriesWorkOrderExecution | undefined>((best, candidate) => {
    if (!best) {
      return candidate;
    }
    return (candidate.stepIndex ?? -1) >= (best.stepIndex ?? -1) ? candidate : best;
  }, undefined);
}

function isPrClosureRun(execution: FactoriesWorkOrderExecution): boolean {
  return execution.run?.appName === PR_CLOSURE_APP_NAME;
}
