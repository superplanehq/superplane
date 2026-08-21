import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";
import { workOrderOwnerDisplay } from "../../lib/workOrderCreator";
import { clockLabel, formatCostCents, formatTokenCount, providerForName } from "./splitRunFormat";

import { OPEN_WORK_ORDER_ARTIFACTS } from "../../__fixtures__/factoryPageFixtureVariants";
import {
  HOUR_AGO,
  OPEN_WORK_ORDER,
  REVIEWER_USER,
  RUNNING_WORK_ORDER,
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import { DESCRIPTION_ARTIFACT, PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
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

const PR_REVIEW_NOTES = presentWorkOrderStatusNotes(OPEN_WORK_ORDER.statusNotes);
const OPEN_PAGE_CHECKS = presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS);

const DRAFT_NEXT_STEP: WorkOrderStatusNotePresentation = {
  key: "start-plan",
  headline: "Start the next stage",
  text: "This work order is a draft. Dispatch it to the Plan and Implement line to start Plan.",
  cta: { label: "Start Plan", href: "#" },
};

const IMPLEMENT_FAILED_NOTE: WorkOrderStatusNotePresentation = {
  key: "implement-failed",
  headline: "Implement did not pass",
  text: "Backend tests failed on the reconciliation worker. The implement step stopped. Open the run to see the diagnosis, then dispatch the line again.",
  cta: { label: "Open failed run", href: "https://superplanehq.semaphoreci.com/" },
  source: { name: "Refund Implementer" },
};

const PLAN_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-plan-md",
  type: "TYPE_MARKDOWN",
  data: {
    name: "plan.md",
    title: "plan.md",
    body: "Add a focused test for the refund reconciliation worker.\nCover the timeout-then-retry path.",
  },
  createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  createdAt: HOUR_AGO,
};

