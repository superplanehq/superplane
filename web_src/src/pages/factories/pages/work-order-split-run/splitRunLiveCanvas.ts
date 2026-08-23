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
  componentTypeLabel,
  emptySplitRunCanvas,
  richStreamForCanvas,
  splitRunCanvasForPhase,
  streamKindForNode,
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

function compareReadyCanvasNodes(
  left: { id: string; type?: string },
  right: { id: string; type?: string },
  indexById: Map<string, number>,
): number {
  const leftTrigger = left.type === "TYPE_TRIGGER" ? 0 : 1;
  const rightTrigger = right.type === "TYPE_TRIGGER" ? 0 : 1;
  if (leftTrigger !== rightTrigger) {
    return leftTrigger - rightTrigger;
  }
  return (indexById.get(left.id) ?? 0) - (indexById.get(right.id) ?? 0);
}

function emptyCanvasGraph(nodes: Array<{ id: string }>): {
  outgoing: Map<string, string[]>;
  indegree: Map<string, number>;
} {
  const outgoing = new Map<string, string[]>();
  const indegree = new Map<string, number>();
  for (const node of nodes) {
    outgoing.set(node.id, []);
    indegree.set(node.id, 0);
  }
  return { outgoing, indegree };
}

function addCanvasEdge(
  outgoing: Map<string, string[]>,
  indegree: Map<string, number>,
  sourceId: string,
  targetId: string,
) {
  const next = outgoing.get(sourceId) ?? [];
  if (next.includes(targetId)) {
    return;
  }
  next.push(targetId);
  outgoing.set(sourceId, next);
  indegree.set(targetId, (indegree.get(targetId) ?? 0) + 1);
}

function fillCanvasGraph(
  outgoing: Map<string, string[]>,
  indegree: Map<string, number>,
  edges: Array<{ sourceId?: string; targetId?: string }> | undefined,
  knownIds: Set<string>,
) {
  for (const edge of edges ?? []) {
    const sourceId = edge.sourceId;
    const targetId = edge.targetId;
    if (!sourceId || !targetId || sourceId === targetId || !knownIds.has(sourceId) || !knownIds.has(targetId)) {
      continue;
    }
    addCanvasEdge(outgoing, indegree, sourceId, targetId);
  }
}

function enqueueReadyTargets<T extends { id: string; type?: string }>(
  nodeId: string,
  graph: { outgoing: Map<string, string[]>; indegree: Map<string, number> },
  ctx: { nodes: T[]; indexById: Map<string, number>; ready: T[] },
) {
  for (const targetId of graph.outgoing.get(nodeId) ?? []) {
    const nextDegree = (graph.indegree.get(targetId) ?? 1) - 1;
    graph.indegree.set(targetId, nextDegree);
    if (nextDegree !== 0) {
      continue;
    }
    const target = ctx.nodes[ctx.indexById.get(targetId) ?? -1];
    if (!target) {
      continue;
    }
    ctx.ready.push(target);
    ctx.ready.sort((left, right) => compareReadyCanvasNodes(left, right, ctx.indexById));
  }
}

function walkCanvasOrder<T extends { id: string; type?: string }>(
  nodes: T[],
  graph: { outgoing: Map<string, string[]>; indegree: Map<string, number> },
  indexById: Map<string, number>,
): string[] {
  const ready = nodes.filter((node) => graph.indegree.get(node.id) === 0);
  ready.sort((left, right) => compareReadyCanvasNodes(left, right, indexById));
  const orderedIds: string[] = [];
  while (ready.length > 0) {
    const node = ready.shift();
    if (!node) {
      break;
    }
    orderedIds.push(node.id);
    enqueueReadyTargets(node.id, graph, { nodes, indexById, ready });
  }
  return orderedIds;
}

function completeCanvasOrder<T extends { id: string }>(orderedIds: string[], nodes: T[]): string[] {
  const seen = new Set(orderedIds);
  for (const node of nodes) {
    if (!seen.has(node.id)) {
      orderedIds.push(node.id);
    }
  }
  return orderedIds;
}

