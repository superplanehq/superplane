import { UNKNOWN_ORG_USER_NAME } from "@/lib/orgUserDisplay";
import { extractArtifactName, extractArtifactTitle, extractArtifactUrl } from "./workOrderArtifact";
import { pullRequestLabel } from "./workOrderPullRequest";
import type { FactoriesFactoryPullRequest } from "@/api-client";
import type {
  UserNameLookup,
  WorkOrderTimelineAutomationActor,
  WorkOrderTimelineEvent,
  WorkOrderTimelineStep,
} from "./workOrderTimelineEvents";

const UNKNOWN_MEMBER_LABEL = UNKNOWN_ORG_USER_NAME;

const ARTIFACT_KIND_SHORT_LABEL: Record<string, string> = {
  markdown: "note",
  branch: "branch",
};

interface EventUserRef {
  id?: string;
}

export function describePullRequestEvent(
  kind: "pullRequestAdded" | "pullRequestUpdated",
  pullRequest: FactoriesFactoryPullRequest,
): string {
  const label = pullRequestLabel(pullRequest);
  return kind === "pullRequestUpdated" ? `updated pull request ${label}` : `added pull request ${label}`;
}

export function describeArtifactAdded(artifact: { type?: string; data?: Record<string, unknown> }): string {
  const label =
    extractArtifactTitle(artifact.data) || extractArtifactUrl(artifact.data) || extractArtifactName(artifact.data);
  const type = formatArtifactKindShort(artifact.type);
  return label ? `attached ${type}: ${label}` : `attached ${type}`;
}

export function resolveUserDisplayName(
  userId: string | undefined,
  resolveUserName?: UserNameLookup,
): string | undefined {
  if (!userId) {
    return undefined;
  }

  return resolveUserName?.(userId) ?? UNKNOWN_MEMBER_LABEL;
}

export function describeAssigneesUpdated(
  payload: {
    user?: EventUserRef;
    assigned?: EventUserRef[];
    unassigned?: EventUserRef[];
  },
  resolveUserName?: UserNameLookup,
): string {
  const actorId = payload.user?.id;
  const assignedIds = (payload.assigned ?? []).map((user) => user.id).filter((id): id is string => Boolean(id));
  const assignedOthers = actorId ? assignedIds.filter((userId) => userId !== actorId) : assignedIds;
  const selfAssigned = Boolean(actorId && assignedIds.includes(actorId));
  const unassignedNames = formatEventUserNames(payload.unassigned, resolveUserName);
  const parts: string[] = [];

  if (selfAssigned && assignedOthers.length === 0) {
    parts.push("took ownership");
  } else if (selfAssigned && assignedOthers.length > 0) {
    const otherNames = formatEventUserNames(
      assignedOthers.map((id) => ({ id })),
      resolveUserName,
    );
    parts.push(otherNames ? `took ownership and assigned ${otherNames} as owner` : "took ownership");
  } else {
    const assignedNames = formatEventUserNames(payload.assigned, resolveUserName);
    if (assignedNames) {
      parts.push(`assigned ${assignedNames} as owner`);
    }
  }

  if (unassignedNames) {
    parts.push(`removed ${unassignedNames} as owner`);
  }

  if (parts.length === 0) {
    return "updated owners";
  }

  return parts.join(" and ");
}

function formatArtifactKindShort(type: string | undefined): string {
  if (!type) {
    return "artifact";
  }
  return ARTIFACT_KIND_SHORT_LABEL[type] ?? "artifact";
}

function formatEventUserNames(users: EventUserRef[] | undefined, resolveUserName?: UserNameLookup): string | null {
  if (!users?.length) {
    return null;
  }

  const names = users.map((user) => resolveUserDisplayName(user.id, resolveUserName) ?? UNKNOWN_MEMBER_LABEL);
  return formatNameList(names);
}

function formatNameList(names: string[]): string {
  if (names.length === 0) {
    return "";
  }

  if (names.length === 1) {
    return names[0];
  }

  if (names.length === 2) {
    return `${names[0]} and ${names[1]}`;
  }

  return `${names.slice(0, -1).join(", ")}, and ${names[names.length - 1]}`;
}

export function toAutomationActor(payload?: {
  nodeId?: string;
  nodeName?: string;
  appId?: string;
  appName?: string;
  lineId?: string;
  lineName?: string;
  stepIndex?: number;
  stepName?: string;
}): WorkOrderTimelineAutomationActor | undefined {
  if (!payload) {
    return undefined;
  }

  const anySet =
    payload.nodeId || payload.nodeName || payload.appId || payload.appName || payload.lineId || payload.lineName;
  if (!anySet) {
    return undefined;
  }

  return {
    nodeId: payload.nodeId,
    nodeName: payload.nodeName,
    appId: payload.appId,
    appName: payload.appName,
    lineId: payload.lineId,
    lineName: payload.lineName,
    stepIndex: payload.stepIndex,
    stepName: payload.stepName,
  };
}

export function findAutomationStep(
  state: { events: WorkOrderTimelineEvent[] },
  actor: WorkOrderTimelineAutomationActor | undefined,
): WorkOrderTimelineStep | undefined {
  if (!actor) {
    return undefined;
  }

  const dispatch = [...state.events]
    .reverse()
    .find(
      (event) =>
        event.kind === "dispatched" &&
        ((actor.lineId && event.lineId === actor.lineId) || (actor.lineName && event.lineName === actor.lineName)),
    );
  if (!dispatch?.steps?.length) {
    return undefined;
  }

  if (actor.stepName) {
    const matchingName = [...dispatch.steps].reverse().find((step) => step.stepName === actor.stepName);
    if (matchingName) {
      return matchingName;
    }
  }

  if (actor.stepIndex !== undefined && actor.stepIndex >= 0 && actor.stepIndex < dispatch.steps.length) {
    return dispatch.steps[actor.stepIndex];
  }

  // No confident match. Let the caller push a top-level event rather than
  // attaching to an arbitrary step (last would misattribute step-0 refs when
  // JSON `omitempty` dropped the index).
  return undefined;
}
