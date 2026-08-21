import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import { OPEN_WORK_ORDER_ARTIFACTS } from "../../__fixtures__/factoryPageFixtureVariants";
import {
  HOUR_AGO,
  OPEN_WORK_ORDER,
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { presentWorkOrderChecks, type WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import { presentWorkOrderStatusNotes, type WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import type { WorkOrderTimelineEvent, WorkOrderTimelineStep } from "../../lib/workOrderTimelineEvents";

export type PopupConcept = "session" | "trace" | "job";

export type PopupLogState = "passed" | "running" | "waiting" | "failed";

export interface PopupLogEntry {
  id: string;
  actor: string;
  title: string;
  detail?: string;
  duration: string;
  state: PopupLogState;
  artifactId?: string;
}

export interface PopupFixture {
  title: string;
  owner: OrgUserDisplay;
  elapsed: string;
  startedLabel: string;
  costUsd: string;
  tokensLabel: string;
  description: FactoriesWorkOrderArtifact;
  outputs: FactoriesWorkOrderArtifact[];
  checks: WorkOrderCheckPresentation[];
  waitingNotes: WorkOrderStatusNotePresentation[];
  log: PopupLogEntry[];
}

const DESCRIPTION_BODY = OPEN_WORK_ORDER.description ?? "";

export const DESCRIPTION_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-description",
  type: "TYPE_MARKDOWN",
  data: {
    name: "description.md",
    title: "description.md",
    body: DESCRIPTION_BODY,
  },
};

export const AGENT_WORK_POPUP: PopupFixture = {
  title: "Reconcile duplicate refunds in ledger",
  owner: {
    id: STORYBOOK_ME_USER_ID,
    name: STORYBOOK_ME_USER_NAME,
    initials: getUserInitials(STORYBOOK_ME_USER_NAME),
    avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
  },
  elapsed: "12 min",
  startedLabel: "Started 1h ago",
  costUsd: "$4.18",
  tokensLabel: "86k tokens",
  description: DESCRIPTION_ARTIFACT,
  outputs: OPEN_WORK_ORDER_ARTIFACTS,
  checks: presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS).filter((check) => check.id !== "check-confidence"),
  waitingNotes: presentWorkOrderStatusNotes(OPEN_WORK_ORDER.statusNotes),
  log: [
    {
      id: "backlog",
      actor: "Backlog",
      title: "Create work order",
      duration: "2s",
      state: "passed",
      artifactId: "art-description",
    },
    {
      id: "plan",
      actor: "Plan",
      title: "Write investigation notes",
      duration: "1m 12s",
      state: "passed",
    },
    {
      id: "implement",
      actor: "Implement",
      title: "Create feature/refund-retry",
      duration: "7s",
      state: "passed",
      artifactId: "art-branch-1",
    },
    {
      id: "verify",
      actor: "Verify",
      title: "Open PR #482",
      duration: "5s",
      state: "passed",
      artifactId: "art-pr-1",
    },
    {
      id: "done",
      actor: "Done",
      title: "Attach investigation notes",
      duration: "2s",
      state: "passed",
      artifactId: "art-md-1",
    },
  ],
};

/** In-flight job: scores arrive after automations finish. The log stays visible. */
export const AGENT_WORK_POPUP_RUNNING: PopupFixture = {
  ...AGENT_WORK_POPUP,
  title: "Add refund reconciliation test",
  elapsed: "4 min so far",
  startedLabel: "Started 1h ago",
  costUsd: "$0.73",
  tokensLabel: "2.7k tokens",
  outputs: [],
  checks: [],
  waitingNotes: [],
  log: [
    {
      id: "backlog",
      actor: "Backlog",
      title: "Create work order",
      duration: "2s",
      state: "passed",
      artifactId: "art-description",
    },
    {
      id: "plan",
      actor: "Plan",
      title: "Write test plan",
      duration: "1m 48s",
      state: "passed",
    },
    {
      id: "implement",
      actor: "Implement",
      title: "Add reconciliation test",
      duration: "4m so far",
      state: "running",
    },
  ],
};

export function buildPopupDispatchEvent(fixture: PopupFixture): WorkOrderTimelineEvent | null {
  if (fixture.log.length === 0) {
    return null;
  }

  const artifacts = [fixture.description, ...fixture.outputs];
  let cursor = Date.parse(HOUR_AGO);
  const steps: WorkOrderTimelineStep[] = fixture.log.map((entry) => {
    const startedAt = new Date(cursor).toISOString();
    const durationMs = logDurationMs(entry.duration);
    const finishedAt = durationMs != null ? new Date(cursor + durationMs).toISOString() : undefined;
    cursor += durationMs ?? 60_000;
    const execution = logExecution(entry.state);
    const artifact = entry.artifactId ? artifacts.find((item) => item.id === entry.artifactId) : undefined;

    return {
      id: entry.id,
      stepName: entry.actor,
      at: finishedAt ?? startedAt,
      startedAt,
      finishedAt,
      artifacts: artifact
        ? [
            {
              id: artifact.id,
              type: artifact.type ?? "TYPE_UNSPECIFIED",
              data: toArtifactDataRecord(artifact.data),
            },
          ]
        : undefined,
      comments: entry.title !== entry.actor ? [{ body: entry.title }] : undefined,
      execution: {
        id: entry.id,
        step: entry.actor,
        state: execution.state,
        result: execution.result,
        createdAt: startedAt,
        updatedAt: finishedAt ?? startedAt,
        run: { appName: entry.actor },
      },
    };
  });

  return {
    id: "popup-dispatch",
    kind: "dispatched",
    at: steps[0]?.startedAt ?? HOUR_AGO,
    lineName: "plan-and-implement",
    title: "Dispatched to plan-and-implement",
    steps,
  };
}

const PHASE_NAMES = ["Plan", "Implement", "Verify", "Done"] as const;

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

function backlogLogEntry(order: FactoriesWorkOrder): PopupLogEntry {
  const ingested = Boolean(order.createdBy?.automation);
  return {
    id: "backlog",
    actor: "Backlog",
    title: ingested ? "Create work order from GitHub issue" : "Create work order",
    duration: "2s",
    state: "passed",
    artifactId: "art-description",
  };
}

function ownerForOrder(order: FactoriesWorkOrder, fallback: OrgUserDisplay): OrgUserDisplay {
  const automation = order.createdBy?.automation;
  if (automation) {
    const name = automation.appName?.trim() || automation.nodeName?.trim();
    if (name) {
      return {
        id: automation.appId || automation.nodeId || name,
        name,
        initials: getUserInitials(name) || "A",
      };
    }
  }

  const user = order.createdBy?.user;
  if (user?.id || user?.name) {
    const name = user.name?.trim() || fallback.name;
    return {
      id: user.id ?? fallback.id,
      name,
      initials: getUserInitials(name),
      avatarUrl: user.id === STORYBOOK_ME_USER_ID ? STORYBOOK_ME_USER_AVATAR_URL : undefined,
    };
  }

  return fallback;
}

export function popupFixtureForWorkOrder(order?: FactoriesWorkOrder): PopupFixture {
  if (!order) {
    return AGENT_WORK_POPUP;
  }

  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order);
  const current = pickCurrentExecution(executions);
  const inVerify = current?.stepIndex === 2;
  const base = displayStatus === "running" ? AGENT_WORK_POPUP_RUNNING : AGENT_WORK_POPUP;

  return {
    ...base,
    title: order.title ?? base.title,
    waitingNotes: displayStatus === "waiting" ? presentWorkOrderStatusNotes(order.statusNotes, displayStatus) : [],
    checks: inVerify
      ? presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS).filter((check) => check.id !== "check-confidence")
      : [],
    description: descriptionArtifactForOrder(order),
    log: [backlogLogEntry(order), ...executions.map(executionToLogEntry)],
    elapsed: displayStatus === "draft" ? "Not started" : base.elapsed,
    owner: ownerForOrder(order, base.owner),
  };
}

