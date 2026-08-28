import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { buildLatestArtifactDataById, overlayLiveArtifactData } from "../../lib/workOrderArtifact";
import {
  indexPullRequestsById,
  overlayLivePullRequest,
  pullRequestFromEventPayload,
} from "../../lib/workOrderPullRequest";
import type { SplitRunStreamLine } from "./splitRunMocks";

interface EventAutomation {
  nodeId?: string;
  nodeName?: string;
}

interface EventRun {
  id?: string;
}

interface ArtifactAddedPayload {
  automation?: EventAutomation;
  run?: EventRun;
  artifact?: {
    id?: string;
    type?: string;
    data?: Record<string, unknown>;
  };
}

interface PullRequestEventPayload {
  automation?: EventAutomation;
  run?: EventRun;
  pullRequest?: {
    id?: string;
    provider?: string;
    repository?: string;
    number?: number | string;
    url?: string;
    title?: string;
    state?: string;
  };
}

/** Everything a single canvas run produced: node-scoped lookups plus the raw lists. */
interface RunArtifactIndex {
  byNodeId: Map<string, FactoriesWorkOrderArtifact>;
  byNodeName: Map<string, FactoriesWorkOrderArtifact>;
  pullRequestsByNodeId: Map<string, FactoriesFactoryPullRequest>;
  pullRequestsByNodeName: Map<string, FactoriesFactoryPullRequest>;
  artifacts: FactoriesWorkOrderArtifact[];
  pullRequests: FactoriesFactoryPullRequest[];
}

function emptyRunArtifactIndex(): RunArtifactIndex {
  return {
    byNodeId: new Map(),
    byNodeName: new Map(),
    pullRequestsByNodeId: new Map(),
    pullRequestsByNodeName: new Map(),
    artifacts: [],
    pullRequests: [],
  };
}

/**
 * Artifacts and pull requests observed on the work order, scoped by the run
 * that produced them. Run scoping is required because node ids and names are
 * not unique across runs (for example a canvas re-run reuses the same node
 * ids), so a global index would leak artifacts from one run onto another.
 */
export interface StreamArtifactIndex {
  byRun: Map<string, RunArtifactIndex>;
}

export function streamArtifactIndexFromEvents(
  events: FactoriesWorkOrderEvent[],
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
  livePullRequests?: FactoriesFactoryPullRequest[],
): StreamArtifactIndex {
  const byRun = new Map<string, RunArtifactIndex>();
  const context = {
    liveById: liveArtifactsById(liveArtifacts),
    latestDataById: buildLatestArtifactDataById(liveArtifacts ?? []),
    livePullRequestsById: indexPullRequestsById(livePullRequests),
  };

  for (const event of sortEventsChronologically(events)) {
    const runId = eventRunId(event);
    // Events without a producing run cannot be scoped safely; drop them
    // rather than folding them into a shared bucket that reintroduces
    // cross-run leakage.
    if (runId) {
      indexEventIntoRun(byRun, runId, event, context);
    }
  }

  return { byRun };
}

function indexEventIntoRun(
  byRun: Map<string, RunArtifactIndex>,
  runId: string,
  event: FactoriesWorkOrderEvent,
  context: {
    liveById: Map<string, FactoriesWorkOrderArtifact>;
    latestDataById: Map<string, Record<string, unknown>>;
    livePullRequestsById: Map<string, FactoriesFactoryPullRequest>;
  },
): void {
  const automation = eventAutomation(event);
  const nodeId = automation?.nodeId?.trim();
  const nodeName = automation?.nodeName?.trim();

  const artifact = artifactFromStreamEvent(event, context.liveById, context.latestDataById);
  if (artifact) {
    addToRunIndex(runArtifactIndex(byRun, runId), ARTIFACT_KEYS, artifact, { nodeId, nodeName });
  }

  const pullRequest = pullRequestFromStreamEvent(event, context.livePullRequestsById);
  if (pullRequest) {
    addToRunIndex(runArtifactIndex(byRun, runId), PULL_REQUEST_KEYS, pullRequest, { nodeId, nodeName });
  }
}

