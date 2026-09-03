import { History } from "lucide-react";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";

/**
 * A replay run carries a synthetic input event attributed to the node that
 * produced the replayed payload (pkg/models/canvas_node_replay.go). That node
 * is therefore a run participant even though it never executed in the replay,
 * so its card would otherwise fall back to the generic "Waiting for the first
 * run..." empty state and claim the node has never run at all.
 */
export const REPLAY_INPUT_EMPTY_STATE_TITLE = "Replay input came from here";
export const REPLAY_INPUT_EMPTY_STATE_DESCRIPTION = "This node did not run in the replay";

type ReceivedSourceNode = { nodeId?: string; receivedAt?: string };

// A multi-input replay writes one input event per incoming source, yet the run
// serializes a single one of them as its root event. The remaining sources are
// only recoverable from the replayed node's own metadata, where an aggregating
// component records every source it received (pkg/components/merge/merge.go).
// Only that execution counts: another aggregating node further down the run can
// group in events from outside the replay, and those sources are not replay
// inputs. The replayed node's execution is the one whose input event is the
// synthetic `replay` one (pkg/models/canvas_node_replay.go).
export function receivedReplayInputSourceNodeIds(execution: CanvasesCanvasNodeExecution): string[] {
  if (execution.inputEvent?.channel !== "replay") {
    return [];
  }

  const metadata = execution.metadata as { sourceNodes?: ReceivedSourceNode[] } | undefined;
  if (!Array.isArray(metadata?.sourceNodes)) {
    return [];
  }

  return metadata.sourceNodes.filter((source) => source.nodeId && source.receivedAt).map((source) => source.nodeId!);
}

export function collectReplayInputSourceNodeIds({
  isReplay,
  rootEventNodeId,
  executions,
  executedNodeIds,
}: {
  isReplay: boolean;
  rootEventNodeId?: string;
  executions: CanvasesCanvasNodeExecution[];
  executedNodeIds: Set<string>;
}): Set<string> {
  if (!isReplay) {
    return new Set();
  }

  const candidates = [rootEventNodeId, ...executions.flatMap(receivedReplayInputSourceNodeIds)];

  return new Set(candidates.filter((nodeId): nodeId is string => !!nodeId && !executedNodeIds.has(nodeId)));
}

export function annotateReplayInputSourceNodes(nodes: CanvasNode[], sourceNodeIds: Set<string>): CanvasNode[] {
  if (sourceNodeIds.size === 0) {
    return nodes;
  }

  return nodes.map((node) => {
    if (!sourceNodeIds.has(node.id)) {
      return node;
    }

    const data = node.data as Record<string, unknown> | undefined;
    if (!data || typeof data !== "object" || data.type !== "component") {
      return node;
    }

    const component = data.component;
    if (!component || typeof component !== "object") {
      return node;
    }

    const record = component as Record<string, unknown>;
    if (record.includeEmptyState !== true) {
      return node;
    }

    return {
      ...node,
      data: {
        ...data,
        component: {
          ...record,
          emptyStateProps: {
            icon: History,
            title: REPLAY_INPUT_EMPTY_STATE_TITLE,
            description: REPLAY_INPUT_EMPTY_STATE_DESCRIPTION,
            purpose: "runtime",
            tone: "neutral",
          },
        },
      },
    };
  });
}