function latestDispatchExecutions(order: FactoriesWorkOrder): FactoriesWorkOrderExecution[] {
  const dispatches = order.lineDispatches ?? [];
  if (dispatches.length === 0) {
    return [];
  }
  const latest = dispatches.reduce((best: FactoriesWorkOrderLineDispatch, candidate) => {
    const bestAt = Date.parse(best.createdAt ?? "") || 0;
    const candidateAt = Date.parse(candidate.createdAt ?? "") || 0;
    return candidateAt >= bestAt ? candidate : best;
  });
  return latest.stepExecutions ?? [];
}

function pickCurrentExecution(executions: FactoriesWorkOrderExecution[]): FactoriesWorkOrderExecution | undefined {
  const active = executions.filter(
    (execution) =>
      execution.state === "STATE_STARTED" ||
      execution.state === "STATE_PENDING" ||
      execution.state === "STATE_CANCELLING",
  );
  const pool = active.length > 0 ? active : executions;
  return pool.reduce<FactoriesWorkOrderExecution | undefined>((best, candidate) => {
    if (!best) {
      return candidate;
    }
    return (candidate.stepIndex ?? -1) >= (best.stepIndex ?? -1) ? candidate : best;
  }, undefined);
}

function executionToLogEntry(execution: FactoriesWorkOrderExecution): PopupLogEntry {
  const stepIndex = execution.stepIndex ?? 0;
  const actor = PHASE_NAMES[stepIndex] ?? execution.step ?? "Step";
  const state = logStateForExecution(execution);
  return {
    id: execution.id ?? actor,
    actor,
    title: execution.step ?? actor,
    duration: state === "running" ? "4m so far" : "1m 12s",
    state,
  };
}

function logStateForExecution(execution: FactoriesWorkOrderExecution): PopupLogState {
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

function logDurationMs(duration: string): number | null {
  if (duration.includes("so far")) {
    return null;
  }

  const minutes = duration.match(/(\d+)\s*m/);
  const seconds = duration.match(/(\d+)\s*s/);
  const ms = (minutes ? Number(minutes[1]) * 60_000 : 0) + (seconds ? Number(seconds[1]) * 1000 : 0);
  return ms > 0 ? ms : null;
}

function logExecution(state: PopupLogState): { state: string; result: string } {
  if (state === "passed") {
    return { state: "STATE_FINISHED", result: "RESULT_PASSED" };
  }
  if (state === "waiting") {
    return { state: "STATE_PENDING", result: "RESULT_UNKNOWN" };
  }
  if (state === "failed") {
    return { state: "STATE_FINISHED", result: "RESULT_FAILED" };
  }
  return { state: "STATE_STARTED", result: "RESULT_UNKNOWN" };
}