export function orderCanvasNodesTopologically<T extends { id: string; type?: string }>(
  nodes: T[],
  edges: Array<{ sourceId?: string; targetId?: string }> | undefined,
): T[] {
  if (nodes.length < 2) {
    return nodes;
  }

  const indexById = new Map(nodes.map((node, index) => [node.id, index]));
  const graph = emptyCanvasGraph(nodes);
  fillCanvasGraph(graph.outgoing, graph.indegree, edges, new Set(indexById.keys()));
  const orderedIds = completeCanvasOrder(walkCanvasOrder(nodes, graph, indexById), nodes);
  return orderedIds.map((id) => nodes[indexById.get(id) ?? -1]).filter((node): node is T => Boolean(node));
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
    kind: streamKindForNode(node),
    componentType: componentTypeLabel(node.component),
    action: liveStreamAction(status, Boolean(execution)),
    iconSlug: presentation.iconSlug,
    iconSrc: presentation.iconSrc,
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
  const lines = orderCanvasNodesTopologically(nodes, canvas?.spec?.edges).map((node) => {
    seen.add(node.id);
    return streamLineForNode(node, latest.get(node.id));
  });
  for (const execution of latest.values()) {
    if (!execution.nodeId || seen.has(execution.nodeId)) {
      continue;
    }
    lines.push(streamLineForNode({ id: execution.nodeId, name: execution.nodeId, type: "TYPE_ACTION" }, execution));
  }
  return lines;
}

function liveCanvasMatchesLineAutomation(line: SplitRunCanvasModel, live: SplitRunCanvasModel): boolean {
  const liveIds = new Set(live.nodes.map((node) => node.id).filter((id): id is string => Boolean(id)));
  const lineIds = line.nodes.map((node) => node.id).filter((id): id is string => Boolean(id));
  if (lineIds.length === 0) {
    return false;
  }
  const overlap = lineIds.filter((id) => liveIds.has(id)).length;
  return overlap >= Math.ceil(lineIds.length / 2);
}

export function resolveSplitRunVisual(
  phase: SplitRunPhase,
  live: { enabled: boolean; isError?: boolean; canvas?: SplitRunCanvasModel; stream: SplitRunStreamLine[] },
): { canvas: SplitRunCanvasModel; stream: SplitRunStreamLine[] | undefined } {
  const lineCanvas = splitRunCanvasForPhase(phase);
  const lineStream = richStreamForCanvas(lineCanvas, descriptionArtifactFromPhase(phase));
  if (phase.canvasKey === null || lineCanvas.nodes.length === 0) {
    return { canvas: lineCanvas, stream: phase.stream };
  }
  if (live.isError) {
    return { canvas: emptySplitRunCanvas(phase), stream: [] };
  }
  if (live.canvas && liveCanvasMatchesLineAutomation(lineCanvas, live.canvas)) {
    return {
      canvas: live.canvas,
      stream: live.stream.length > 0 ? attachMissingStreamArtifacts(live.stream, lineStream) : lineStream,
    };
  }
  return { canvas: lineCanvas, stream: lineStream };
}

function attachMissingStreamArtifacts(
  stream: SplitRunStreamLine[],
  source: SplitRunStreamLine[],
): SplitRunStreamLine[] {
  const artifacts = new Map<string, NonNullable<SplitRunStreamLine["artifact"]>>();
  for (const line of source) {
    if (line.nodeId && line.artifact) {
      artifacts.set(line.nodeId, line.artifact);
    }
  }
  return stream.map((line) => {
    if (line.artifact || !line.nodeId) {
      return line;
    }
    const artifact = artifacts.get(line.nodeId);
    return artifact ? { ...line, artifact } : line;
  });
}

export function liveCanvasKeyForPhase(phase: SplitRunPhase): SplitRunCanvasModel["key"] {
  return canvasKeyForAutomation({ id: phase.appId, name: phase.componentName }) ?? canvasKeyForPhase(phase);
}

function descriptionArtifactFromPhase(phase: SplitRunPhase) {
  return phase.artifacts.find((artifact) => {
    const data = artifact.data;
    return Boolean(data && "name" in data && data.name === "description.md");
  });
}

function liveStreamAction(status: FactoryNodeStatus, hasExecution: boolean): string {
  if (!hasExecution) {
    return "did not run";
  }
  if (status === "running" || status === "cancelling") {
    return "running";
  }
  if (status === "failed" || status === "error") {
    return "failed";
  }
  if (status === "triggered") {
    return "triggered";
  }
  if (status === "passed") {
    return "passed";
  }
  return "—";
}
