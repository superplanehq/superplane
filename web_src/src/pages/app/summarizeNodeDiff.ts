import type { CanvasesCanvasVersion } from "@/api-client";
import * as yaml from "js-yaml";
import { getComparableIntegrationId } from "./utils";

type DiffLine = {
  prefix: "meta" | "context" | "+" | "-";
  text: string;
};

export type VersionNodeDiffItem = {
  id: string;
  name: string;
  changeType: "added" | "updated" | "removed";
  lines: DiffLine[];
};

export type VersionNodeDiffSummary = {
  items: VersionNodeDiffItem[];
  addedCount: number;
  updatedCount: number;
  removedCount: number;
};

function toNodeMap(nodes: Array<Record<string, unknown>>) {
  const map = new Map<string, Record<string, unknown>>();
  nodes.forEach((node) => {
    const nodeID = String(node.id || "");
    if (nodeID) {
      map.set(nodeID, node);
    }
  });
  return map;
}

function comparableNode(node: Record<string, unknown>) {
  return {
    name: node.name || null,
    type: node.type || null,
    ref: node.ref || null,
    configuration: node.configuration || null,
    position: node.position || null,
    isCollapsed: node.isCollapsed || false,
    integrationId: getComparableIntegrationId(node),
  };
}

function formatDiffValueLines(value: unknown): string[] {
  const normalizedValue = value === undefined ? null : value;
  const dumped = yaml
    .dump(normalizedValue, {
      lineWidth: -1,
      noRefs: true,
      sortKeys: true,
    })
    .trimEnd();
  return dumped.split("\n");
}

function buildYamlFieldLines(prefix: "+" | "-", key: string, value: unknown): DiffLine[] {
  const valueLines = formatDiffValueLines(value);
  if (valueLines.length === 1) {
    return [{ prefix, text: `${key}: ${valueLines[0]}` }];
  }

  return [{ prefix, text: `${key}:` }, ...valueLines.map((line) => ({ prefix, text: `  ${line}` }))];
}

function buildNodeLines(prefix: "+" | "-", node: Record<string, unknown>): DiffLine[] {
  const nodeComparable = comparableNode(node) as Record<string, unknown>;
  const keys = ["id", ...Object.keys(nodeComparable).sort((left, right) => left.localeCompare(right))];
  const nodeFilePath = `nodes/${String(node.id || "unknown")}.yaml`;
  const header: DiffLine[] = [
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

function buildUpdatedLines(previousNode: Record<string, unknown>, currentNode: Record<string, unknown>): DiffLine[] {
  const previousComparable = comparableNode(previousNode) as Record<string, unknown>;
  const currentComparable = comparableNode(currentNode) as Record<string, unknown>;
  const allKeys = ["id", ...Object.keys({ ...previousComparable, ...currentComparable })].sort((left, right) =>
    left.localeCompare(right),
  );

  const nodeFilePath = `nodes/${String(currentNode.id || previousNode.id || "unknown")}.yaml`;
  const lines: DiffLine[] = [
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

function appendNodeDiffItem(
  items: VersionNodeDiffItem[],
  counts: { addedCount: number; updatedCount: number; removedCount: number },
  nodeID: string,
  previousNode: Record<string, unknown> | undefined,
  currentNode: Record<string, unknown> | undefined,
) {
  if (!previousNode && currentNode) {
    items.push({
      id: nodeID,
      name: String(currentNode.name || "Unnamed node"),
      changeType: "added",
      lines: buildNodeLines("+", currentNode),
    });
    counts.addedCount += 1;
    return;
  }

  if (previousNode && !currentNode) {
    items.push({
      id: nodeID,
      name: String(previousNode.name || "Unnamed node"),
      changeType: "removed",
      lines: buildNodeLines("-", previousNode),
    });
    counts.removedCount += 1;
    return;
  }

  if (!previousNode || !currentNode) {
    return;
  }

  if (JSON.stringify(comparableNode(previousNode)) !== JSON.stringify(comparableNode(currentNode))) {
    items.push({
      id: nodeID,
      name: String(currentNode.name || previousNode.name || "Unnamed node"),
      changeType: "updated",
      lines: buildUpdatedLines(previousNode, currentNode),
    });
    counts.updatedCount += 1;
  }
}

export function summarizeNodeDiff(
  currentVersion?: CanvasesCanvasVersion,
  previousVersion?: CanvasesCanvasVersion,
): VersionNodeDiffSummary {
  const previousNodes = (previousVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;
  const currentNodes = (currentVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;

  const previousByID = toNodeMap(previousNodes);
  const currentByID = toNodeMap(currentNodes);
  const allNodeIDs = Array.from(new Set([...previousByID.keys(), ...currentByID.keys()])).sort((left, right) =>
    left.localeCompare(right),
  );

  const items: VersionNodeDiffItem[] = [];
  const counts = { addedCount: 0, updatedCount: 0, removedCount: 0 };

  allNodeIDs.forEach((nodeID) => {
    appendNodeDiffItem(items, counts, nodeID, previousByID.get(nodeID), currentByID.get(nodeID));
  });

  return { items, ...counts };
}
