import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { buildLatestArtifactDataById, overlayLiveArtifactData } from "../../lib/workOrderArtifact";
import {
  indexPullRequestsById,
  overlayLivePullRequest,
  pullRequestFromEventPayload,
} from "../../lib/workOrderPullRequest";
import type { SplitRunPhase, SplitRunStreamBroadcast, SplitRunStreamLine } from "./splitRunMocks";

interface EventRunRef {
  id?: string;
}

interface ArtifactAddedPayload {
  automation?: {
    nodeId?: string;
    nodeName?: string;
  };
  artifact?: {
    id?: string;
    type?: string;
    data?: Record<string, unknown>;
  };
  run?: EventRunRef;
}

interface PullRequestEventPayload {
  automation?: {
    nodeId?: string;
    nodeName?: string;
  };
  pullRequest?: {
    id?: string;
    provider?: string;
    repository?: string;
    number?: number | string;
    url?: string;
    title?: string;
    state?: string;
  };
  run?: EventRunRef;
}

/**
 * An indexed value plus the id of the run that produced it. `runId` is
 * undefined when the source event has no run reference (older data);
 * callers that scope by run treat that as "attach regardless of run".
 */
interface RunScoped<T> {
  value: T;
  runId?: string;
}

interface ActivityBroadcastPayload {
  automation?: {
    nodeId?: string;
    nodeName?: string;
  };
  title?: string;
  body?: string;
  url?: string;
  run?: EventRunRef;
}

export interface StreamArtifactIndex {
  byNodeId: Map<string, RunScoped<FactoriesWorkOrderArtifact>>;
  byNodeName: Map<string, RunScoped<FactoriesWorkOrderArtifact>>;
  pullRequestsByNodeId: Map<string, RunScoped<FactoriesFactoryPullRequest>>;
  pullRequestsByNodeName: Map<string, RunScoped<FactoriesFactoryPullRequest>>;
  broadcastsByNodeId: Map<string, RunScoped<SplitRunStreamBroadcast[]>>;
  broadcastsByNodeName: Map<string, RunScoped<SplitRunStreamBroadcast[]>>;
  /** Every artifact a canvas run produced, in the order they were added. */
  byRunId: Map<string, FactoriesWorkOrderArtifact[]>;
}

export function streamArtifactIndexFromEvents(
  events: FactoriesWorkOrderEvent[],
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
  livePullRequests?: FactoriesFactoryPullRequest[],
): StreamArtifactIndex {
  const index: StreamArtifactIndex = {
    byNodeId: new Map(),
    byNodeName: new Map(),
    pullRequestsByNodeId: new Map(),
    pullRequestsByNodeName: new Map(),
    broadcastsByNodeId: new Map(),
    broadcastsByNodeName: new Map(),
    byRunId: new Map(),
  };
  const liveById = liveArtifactsById(liveArtifacts);
  const latestDataById = buildLatestArtifactDataById(liveArtifacts ?? []);
  const livePullRequestsById = indexPullRequestsById(livePullRequests);

  for (const event of sortEventsChronologically(events)) {
    indexStreamEvent(index, event, liveById, latestDataById, livePullRequestsById);
  }

  for (const artifact of liveArtifacts ?? []) {
    const runId = canvasRunIdFromArtifact(artifact);
    if (runId) {
      appendRunArtifact(index.byRunId, runId, artifact);
    }
  }

  return index;
}

function indexStreamEvent(
  index: StreamArtifactIndex,
  event: FactoriesWorkOrderEvent,
  liveById: Map<string, FactoriesWorkOrderArtifact>,
  latestDataById: Map<string, Record<string, unknown>>,
  livePullRequestsById: Map<string, FactoriesFactoryPullRequest>,
): void {
  const automation = eventAutomation(event);
  const nodeId = automation?.nodeId?.trim();
  const nodeName = automation?.nodeName?.trim();
  const runId = eventRunId(event);

  const artifact = artifactFromStreamEvent(event, liveById, latestDataById);
  if (artifact) {
    if (nodeId) {
      index.byNodeId.set(nodeId, { value: artifact, runId });
    } else if (nodeName) {
      index.byNodeName.set(nodeName, { value: artifact, runId });
    }
    if (runId) {
      appendRunArtifact(index.byRunId, runId, artifact);
    }
  }

  const pullRequest = pullRequestFromStreamEvent(event, livePullRequestsById);
  if (pullRequest) {
    if (nodeId) {
      index.pullRequestsByNodeId.set(nodeId, { value: pullRequest, runId });
    } else if (nodeName) {
      index.pullRequestsByNodeName.set(nodeName, { value: pullRequest, runId });
    }
  }

  const broadcast = broadcastFromStreamEvent(event);
  if (broadcast) {
    appendBroadcast(index, nodeId, nodeName, runId, broadcast);
  }
}

/**
 * Puts the artifacts a canvas run produced onto that run's stage. Live
 * phases do not carry those artifacts themselves; the stage list reads
 * them from here.
 */
