import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

import { buildLatestArtifactDataById, overlayLiveArtifactData } from "../../lib/workOrderArtifact";
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

export interface StreamArtifactIndex {
  byNodeId: Map<string, FactoriesWorkOrderArtifact>;
  byNodeName: Map<string, FactoriesWorkOrderArtifact>;
}

export function streamArtifactIndexFromEvents(
  events: FactoriesWorkOrderEvent[],
  liveArtifacts: FactoriesWorkOrderArtifact[] | undefined,
): StreamArtifactIndex {
  const byNodeId = new Map<string, FactoriesWorkOrderArtifact>();
  const byNodeName = new Map<string, FactoriesWorkOrderArtifact>();
  const liveById = liveArtifactsById(liveArtifacts);
  const latestDataById = buildLatestArtifactDataById(liveArtifacts);

  for (const event of sortEventsChronologically(events)) {
    if (event.type !== "order.artifact.added") {
      continue;
    }

    const payload = (event.event ?? {}) as ArtifactAddedPayload;
    const artifact = resolveAddedArtifact(payload.artifact, liveById, latestDataById);
    if (!artifact) {
      continue;
    }

    const nodeId = payload.automation?.nodeId?.trim();
    if (nodeId) {
      byNodeId.set(nodeId, artifact);
      continue;
    }

    const nodeName = payload.automation?.nodeName?.trim();
    if (nodeName) {
      byNodeName.set(nodeName, artifact);
    }
  }

  return { byNodeId, byNodeName };
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
): SplitRunStreamLine[] | undefined {
  return attachArtifactsToStream(stream, streamArtifactIndexFromEvents(events, liveArtifacts));
}

function attachLineArtifact(line: SplitRunStreamLine, index: StreamArtifactIndex): SplitRunStreamLine {
  const byId = line.nodeId ? index.byNodeId.get(line.nodeId) : undefined;
  if (byId) {
    return { ...line, artifact: byId };
  }

  const byName = index.byNodeName.get(line.componentName);
  if (byName) {
    return { ...line, artifact: byName };
  }

  return line;
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
