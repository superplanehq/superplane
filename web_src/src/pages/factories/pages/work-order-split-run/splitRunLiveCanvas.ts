import type {
  CanvasesCanvas,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import {
  canvasKeyForAutomation,
  canvasKeyForPhase,
  componentPresentation,
  emptySplitRunCanvas,
  richStreamForCanvas,
  splitRunCanvasForPhase,
  type SplitRunCanvasModel,
} from "./splitRunCanvases";
import { clockLabel } from "./splitRunFormat";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";

export function nodeStatusFromExecution(
  execution: CanvasesCanvasNodeExecutionRef | undefined,
  isTrigger: boolean,
): FactoryNodeStatus {
  if (!execution) {
    return "did_not_run";
  }
  if (execution.state === "STATE_STARTED") {
    return "running";
  }
  if (execution.state === "STATE_CANCELLING") {
    return "cancelling";
  }
  if (execution.state === "STATE_PENDING") {
    return "pending";
  }
  if (execution.result === "RESULT_FAILED") {
    return "failed";
  }
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }
  if (execution.result === "RESULT_PASSED") {
    return isTrigger ? "triggered" : "passed";
  }
  return "pending";
}

export function metricFromExecution(execution: CanvasesCanvasNodeExecutionRef | undefined): string {
  if (!execution) {
    return "—";
  }
  const start = Date.parse(execution.createdAt ?? "");
  if (!Number.isFinite(start)) {
    return "—";
  }
  const end = Date.parse(execution.updatedAt ?? "") || start;
  return formatMinutesSecondsDuration(Math.max(0, end - start)) || "—";
}

export function streamStatusFromNode(status: FactoryNodeStatus): SplitRunPhaseStatus {
  if (status === "running" || status === "cancelling") {
    return "running";
  }
  if (status === "failed" || status === "error") {
    return "failed";
  }
  if (status === "passed" || status === "triggered") {
    return "passed";
  }
  return "pending";
}

function createdAtMs(value?: string): number {
  const time = Date.parse(value ?? "");
  return Number.isFinite(time) ? time : Number.POSITIVE_INFINITY;
}

function latestExecutionByNode(
  executions: CanvasesCanvasNodeExecutionRef[] | undefined,
): Map<string, CanvasesCanvasNodeExecutionRef> {
  const latest = new Map<string, CanvasesCanvasNodeExecutionRef>();
  for (const execution of executions ?? []) {
    if (!execution.nodeId) {
      continue;
    }
    const current = latest.get(execution.nodeId);
    if (!current || createdAtMs(execution.createdAt) >= createdAtMs(current.createdAt)) {
      latest.set(execution.nodeId, execution);
    }
  }
  return latest;
}

function triggerIdForRootEvent(
  run: CanvasesCanvasRun | undefined,
  nodes: Array<ComponentsNode & { id: string }>,
): string | undefined {
  const rootNodeId = run?.rootEvent?.nodeId;
  if (rootNodeId && nodes.some((node) => node.id === rootNodeId)) {
    return rootNodeId;
  }
  return nodes.find((node) => node.type === "TYPE_TRIGGER")?.id;
}

function attachTriggerFromRootEvent(
  run: CanvasesCanvasRun | undefined,
  nodes: Array<ComponentsNode & { id: string }>,
  latest: Map<string, CanvasesCanvasNodeExecutionRef>,
): void {
  const at = run?.rootEvent?.createdAt ?? run?.createdAt;
  const triggerId = triggerIdForRootEvent(run, nodes);
  if (!at || !triggerId) {
    return;
  }
  const existing = latest.get(triggerId);
  if (!existing) {
    latest.set(triggerId, {
      nodeId: triggerId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: at,
      updatedAt: at,
    });
    return;
  }
  if (existing.createdAt) {
    return;
  }
  latest.set(triggerId, { ...existing, createdAt: at, updatedAt: existing.updatedAt ?? at });
}