export function phasesWithRunArtifacts(phases: SplitRunPhase[], index: StreamArtifactIndex): SplitRunPhase[] {
  if (index.byRunId.size === 0) {
    return phases;
  }
  let changed = false;
  const next = phases.map((phase) => {
    const produced = phase.runId ? index.byRunId.get(phase.runId) : undefined;
    if (!produced?.length) {
      return phase;
    }
    const artifacts = mergeArtifacts(phase.artifacts, produced);
    if (sameArtifacts(phase.artifacts, artifacts)) {
      return phase;
    }
    changed = true;
    return { ...phase, artifacts };
  });
  return changed ? next : phases;
}

/** Copies indexed artifacts and broadcasts onto each phase stream. */
export function phasesWithAttachedStreams(phases: SplitRunPhase[], index: StreamArtifactIndex): SplitRunPhase[] {
  let changed = false;
  const next = phases.map((phase) => {
    const stream = attachArtifactsToStream(phase.stream, index, phase.runId);
    if (!stream || sameStream(phase.stream, stream)) {
      return phase;
    }
    changed = true;
    return { ...phase, stream };
  });
  return changed ? next : phases;
}

function sameStream(left: SplitRunStreamLine[] | undefined, right: SplitRunStreamLine[]): boolean {
  return left === right || (left?.length === right.length && left.every((line, index) => line === right[index]));
}

function appendRunArtifact(
  byRunId: Map<string, FactoriesWorkOrderArtifact[]>,
  runId: string,
  artifact: FactoriesWorkOrderArtifact,
) {
  const list = byRunId.get(runId) ?? [];
  const index = artifact.id ? list.findIndex((item) => item.id === artifact.id) : -1;
  if (index >= 0) {
    list[index] = artifact;
  } else {
    list.push(artifact);
  }
  byRunId.set(runId, list);
}