const ARTIFACT_KEYS = { list: "artifacts", byNodeId: "byNodeId", byNodeName: "byNodeName" } as const;
const PULL_REQUEST_KEYS = {
  list: "pullRequests",
  byNodeId: "pullRequestsByNodeId",
  byNodeName: "pullRequestsByNodeName",
} as const;

function addToRunIndex<T>(
  runIndex: RunArtifactIndex,
  keys: {
    list: "artifacts" | "pullRequests";
    byNodeId: "byNodeId" | "pullRequestsByNodeId";
    byNodeName: "byNodeName" | "pullRequestsByNodeName";
  },
  value: T,
  node: { nodeId: string | undefined; nodeName: string | undefined },
): void {
  (runIndex[keys.list] as T[]).push(value);
  if (node.nodeId) {
    (runIndex[keys.byNodeId] as Map<string, T>).set(node.nodeId, value);
  } else if (node.nodeName) {
    (runIndex[keys.byNodeName] as Map<string, T>).set(node.nodeName, value);
  }
}

export function attachArtifactsToStream(
  stream: SplitRunStreamLine[] | undefined,
  index: StreamArtifactIndex,
  runId: string | undefined,
): SplitRunStreamLine[] | undefined {
  if (!stream) {
    return undefined;
  }

  const runIndex = (runId && index.byRun.get(runId)) || undefined;
  return stream.map((line) => attachLineArtifact(line, runIndex));
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

function runArtifactIndex(byRun: Map<string, RunArtifactIndex>, runId: string): RunArtifactIndex {
  const existing = byRun.get(runId);
  if (existing) {
    return existing;
  }
  const created = emptyRunArtifactIndex();
  byRun.set(runId, created);
  return created;
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

function eventAutomation(event: FactoriesWorkOrderEvent): EventAutomation | undefined {
  const payload = (event.event ?? {}) as ArtifactAddedPayload & PullRequestEventPayload;
  return payload.automation;
}

function eventRunId(event: FactoriesWorkOrderEvent): string | undefined {
  const payload = (event.event ?? {}) as ArtifactAddedPayload & PullRequestEventPayload;
  return payload.run?.id?.trim() || undefined;
}

function attachLineArtifact(line: SplitRunStreamLine, index: RunArtifactIndex | undefined): SplitRunStreamLine {
  if (!index) {
    return line;
  }

  const next = { ...line };
  const artifactById = line.nodeId ? index.byNodeId.get(line.nodeId) : undefined;
  const artifactByName = index.byNodeName.get(line.componentName);
  next.artifact = artifactById ?? artifactByName ?? soleRunArtifact(line, index);

  const pullRequestById = line.nodeId ? index.pullRequestsByNodeId.get(line.nodeId) : undefined;
  const pullRequestByName = index.pullRequestsByNodeName.get(line.componentName);
  next.pullRequest = pullRequestById ?? pullRequestByName ?? soleRunPullRequest(line, index);

  return next;
}

/**
 * A PR-activity run renders as a single synthetic line with no nodeId, so it
 * never matches byNodeId/byNodeName. Fall back to the run's own artifacts so
 * the line still shows what its run genuinely produced, and nothing else.
 */
function soleRunArtifact(line: SplitRunStreamLine, index: RunArtifactIndex): FactoriesWorkOrderArtifact | undefined {
  if (line.nodeId || index.artifacts.length !== 1) {
    return line.artifact;
  }
  return index.artifacts[0];
}

function soleRunPullRequest(
  line: SplitRunStreamLine,
  index: RunArtifactIndex,
): FactoriesFactoryPullRequest | undefined {
  if (line.nodeId || index.pullRequests.length !== 1) {
    return line.pullRequest;
  }
  return index.pullRequests[0];
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
