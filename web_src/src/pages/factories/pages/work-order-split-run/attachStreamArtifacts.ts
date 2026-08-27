import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { buildLatestArtifactDataById, overlayLiveArtifactData } from "../../lib/workOrderArtifact";
import {
  indexPullRequestsById,
  overlayLivePullRequest,
  pullRequestFromEventPayload,
} from "../../lib/workOrderPullRequest";
import type { SplitRunStreamLine } from "./splitRunMocks";

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
}

export interface StreamArtifactIndex {
  byNodeId: Map<string, FactoriesWorkOrderArtifact>;
  byNodeName: Map<string, FactoriesWorkOrderArtifact>;
  pullRequestsByNodeId: Map<string, FactoriesFactoryPullRequest>;
  pullRequestsByNodeName: Map<string, FactoriesFactoryPullRequest>;
}

export function streamArtifactIndexFromEvents(
  events: FactoriesWorkOrderEvent[],
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
  livePullRequests?: FactoriesFactoryPullRequest[],
): StreamArtifactIndex {
  const byNodeId = new Map<string, FactoriesWorkOrderArtifact>();
  const byNodeName = new Map<string, FactoriesWorkOrderArtifact>();
  const pullRequestsByNodeId = new Map<string, FactoriesFactoryPullRequest>();
  const pullRequestsByNodeName = new Map<string, FactoriesFactoryPullRequest>();
  const liveById = liveArtifactsById(liveArtifacts);
  const latestDataById = buildLatestArtifactDataById(liveArtifacts ?? []);
  const livePullRequestsById = indexPullRequestsById(livePullRequests);

  for (const event of sortEventsChronologically(events)) {
    const automation = eventAutomation(event);
    const nodeId = automation?.nodeId?.trim();
    const nodeName = automation?.nodeName?.trim();

    const artifact = artifactFromStreamEvent(event, liveById, latestDataById);
    if (artifact) {
      if (nodeId) {
        byNodeId.set(nodeId, artifact);
      } else if (nodeName) {
        byNodeName.set(nodeName, artifact);
      }
    }

    const pullRequest = pullRequestFromStreamEvent(event, livePullRequestsById);
    if (pullRequest) {
      if (nodeId) {
        pullRequestsByNodeId.set(nodeId, pullRequest);
      } else if (nodeName) {
        pullRequestsByNodeName.set(nodeName, pullRequest);
      }
    }
  }

  return { byNodeId, byNodeName, pullRequestsByNodeId, pullRequestsByNodeName };
}

export function attachArtifactsToStream(
  stream: SplitRunStreamLine[] | undefined,
  index: StreamArtifactIndex,
): SplitRunStreamLine[] | undefined {
  if (!stream) {
    return undefined;
  }

  return stream.map((line) => attachLineArtifact(line, index));
}

export function attachStreamArtifacts(
  stream: SplitRunStreamLine[] | undefined,
  events: FactoriesWorkOrderEvent[],
  liveArtifacts?: FactoriesWorkOrderArtifact[],
  livePullRequests?: FactoriesFactoryPullRequest[],
): SplitRunStreamLine[] | undefined {
  return attachArtifactsToStream(stream, streamArtifactIndexFromEvents(events, liveArtifacts, livePullRequests));
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

function attachLineArtifact(line: SplitRunStreamLine, index: StreamArtifactIndex): SplitRunStreamLine {
  const next = { ...line };
  const artifactById = line.nodeId ? index.byNodeId.get(line.nodeId) : undefined;
  const artifactByName = index.byNodeName.get(line.componentName);
  if (artifactById || artifactByName) {
    next.artifact = artifactById ?? artifactByName;
  }

  const pullRequestById = line.nodeId ? index.pullRequestsByNodeId.get(line.nodeId) : undefined;
  const pullRequestByName = index.pullRequestsByNodeName.get(line.componentName);
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
