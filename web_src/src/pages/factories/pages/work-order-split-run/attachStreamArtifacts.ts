import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { buildLatestArtifactDataById, overlayLiveArtifactData } from "../../lib/workOrderArtifact";
import {
  indexPullRequestsById,
  overlayLivePullRequest,
  pullRequestFromEventPayload,
} from "../../lib/workOrderPullRequest";
import type { SplitRunStreamLine } from "./splitRunMocks";

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

export interface StreamArtifactIndex {
  byNodeId: Map<string, RunScoped<FactoriesWorkOrderArtifact>>;
  byNodeName: Map<string, RunScoped<FactoriesWorkOrderArtifact>>;
  pullRequestsByNodeId: Map<string, RunScoped<FactoriesFactoryPullRequest>>;
  pullRequestsByNodeName: Map<string, RunScoped<FactoriesFactoryPullRequest>>;
}

export function streamArtifactIndexFromEvents(
  events: FactoriesWorkOrderEvent[],
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
  livePullRequests?: FactoriesFactoryPullRequest[],
): StreamArtifactIndex {
  const byNodeId = new Map<string, RunScoped<FactoriesWorkOrderArtifact>>();
  const byNodeName = new Map<string, RunScoped<FactoriesWorkOrderArtifact>>();
  const pullRequestsByNodeId = new Map<string, RunScoped<FactoriesFactoryPullRequest>>();
  const pullRequestsByNodeName = new Map<string, RunScoped<FactoriesFactoryPullRequest>>();
  const liveById = liveArtifactsById(liveArtifacts);
  const latestDataById = buildLatestArtifactDataById(liveArtifacts ?? []);
  const livePullRequestsById = indexPullRequestsById(livePullRequests);

  for (const event of sortEventsChronologically(events)) {
    const automation = eventAutomation(event);
    const nodeId = automation?.nodeId?.trim();
    const nodeName = automation?.nodeName?.trim();
    const runId = eventRunId(event);

    const artifact = artifactFromStreamEvent(event, liveById, latestDataById);
    if (artifact) {
      if (nodeId) {
        byNodeId.set(nodeId, { value: artifact, runId });
      } else if (nodeName) {
        byNodeName.set(nodeName, { value: artifact, runId });
      }
    }

    const pullRequest = pullRequestFromStreamEvent(event, livePullRequestsById);
    if (pullRequest) {
      if (nodeId) {
        pullRequestsByNodeId.set(nodeId, { value: pullRequest, runId });
      } else if (nodeName) {
        pullRequestsByNodeName.set(nodeName, { value: pullRequest, runId });
      }
    }
  }

  return { byNodeId, byNodeName, pullRequestsByNodeId, pullRequestsByNodeName };
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