function liveExecutionsByNode(
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): { nodes: Array<ComponentsNode & { id: string }>; latest: Map<string, CanvasesCanvasNodeExecutionRef> } {
  const nodes = (canvas?.spec?.nodes ?? []).filter((node): node is ComponentsNode & { id: string } => Boolean(node.id));
  const latest = latestExecutionByNode(run?.executions);
  attachTriggerFromRootEvent(run, nodes, latest);
  return { nodes, latest };
}

function streamLineForNode(
  node: ComponentsNode & { id: string },
  execution: CanvasesCanvasNodeExecutionRef | undefined,
): SplitRunStreamLine {
  const presentation = componentPresentation(node.component);
  const status = nodeStatusFromExecution(execution, node.type === "TYPE_TRIGGER");
  const duration = metricFromExecution(execution);
  return {
    id: node.id,
    nodeId: node.id,
    at: clockLabel(execution?.createdAt),
    componentName: node.name ?? presentation.title,
    status: streamStatusFromNode(status),
    duration: duration === "—" ? undefined : duration,
  };
}

export function splitRunCanvasFromLive(input: {
  canvas: CanvasesCanvas;
  run?: CanvasesCanvasRun;
  fallbackTitle: string;
  key?: SplitRunCanvasModel["key"];
}): SplitRunCanvasModel | undefined {
  const { nodes, latest } = liveExecutionsByNode(input.canvas, input.run);
  if (!nodes.length) {
    return undefined;
  }
  const statuses: Record<string, FactoryNodeStatus> = {};
  const metrics: Record<string, string> = {};
  for (const node of nodes) {
    const execution = latest.get(node.id);
    statuses[node.id] = nodeStatusFromExecution(execution, node.type === "TYPE_TRIGGER");
    metrics[node.id] = metricFromExecution(execution);
  }
  return {
    key:
      input.key ??
      canvasKeyForAutomation({
        id: input.canvas.metadata?.id,
        name: input.canvas.metadata?.name,
      }) ??
      "implementation",
    title: input.canvas.metadata?.name ?? input.fallbackTitle,
    nodes,
    edges: input.canvas.spec?.edges ?? [],
    statuses,
    metrics,
  };
}

export function streamFromLiveRun(
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): SplitRunStreamLine[] {
  const { nodes, latest } = liveExecutionsByNode(canvas, run);
  if (nodes.length === 0 && latest.size === 0) {
    return [];
  }
  const seen = new Set<string>();
  const lines = nodes.map((node) => {
    seen.add(node.id);
    return streamLineForNode(node, latest.get(node.id));
  });
  for (const execution of latest.values()) {
    if (!execution.nodeId || seen.has(execution.nodeId)) {
      continue;
    }
    lines.push(streamLineForNode({ id: execution.nodeId, name: execution.nodeId, type: "TYPE_ACTION" }, execution));
  }
  return lines.sort((left, right) => {
    const delta =
      createdAtMs(latest.get(left.nodeId ?? left.id)?.createdAt) -
      createdAtMs(latest.get(right.nodeId ?? right.id)?.createdAt);
    if (delta !== 0) {
      return delta;
    }
    return (left.componentName ?? "").localeCompare(right.componentName ?? "");
  });
}

export function resolveSplitRunVisual(
  phase: SplitRunPhase,
  live: { enabled: boolean; isError?: boolean; canvas?: SplitRunCanvasModel; stream: SplitRunStreamLine[] },
): { canvas: SplitRunCanvasModel; stream: SplitRunStreamLine[] | undefined } {
  if (live.isError) {
    return { canvas: emptySplitRunCanvas(phase), stream: [] };
  }
  if (live.enabled) {
    return {
      canvas: live.canvas ?? emptySplitRunCanvas(phase),
      stream: live.stream.length > 0 ? live.stream : phase.stream,
    };
  }
  const canvas = splitRunCanvasForPhase(phase);
  return { canvas, stream: richStreamForCanvas(canvas) };
}

export function liveCanvasKeyForPhase(phase: SplitRunPhase): SplitRunCanvasModel["key"] {
  return canvasKeyForAutomation({ id: phase.appId, name: phase.componentName }) ?? canvasKeyForPhase(phase);
}