function canvasRunIdFromArtifact(artifact: FactoriesWorkOrderArtifact): string | undefined {
  const data = artifact.data;
  if (!data || typeof data !== "object") {
    return undefined;
  }
  const value = data.canvasRunId;
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function mergeArtifacts(
  existing: FactoriesWorkOrderArtifact[],
  produced: FactoriesWorkOrderArtifact[],
): FactoriesWorkOrderArtifact[] {
  const merged = [...existing];
  for (const artifact of produced) {
    const index = artifact.id ? merged.findIndex((item) => item.id === artifact.id) : -1;
    if (index >= 0) {
      merged[index] = artifact;
      continue;
    }
    merged.push(artifact);
  }
  return merged;
}

function sameArtifacts(left: FactoriesWorkOrderArtifact[], right: FactoriesWorkOrderArtifact[]): boolean {
  return left.length === right.length && left.every((item, index) => item === right[index]);
}

/**
 * Attaches artifacts/pull requests to a phase's stream lines.
 *
 * `runId` scopes the attachment to the canvas run that owns this stream
 * (a phase's `runId`). An indexed value produced by a different run is
 * skipped, so one run's artifacts never leak onto another run's phase
 * (e.g. a PLAN.md produced by a planning run should not show up on an
 * unrelated PR-activity run). When `runId` is omitted, attachment is
 * unscoped (matches prior behavior, used for previews and tests that
 * don't track runs).
 */
export function attachArtifactsToStream(
  stream: SplitRunStreamLine[] | undefined,
  index: StreamArtifactIndex,
  runId?: string,
): SplitRunStreamLine[] | undefined {
  if (!stream) {
    return undefined;
  }

  return stream.map((line) => attachLineArtifact(line, index, runId));
}

export function attachStreamArtifacts(
  stream: SplitRunStreamLine[] | undefined,
  events: FactoriesWorkOrderEvent[],
  liveArtifacts?: FactoriesWorkOrderArtifact[],
  livePullRequests?: FactoriesFactoryPullRequest[],
  runId?: string,
): SplitRunStreamLine[] | undefined {
  return attachArtifactsToStream(stream, streamArtifactIndexFromEvents(events, liveArtifacts, livePullRequests), runId);
}

function artifactFromStreamEvent(
  event: FactoriesWorkOrderEvent,
  liveById: Map<string, FactoriesWorkOrderArtifact>,
  latestDataById: Map<string, Record<string, unknown>>,
): FactoriesWorkOrderArtifact | undefined {
  if (event.type !== "order.artifact.added") {
    return undefined;
  }
  const payload = (event.event ?? {}) as ArtifactAddedPayload;
  return resolveAddedArtifact(payload.artifact, liveById, latestDataById);
}

function pullRequestFromStreamEvent(
  event: FactoriesWorkOrderEvent,
  liveById: Map<string, FactoriesFactoryPullRequest>,
): FactoriesFactoryPullRequest | undefined {
  if (event.type !== "order.pull_request.added" && event.type !== "order.pull_request.updated") {
    return undefined;
  }
  const payload = (event.event ?? {}) as PullRequestEventPayload;
  if (!payload.pullRequest) {
    return undefined;
  }
  return overlayLivePullRequest(pullRequestFromEventPayload(payload.pullRequest), liveById);
}

function broadcastFromStreamEvent(event: FactoriesWorkOrderEvent): SplitRunStreamBroadcast | undefined {
  if (event.type !== "order.activity.broadcast") {
    return undefined;
  }
  const payload = (event.event ?? {}) as ActivityBroadcastPayload;
  const title = payload.title?.trim();
  const body = payload.body?.trim() || undefined;
  const url = payload.url?.trim() || undefined;
  if (!title || (!body && !url)) {
    return undefined;
  }
  return { title, body, url };
}

function appendBroadcast(
  index: StreamArtifactIndex,
  nodeId: string | undefined,
  nodeName: string | undefined,
  runId: string | undefined,
  broadcast: SplitRunStreamBroadcast,
) {
  if (nodeId) {
    appendScopedBroadcast(index.broadcastsByNodeId, nodeId, runId, broadcast);
    return;
  }
  if (nodeName) {
    appendScopedBroadcast(index.broadcastsByNodeName, nodeName, runId, broadcast);
  }
}

function appendScopedBroadcast(
  target: Map<string, RunScoped<SplitRunStreamBroadcast[]>>,
  key: string,
  runId: string | undefined,
  broadcast: SplitRunStreamBroadcast,
) {
  const current = target.get(key);
  if (current && current.runId === runId) {
    current.value = [...current.value, broadcast];
    return;
  }
  target.set(key, { value: [broadcast], runId });
}

function eventAutomation(event: FactoriesWorkOrderEvent): { nodeId?: string; nodeName?: string } | undefined {
  const payload = (event.event ?? {}) as ArtifactAddedPayload & PullRequestEventPayload;
  return payload.automation;
}

function eventRunId(event: FactoriesWorkOrderEvent): string | undefined {
  const payload = (event.event ?? {}) as ArtifactAddedPayload & PullRequestEventPayload;
  return payload.run?.id?.trim() || undefined;
}

/** Keeps `entry` only when it belongs to `runId` (or either side is unscoped). */
function matchesRun<T>(entry: RunScoped<T> | undefined, runId: string | undefined): T | undefined {
  if (!entry) {
    return undefined;
  }
  if (runId && entry.runId && entry.runId !== runId) {
    return undefined;
  }
  return entry.value;
}

function attachLineArtifact(line: SplitRunStreamLine, index: StreamArtifactIndex, runId?: string): SplitRunStreamLine {
  const next = { ...line };
  const artifactById = matchesRun(line.nodeId ? index.byNodeId.get(line.nodeId) : undefined, runId);
  const artifactByName = matchesRun(index.byNodeName.get(line.componentName), runId);
  if (artifactById || artifactByName) {
    next.artifact = artifactById ?? artifactByName;
  }

  const pullRequestById = matchesRun(line.nodeId ? index.pullRequestsByNodeId.get(line.nodeId) : undefined, runId);
  const pullRequestByName = matchesRun(index.pullRequestsByNodeName.get(line.componentName), runId);
  if (pullRequestById || pullRequestByName) {
    next.pullRequest = pullRequestById ?? pullRequestByName;
  }

  const broadcastsById = matchesRun(line.nodeId ? index.broadcastsByNodeId.get(line.nodeId) : undefined, runId);
  const broadcastsByName = matchesRun(index.broadcastsByNodeName.get(line.componentName), runId);
  const broadcasts = broadcastsById ?? broadcastsByName;
  if (broadcasts?.length) {
    next.broadcasts = broadcasts;
  }

  return next;
}

function resolveAddedArtifact(
  snapshot: ArtifactAddedPayload["artifact"],
  liveById: Map<string, FactoriesWorkOrderArtifact>,
  latestDataById: Map<string, Record<string, unknown>>,
): FactoriesWorkOrderArtifact | undefined {
  if (!snapshot?.type) {
    return undefined;
  }

  const live = snapshot.id ? liveById.get(snapshot.id) : undefined;
  if (live) {
    return live;
  }

  const overlaid = overlayLiveArtifactData(
    { id: snapshot.id, type: snapshot.type, data: snapshot.data },
    latestDataById,
  );
  return {
    id: overlaid.id,
    type: overlaid.type as FactoriesWorkOrderArtifact["type"],
    data: overlaid.data,
  };
}

function liveArtifactsById(
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
): Map<string, FactoriesWorkOrderArtifact> {
  const byId = new Map<string, FactoriesWorkOrderArtifact>();
  for (const artifact of liveArtifacts ?? []) {
    if (artifact.id) {
      byId.set(artifact.id, artifact);
    }
  }
  return byId;
}

function sortEventsChronologically(events: FactoriesWorkOrderEvent[]): FactoriesWorkOrderEvent[] {
  return [...events].sort((left, right) => {
    const timeDiff = timestampMs(left.timestamp) - timestampMs(right.timestamp);
    if (timeDiff !== 0) {
      return timeDiff;
    }
    return (left.type ?? "").localeCompare(right.type ?? "");
  });
}

function timestampMs(value: string | undefined): number {
  const parsed = Date.parse(value ?? "");
  return Number.isNaN(parsed) ? 0 : parsed;
}
