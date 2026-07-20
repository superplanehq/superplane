import type { CanvasesCanvasVersion } from "@/api-client";
import * as yaml from "js-yaml";
import { getComparableIntegrationId } from "./utils";

export type DraftDiffLine = {
  prefix: "meta" | "context" | "+" | "-";
  text: string;
};

export type DraftNodeDiffItem = {
  id: string;
  name: string;
  changeType: "added" | "updated" | "removed";
  lines: DraftDiffLine[];
};

export type DraftDiffStatus = DraftNodeDiffItem["changeType"];

export type DraftNodeDiffSummary = {
  items: DraftNodeDiffItem[];
  addedCount: number;
  updatedCount: number;
  removedCount: number;
};

function comparableEdgesSnapshot(edges: unknown): string {
  const list = (Array.isArray(edges) ? edges : []) as Array<Record<string, unknown>>;
  const normalized = list.map((edge) => ({
    sourceId: String(edge.sourceId ?? ""),
    targetId: String(edge.targetId ?? ""),
    channel: String(edge.channel ?? "default"),
  }));
  normalized.sort((left, right) => {
    const bySource = left.sourceId.localeCompare(right.sourceId);
    if (bySource !== 0) {
      return bySource;
    }

    const byTarget = left.targetId.localeCompare(right.targetId);
    if (byTarget !== 0) {
      return byTarget;
    }

    return left.channel.localeCompare(right.channel);
  });
  return JSON.stringify(normalized);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function normalizeComparableValue(value: unknown): unknown {
  if (value === undefined) {
    return null;
  }

  if (Array.isArray(value)) {
    return value.map(normalizeComparableValue);
  }

  if (!isRecord(value)) {
    return value;
  }

  const normalized = Object.keys(value)
    .sort((left, right) => left.localeCompare(right))
    .reduce<Record<string, unknown>>((acc, key) => {
      if (value[key] === undefined) {
        return acc;
      }

      acc[key] = normalizeComparableValue(value[key]);
      return acc;
    }, {});

  return Object.keys(normalized).length > 0 ? normalized : null;
}

function comparableNode(node: Record<string, unknown>) {
  return {
    name: node.name || null,
    type: node.type || null,
    ref: node.ref || null,
    configuration: normalizeComparableValue(node.configuration),
    position: normalizeComparableValue(node.position),
    isCollapsed: node.isCollapsed || false,
    integrationId: getComparableIntegrationId(node),
  };
}

function nodesByID(nodes: Array<Record<string, unknown>>): Map<string, Record<string, unknown>> {
  const map = new Map<string, Record<string, unknown>>();
  nodes.forEach((node) => {
    const nodeID = String(node.id || "");
    if (nodeID) {
      map.set(nodeID, node);
    }
  });
  return map;
}

function formatDiffValueLines(value: unknown): string[] {
  const normalizedValue = value === undefined ? null : value;
  return yaml
    .dump(normalizedValue, {
      lineWidth: -1,
      noRefs: true,
      sortKeys: true,
    })
    .trimEnd()
    .split("\n");
}

function buildYamlFieldLines(prefix: "+" | "-", key: string, value: unknown): DraftDiffLine[] {
  const valueLines = formatDiffValueLines(value);
  if (valueLines.length === 1) {
    return [{ prefix, text: `${key}: ${valueLines[0]}` }];
  }

  return [{ prefix, text: `${key}:` }, ...valueLines.map((line) => ({ prefix, text: `  ${line}` }))];
}

function buildNodeLines(prefix: "+" | "-", node: Record<string, unknown>): DraftDiffLine[] {
  const nodeComparable = comparableNode(node) as Record<string, unknown>;
  const keys = ["id", ...Object.keys(nodeComparable).sort((left, right) => left.localeCompare(right))];
  const nodeFilePath = `nodes/${String(node.id || "unknown")}.yaml`;
  const header: DraftDiffLine[] = [
    { prefix: "meta", text: `diff --git a/${nodeFilePath} b/${nodeFilePath}` },
    { prefix: "meta", text: `--- ${prefix === "-" ? `a/${nodeFilePath}` : "/dev/null"}` },
    { prefix: "meta", text: `+++ ${prefix === "+" ? `b/${nodeFilePath}` : "/dev/null"}` },
    { prefix: "context", text: "@@ -1,0 +1,0 @@" },
  ];

  return keys.flatMap((key) => {
    const value = key === "id" ? node.id : nodeComparable[key];
    return [...(key === "id" ? header : []), ...buildYamlFieldLines(prefix, key, value)];
  });
}

function buildUpdatedLines(
  previousNode: Record<string, unknown>,
  currentNode: Record<string, unknown>,
): DraftDiffLine[] {
  const previousComparable = comparableNode(previousNode) as Record<string, unknown>;
  const currentComparable = comparableNode(currentNode) as Record<string, unknown>;
  const allKeys = ["id", ...Object.keys({ ...previousComparable, ...currentComparable })].sort((left, right) =>
    left.localeCompare(right),
  );

  const nodeFilePath = `nodes/${String(currentNode.id || previousNode.id || "unknown")}.yaml`;
  const lines: DraftDiffLine[] = [
    { prefix: "meta", text: `diff --git a/${nodeFilePath} b/${nodeFilePath}` },
    { prefix: "meta", text: `--- a/${nodeFilePath}` },
    { prefix: "meta", text: `+++ b/${nodeFilePath}` },
    { prefix: "context", text: "@@ -1,0 +1,0 @@" },
  ];

  allKeys.forEach((key) => {
    const previousValue = key === "id" ? previousNode.id : previousComparable[key];
    const currentValue = key === "id" ? currentNode.id : currentComparable[key];
    if (JSON.stringify(previousValue) === JSON.stringify(currentValue)) {
      return;
    }

    lines.push(...buildYamlFieldLines("-", key, previousValue));
    lines.push(...buildYamlFieldLines("+", key, currentValue));
  });

  return lines;
}

/** True when draft workflow graph differs from live (nodes and/or edges). */
export function hasDraftVersusLiveGraphDiff(
  liveVersion?: CanvasesCanvasVersion,
  draftVersion?: CanvasesCanvasVersion,
): boolean {
  const { items } = buildDraftNodeDiffSummary(liveVersion, draftVersion);
  if (items.length > 0) {
    return true;
  }

  return comparableEdgesSnapshot(liveVersion?.spec?.edges) !== comparableEdgesSnapshot(draftVersion?.spec?.edges);
}

export function buildDraftNodeDiffSummary(
  liveVersion?: CanvasesCanvasVersion,
  draftVersion?: CanvasesCanvasVersion,
): DraftNodeDiffSummary {
  const liveNodes = (liveVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;
  const draftNodes = (draftVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;

  const liveByID = nodesByID(liveNodes);
  const draftByID = nodesByID(draftNodes);
  const allNodeIDs = Array.from(new Set([...liveByID.keys(), ...draftByID.keys()])).sort((left, right) =>
    left.localeCompare(right),
  );

  const items: DraftNodeDiffItem[] = [];
  let addedCount = 0;
  let removedCount = 0;
  let updatedCount = 0;

  allNodeIDs.forEach((nodeID) => {
    const liveNode = liveByID.get(nodeID);
    const draftNode = draftByID.get(nodeID);

    if (!liveNode && draftNode) {
      items.push({
        id: nodeID,
        name: String(draftNode.name || "Unnamed node"),
        changeType: "added",
        lines: buildNodeLines("+", draftNode),
      });
      addedCount += 1;
      return;
    }

    if (liveNode && !draftNode) {
      items.push({
        id: nodeID,
        name: String(liveNode.name || "Unnamed node"),
        changeType: "removed",
        lines: buildNodeLines("-", liveNode),
      });
      removedCount += 1;
      return;
    }

    if (!liveNode || !draftNode) {
      return;
    }

    if (JSON.stringify(comparableNode(liveNode)) !== JSON.stringify(comparableNode(draftNode))) {
      items.push({
        id: nodeID,
        name: String(draftNode.name || liveNode.name || "Unnamed node"),
        changeType: "updated",
        lines: buildUpdatedLines(liveNode, draftNode),
      });
      updatedCount += 1;
    }
  });

  return { items, addedCount, updatedCount, removedCount };
}

/**
 * Returns node diff state for canvas rendering. Position-only changes are ignored
 * so moving a node does not mark it visually edited.
 */
export function buildDraftDiffMap(
  liveVersion?: CanvasesCanvasVersion,
  draftVersion?: CanvasesCanvasVersion,
): {
  statusMap: Record<string, DraftDiffStatus>;
  removedNodes: Array<Record<string, unknown>>;
} {
  const liveNodes = (liveVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;
  const draftNodes = (draftVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;

  const functionalSnapshot = (node: Record<string, unknown>) =>
    JSON.stringify({
      name: node.name || null,
      type: node.type || null,
      ref: node.ref || null,
      configuration: normalizeComparableValue(node.configuration),
      isCollapsed: node.isCollapsed || false,
      integrationId: getComparableIntegrationId(node),
    });

  const liveByID = nodesByID(liveNodes);
  const draftByID = nodesByID(draftNodes);
  const allNodeIDs = new Set([...liveByID.keys(), ...draftByID.keys()]);
  const statusMap: Record<string, DraftDiffStatus> = {};

  for (const nodeID of allNodeIDs) {
    const liveNode = liveByID.get(nodeID);
    const draftNode = draftByID.get(nodeID);

    if (!liveNode && draftNode) {
      statusMap[nodeID] = "added";
      continue;
    }

    if (liveNode && !draftNode) {
      statusMap[nodeID] = "removed";
      continue;
    }

    if (liveNode && draftNode && functionalSnapshot(liveNode) !== functionalSnapshot(draftNode)) {
      statusMap[nodeID] = "updated";
    }
  }

  const removedNodes = liveNodes.filter((node) => statusMap[String(node.id)] === "removed");

  return { statusMap, removedNodes };
}