const OWNER: OrgUserDisplay = {
  id: STORYBOOK_ME_USER_ID,
  name: STORYBOOK_ME_USER_NAME,
  initials: getUserInitials(STORYBOOK_ME_USER_NAME),
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

export const SPLIT_RUN_RUNNING: SplitRunFixture = {
  title: "Add refund reconciliation test",
  owner: OWNER,
  elapsed: "4 min so far",
  startedLabel: "Started 1h ago",
  costUsd: "$0.73",
  tokensLabel: "2.7k tokens",
  lineName: "plan-and-implement",
  lineStatus: "running",
  currentPhaseId: "implement",
  waitingNotes: [],
  checks: [],
  phases: [
    {
      id: "backlog",
      name: "Backlog",
      status: "passed",
      duration: "2s",
      componentName: "Create work order",
      artifacts: [DESCRIPTION_ARTIFACT],
      stream: [
        {
          id: "backlog-create",
          at: "12:24:02",
          componentName: "Create Work Order",
          status: "passed",
          duration: "2s",
          detail: "description.md",
        },
      ],
      canvasSteps: [
        {
          id: "create-work-order",
          title: "Create work order",
          componentName: "Create Work Order",
          provider: "superplane",
          status: "passed",
          detail: "description.md",
          duration: "2s",
        },
      ],
    },
    {
      id: "plan",
      name: "Plan",
      status: "passed",
      duration: "1m 12s",
      componentName: "Refund Planner",
      artifacts: [PLAN_ARTIFACT],
      stream: [
        {
          id: "plan-read",
          at: "12:24:05",
          componentName: "Read Work Order",
          status: "passed",
          duration: "4s",
          detail: "description.md",
        },
        {
          id: "plan-write",
          at: "12:24:09",
          componentName: "Refund Planner",
          status: "passed",
          duration: "1m 8s",
          detail: "plan.md",
        },
      ],
      canvasSteps: [
        {
          id: "read-order",
          title: "Read work order",
          componentName: "Read Work Order",
          provider: "superplane",
          status: "passed",
          detail: "description.md",
          duration: "4s",
        },
        {
          id: "refund-planner",
          title: "Write plan",
          componentName: "Refund Planner",
          provider: "superplane",
          status: "passed",
          detail: "plan.md",
          duration: "1m 8s",
        },
      ],
    },
    {
      id: "implement",
      name: "Implement",
      status: "running",
      duration: "4m so far",
      componentName: "Refund Implementer",
      artifacts: OPEN_WORK_ORDER_ARTIFACTS.filter((artifact) => artifact.id === "art-branch-1"),
      stream: [
        {
          id: "impl-branch",
          at: "12:25:14",
          componentName: "Create Branch",
          status: "passed",
          duration: "4s",
          detail: "feature/refund-retry",
        },
        {
          id: "impl-read",
          at: "12:25:18",
          componentName: "Read Artifact",
          status: "passed",
          duration: "3s",
          detail: "plan.md",
        },
        {
          id: "impl-write-file",
          at: "12:25:22",
          componentName: "Write File",
          status: "passed",
          duration: "11s",
          detail: "reconciliation_worker_test.go",
        },
        {
          id: "impl-agent",
          at: "12:25:33",
          componentName: "Refund Implementer",
          status: "running",
          duration: "4m so far",
          detail: "reconciliation_worker_test.go",
        },
        {
          id: "impl-pr",
          at: "—",
          componentName: "Create Pull Request",
          status: "pending",
          detail: "Waits on Refund Implementer",
        },
      ],
      canvasSteps: [
        {
          id: "read-plan",
          title: "Read plan",
          componentName: "Read Artifact",
          provider: "superplane",
          status: "passed",
          detail: "plan.md",
          duration: "3s",
        },
        {
          id: "refund-implementer",
          title: "Write test",
          componentName: "Refund Implementer",
          provider: "superplane",
          status: "running",
          detail: "reconciliation_worker_test.go",
          duration: "4m so far",
        },
        {
          id: "open-pr",
          title: "Open draft PR",
          componentName: "Create Pull Request",
          provider: "github",
          status: "pending",
          detail: "Waits on Refund Implementer",
        },
      ],
    },
  ],
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
  options?: { checks?: FactoriesWorkOrderCheck[] },
): SplitRunFixture {
  if (!order) {
    return SPLIT_RUN_RUNNING;
  }
  if (order.id === RUNNING_WORK_ORDER.id) {
    return {
      ...SPLIT_RUN_RUNNING,
      title: order.title ?? SPLIT_RUN_RUNNING.title,
      owner: workOrderOwnerDisplay(order, SPLIT_RUN_RUNNING.owner),
      costUsd: formatCostCents(order.totalCostCents) ?? SPLIT_RUN_RUNNING.costUsd,
      tokensLabel: formatTokenCount(order.totalTokens) ?? SPLIT_RUN_RUNNING.tokensLabel,
    };
  }

  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order);
  const current = pickCurrentExecution(executions);
  const backlog = backlogPhase(order);
  const phases = [backlog, ...executions.map((execution) => executionToPhase(execution))];
  const currentPhaseId = current ? phaseIdForExecution(current) : backlog.id;

  return {
    title: order.title ?? SPLIT_RUN_RUNNING.title,
    owner: workOrderOwnerDisplay(order, SPLIT_RUN_RUNNING.owner),
    elapsed: elapsedForDisplay(displayStatus),
    startedLabel: SPLIT_RUN_RUNNING.startedLabel,
    costUsd: formatCostCents(order.totalCostCents) ?? (displayStatus === "draft" ? "$0.00" : SPLIT_RUN_RUNNING.costUsd),
    tokensLabel:
      formatTokenCount(order.totalTokens) ?? (displayStatus === "draft" ? "0 tokens" : SPLIT_RUN_RUNNING.tokensLabel),
    lineName: latestDispatch(order)?.line?.name ?? SPLIT_RUN_RUNNING.lineName,
    lineStatus: lineStatusForDisplay(displayStatus),
    currentPhaseId,
    phases,
    ...reviewSurfaces(order, displayStatus, options?.checks),
  };
}

