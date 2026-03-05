import { CanvasesCanvasVersion } from "@/api-client";
import { useMemo } from "react";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import * as yaml from "js-yaml";
import "react-diff-view/style/index.css";

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

export type DraftNodeDiffSummary = {
  items: DraftNodeDiffItem[];
  addedCount: number;
  updatedCount: number;
  removedCount: number;
};

export function buildDraftNodeDiffSummary(
  liveVersion?: CanvasesCanvasVersion,
  draftVersion?: CanvasesCanvasVersion,
): DraftNodeDiffSummary {
  const liveNodes = (liveVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;
  const draftNodes = (draftVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;

  const byID = (nodes: Array<Record<string, unknown>>): Map<string, Record<string, unknown>> => {
    const map = new Map<string, Record<string, unknown>>();
    nodes.forEach((node) => {
      const nodeID = String(node.id || "");
      if (nodeID) {
        map.set(nodeID, node);
      }
    });
    return map;
  };

  const comparableNode = (node: Record<string, unknown>) => ({
    name: node.name || null,
    type: node.type || null,
    ref: node.ref || null,
    configuration: node.configuration || null,
    position: node.position || null,
    isCollapsed: node.isCollapsed || false,
    integrationId: node.integrationId || null,
  });

  const formatDiffValueLines = (value: unknown): string[] => {
    const normalizedValue = value === undefined ? null : value;
    return yaml
      .dump(normalizedValue, {
        lineWidth: -1,
        noRefs: true,
        sortKeys: true,
      })
      .trimEnd()
      .split("\n");
  };

  const buildYamlFieldLines = (prefix: "+" | "-", key: string, value: unknown): DraftDiffLine[] => {
    const valueLines = formatDiffValueLines(value);
    if (valueLines.length === 1) {
      return [{ prefix, text: `${key}: ${valueLines[0]}` }];
    }

    return [{ prefix, text: `${key}:` }, ...valueLines.map((line) => ({ prefix, text: `  ${line}` }))];
  };

  const buildNodeLines = (prefix: "+" | "-", node: Record<string, unknown>): DraftDiffLine[] => {
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
  };

  const buildUpdatedLines = (
    previousNode: Record<string, unknown>,
    currentNode: Record<string, unknown>,
  ): DraftDiffLine[] => {
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
  };

  const liveByID = byID(liveNodes);
  const draftByID = byID(draftNodes);
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

function toUnifiedDiffText(lines: DraftDiffLine[]): string {
  return lines
    .map((line) => {
      if (line.prefix === "meta") {
        return line.text;
      }

      if (line.prefix === "context") {
        if (line.text.startsWith("@@")) {
          return line.text;
        }

        return ` ${line.text}`;
      }

      return `${line.prefix}${line.text}`;
    })
    .join("\n");
}

export function DraftNodeDiffView({ nodeID, lines }: { nodeID: string; lines: DraftDiffLine[] }) {
  const files = useMemo(() => parseDiff(toUnifiedDiffText(lines), { nearbySequences: "zip" }), [lines]);

  if (!files.length) {
    return <p className="text-xs text-slate-600">No diff available for this node.</p>;
  }

  return (
    <div className="overflow-hidden rounded-md border border-slate-200 bg-white">
      <div className="max-h-96 overflow-auto">
        {files.map((file) => (
          <Diff
            key={`${nodeID}-${file.oldRevision}-${file.newRevision}`}
            viewType="split"
            diffType={file.type}
            hunks={file.hunks}
          >
            {(hunks) => hunks.map((hunk) => <Hunk key={`${nodeID}-${hunk.content}`} hunk={hunk} />)}
          </Diff>
        ))}
      </div>
    </div>
  );
}
