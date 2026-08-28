import type {
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";
import { workOrderOwnerDisplay } from "../../lib/workOrderCreator";

import {
  OPEN_WORK_ORDER_ARTIFACTS,
  OPEN_WORK_ORDER_PULL_REQUESTS,
} from "../../__fixtures__/factoryPageFixtureVariants";
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
  pullRequestId?: string;
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
  pullRequests?: FactoriesFactoryPullRequest[];
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
  pullRequests: OPEN_WORK_ORDER_PULL_REQUESTS,
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
      actor: "Create plan",
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
      pullRequestId: "art-pr-1",
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
      actor: "Create plan",
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
  const pullRequests = fixture.pullRequests ?? [];
  let cursor = Date.parse(HOUR_AGO);
  const steps: WorkOrderTimelineStep[] = fixture.log.map((entry) => {
    const startedAt = new Date(cursor).toISOString();
    const durationMs = logDurationMs(entry.duration);
    const finishedAt = durationMs != null ? new Date(cursor + durationMs).toISOString() : undefined;
    cursor += durationMs ?? 60_000;
    const execution = logExecution(entry.state);
    const artifact = entry.artifactId ? artifacts.find((item) => item.id === entry.artifactId) : undefined;
    const pullRequest = entry.pullRequestId ? pullRequests.find((item) => item.id === entry.pullRequestId) : undefined;

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
      pullRequests: pullRequest ? [pullRequest] : undefined,
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

const PHASE_NAMES = ["Implement", "Verify", "Done"] as const;
const PR_CLOSURE_APP_NAME = "PR Closure";

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

export function popupFixtureForWorkOrder(order?: FactoriesWorkOrder): PopupFixture {
  if (!order) {
    return AGENT_WORK_POPUP;
  }

  const displayStatus = getWorkOrderDisplayStatus(order);
  const executions = latestDispatchExecutions(order);
  const current = pickCurrentExecution(executions);
  const inVerify = (current?.step ?? "").toLowerCase().includes("verify");
  const base = displayStatus === "running" ? AGENT_WORK_POPUP_RUNNING : AGENT_WORK_POPUP;

  return {
    ...base,
    title: order.title ?? base.title,
    waitingNotes: displayStatus === "waiting" ? presentWorkOrderStatusNotes(order.statusNotes, displayStatus) : [],
    checks: inVerify
      ? presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS).filter((check) => check.id !== "check-confidence")
      : [],
    description: descriptionArtifactForOrder(order),
    outputs: [],
    log: [backlogLogEntry(order), ...executions.map((execution) => executionToLogEntry(order, execution))],
    elapsed: displayStatus === "draft" ? "Not started" : base.elapsed,
    owner: workOrderOwnerDisplay(order, base.owner),
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

function isPrClosureRun(execution: FactoriesWorkOrderExecution): boolean {
  return execution.run?.appName === PR_CLOSURE_APP_NAME;
}

function doneLogTitle(order: FactoriesWorkOrder, execution: FactoriesWorkOrderExecution): string {
  const fromPullRequest = isPrClosureRun(execution);
  if (order.result === "RESULT_REJECTED") {
    return fromPullRequest ? "Reject work order from closed pull request" : "Reject work order";
  }
  return fromPullRequest ? "Complete work order from merged pull request" : "Complete work order";
}

function executionToLogEntry(order: FactoriesWorkOrder, execution: FactoriesWorkOrderExecution): PopupLogEntry {
  const stepIndex = execution.stepIndex ?? 0;
  const actor = execution.step?.trim() || PHASE_NAMES[stepIndex] || "Step";
  const state = logStateForExecution(execution);
  const isDone = stepIndex === 2 || (execution.step ?? "").toLowerCase().includes("done");
  return {
    id: execution.id ?? actor,
    actor,
    title: isDone ? doneLogTitle(order, execution) : (execution.step ?? actor),
    duration: state === "running" ? "4m so far" : "1m 12s",
    state,
    artifactId: undefined,
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

function logExecution(state: PopupLogState): Pick<FactoriesWorkOrderExecution, "state" | "result"> {
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