function reviewSurfaces(
  order: FactoriesWorkOrder,
  displayStatus: WorkOrderDisplayStatus,
  apiChecks?: FactoriesWorkOrderCheck[],
): Pick<SplitRunFixture, "waitingNotes" | "checks" | "footerTone"> {
  const executions = latestDispatchExecutions(order);
  const current = pickCurrentExecution(executions);
  const column = boardColumnFor(current, executions.length);
  const implementFailed = column === "implement" && current?.result === "RESULT_FAILED";
  const showChecks = column === "verify" || column === "done";
  const checks = showChecks
    ? apiChecks !== undefined
      ? presentWorkOrderChecks(apiChecks)
      : OPEN_PAGE_CHECKS
    : [];

  if (displayStatus === "draft") {
    return { waitingNotes: [DRAFT_NEXT_STEP], checks: [], footerTone: "draft" };
  }
  if (implementFailed) {
    return { waitingNotes: [IMPLEMENT_FAILED_NOTE], checks: [], footerTone: "failed" };
  }
  if (column === "implement" && (displayStatus === "waiting" || current?.state === "STATE_PENDING")) {
    return { waitingNotes: PR_REVIEW_NOTES, checks: [], footerTone: "waiting" };
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

function backlogPhase(order: FactoriesWorkOrder): SplitRunPhase {
  const ingested = Boolean(order.createdBy?.automation);
  const componentName = ingested ? "Create work order from GitHub issue" : "Create work order";
  const line: SplitRunStreamLine = {
    id: "backlog-create",
    at: clockLabel(order.createdAt),
    componentName,
    status: "passed",
    duration: "2s",
    detail: "description.md",
  };
  return {
    id: "backlog",
    name: "Backlog",
    status: "passed",
    duration: "2s",
    componentName,
    artifacts: [descriptionArtifactForOrder(order)],
    stream: [line],
    canvasSteps: [streamLineToCanvasStep(line, "superplane")],
  };
}

function executionToPhase(execution: FactoriesWorkOrderExecution): SplitRunPhase {
  const status = statusForExecution(execution);
  const name = execution.step ?? "Step";
  const componentName = execution.run?.appName ?? name;
  const duration = durationForStatus(status);
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

function lineStatusForDisplay(status: WorkOrderDisplayStatus): SplitRunPhaseStatus {
  if (status === "running") return "running";
  if (status === "completed") return "passed";
  if (status === "waiting") return "waiting";
  if (status === "failed") return "failed";
  return "pending";
}

function elapsedForDisplay(status: WorkOrderDisplayStatus): string {
  if (status === "draft") return "Not started";
  if (status === "running") return "4 min so far";
  if (status === "waiting") return "Waiting";
  return "12 min";
}

function durationForStatus(status: SplitRunPhaseStatus): string {
  if (status === "running") return "4m so far";
  if (status === "waiting" || status === "pending") return "—";
  return "1m 12s";
}

function phaseIdForExecution(execution: FactoriesWorkOrderExecution): string {
  const name = (execution.step ?? "step").toLowerCase().replace(/[^a-z0-9]+/g, "-");
  return `${name}-${execution.stepIndex ?? 0}`;
}

function latestDispatch(order: FactoriesWorkOrder): FactoriesWorkOrderLineDispatch | undefined {
  const dispatches = order.lineDispatches ?? [];
  if (dispatches.length === 0) {
    return undefined;
  }
  return dispatches.reduce((best, candidate) => {
    const bestAt = Date.parse(best.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= bestAt ? candidate : best;
  });
}

function latestDispatchExecutions(order: FactoriesWorkOrder): FactoriesWorkOrderExecution[] {
  return latestDispatch(order)?.stepExecutions ?? [];
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

function descriptionArtifactForOrder(order: FactoriesWorkOrder): FactoriesWorkOrderArtifact {
  return {
    ...DESCRIPTION_ARTIFACT,
    data: {
      name: "description.md",
      title: "description.md",
      body: order.description ?? "",
    },
  };
}
