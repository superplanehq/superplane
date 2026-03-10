import { CanvasesCanvasVersion } from "@/api-client";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { Avatar, AvatarFallback, AvatarImage } from "@/ui/avatar";
import { cn } from "@/lib/utils";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import { useMemo } from "react";
import * as yaml from "js-yaml";
import "react-diff-view/style/index.css";
import { WorkflowMarkdownPreview } from "./WorkflowMarkdownPreview";

export function buildInitials(name?: string): string {
  const safeName = (name || "").trim();
  if (!safeName) {
    return "U";
  }

  return safeName
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}

export function formatTimestamp(raw?: string): string | undefined {
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

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

export function summarizeNodeDiff(
  currentVersion?: CanvasesCanvasVersion,
  previousVersion?: CanvasesCanvasVersion,
): VersionNodeDiffSummary {
  const previousNodes = (previousVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;
  const currentNodes = (currentVersion?.spec?.nodes || []) as Array<Record<string, unknown>>;

  const toNodeMap = (nodes: Array<Record<string, unknown>>) => {
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
    const dumped = yaml
      .dump(normalizedValue, {
        lineWidth: -1,
        noRefs: true,
        sortKeys: true,
      })
      .trimEnd();
    return dumped.split("\n");
  };

  const buildYamlFieldLines = (prefix: "+" | "-", key: string, value: unknown): DiffLine[] => {
    const valueLines = formatDiffValueLines(value);
    if (valueLines.length === 1) {
      return [{ prefix, text: `${key}: ${valueLines[0]}` }];
    }

    return [{ prefix, text: `${key}:` }, ...valueLines.map((line) => ({ prefix, text: `  ${line}` }))];
  };

  const buildNodeLines = (prefix: "+" | "-", node: Record<string, unknown>): DiffLine[] => {
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
  };

  const buildUpdatedLines = (
    previousNode: Record<string, unknown>,
    currentNode: Record<string, unknown>,
  ): DiffLine[] => {
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
  };

  const previousByID = toNodeMap(previousNodes);
  const currentByID = toNodeMap(currentNodes);
  const allNodeIDs = Array.from(new Set([...previousByID.keys(), ...currentByID.keys()])).sort((left, right) =>
    left.localeCompare(right),
  );

  const items: VersionNodeDiffItem[] = [];
  let addedCount = 0;
  let removedCount = 0;
  let updatedCount = 0;

  allNodeIDs.forEach((nodeID) => {
    const previousNode = previousByID.get(nodeID);
    const currentNode = currentByID.get(nodeID);

    if (!previousNode && currentNode) {
      items.push({
        id: nodeID,
        name: String(currentNode.name || "Unnamed node"),
        changeType: "added",
        lines: buildNodeLines("+", currentNode),
      });
      addedCount += 1;
      return;
    }

    if (previousNode && !currentNode) {
      items.push({
        id: nodeID,
        name: String(previousNode.name || "Unnamed node"),
        changeType: "removed",
        lines: buildNodeLines("-", previousNode),
      });
      removedCount += 1;
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
      updatedCount += 1;
    }
  });

  return { items, addedCount, removedCount, updatedCount };
}

function toUnifiedDiffText(lines: DiffLine[]): string {
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

function NodeGitDiff({ lines, nodeID }: { lines: DiffLine[]; nodeID: string }) {
  const files = useMemo(() => parseDiff(toUnifiedDiffText(lines), { nearbySequences: "zip" }), [lines]);

  if (files.length === 0) {
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

export function VersionNodeDiffAccordion({
  summary,
  className,
  conflictingNodeIDs,
  emptyMessage = "No node changes found between these versions.",
}: {
  summary: VersionNodeDiffSummary;
  className?: string;
  conflictingNodeIDs?: Set<string>;
  emptyMessage?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <p className="text-sm text-slate-700">
        Added: {summary.addedCount} · Updated: {summary.updatedCount} · Removed: {summary.removedCount}
      </p>
      {summary.items.length === 0 ? (
        <p className="text-xs text-slate-600">{emptyMessage}</p>
      ) : (
        <Accordion type="multiple" className="w-full rounded-md border border-slate-200 px-3">
          {summary.items.map((item, index) => (
            <AccordionItem
              key={`${item.id}-${item.changeType}-${index}`}
              value={`${item.id}-${item.changeType}-${index}`}
              className="border-slate-200"
            >
              <AccordionTrigger className="py-3 hover:no-underline">
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className={cn(
                      "inline-flex min-w-8 justify-center rounded px-1.5 py-0.5 text-[11px] font-semibold",
                      item.changeType === "removed"
                        ? "bg-amber-100 text-amber-700"
                        : item.changeType === "added"
                          ? "bg-emerald-100 text-emerald-700"
                          : "bg-sky-100 text-sky-700",
                    )}
                  >
                    {item.changeType === "updated" ? "+/-" : item.changeType === "removed" ? "-" : "+"}
                  </span>
                  <span className="truncate text-sm text-slate-900">{item.name}</span>
                  <span className="truncate text-xs text-slate-500">{item.id}</span>
                  {conflictingNodeIDs?.has(item.id) ? (
                    <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-red-700">
                      conflict
                    </span>
                  ) : null}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                <NodeGitDiff lines={item.lines} nodeID={item.id} />
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      )}
    </div>
  );
}

export function ChangeRequestDescriptionCard({
  ownerName,
  ownerAvatarUrl,
  timestamp,
  content,
  actionLabel = "commented",
}: {
  ownerName: string;
  ownerAvatarUrl?: string;
  timestamp?: string;
  content: string;
  actionLabel?: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <Avatar className="mt-1 h-8 w-8">
        <AvatarImage src={ownerAvatarUrl} alt={ownerName} />
        <AvatarFallback className="text-[10px] font-medium">{buildInitials(ownerName)}</AvatarFallback>
      </Avatar>
      <div className="relative min-w-0 flex-1">
        <div className="rounded-md border border-slate-200 bg-white">
          <div className="relative border-b border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
            <span className="pointer-events-none absolute -left-2 top-1/2 z-10 h-0 w-0 -translate-y-1/2 border-y-[8px] border-y-transparent border-r-[8px] border-r-slate-200" />
            <span className="pointer-events-none absolute -left-[7px] top-1/2 z-10 h-0 w-0 -translate-y-1/2 border-y-[7px] border-y-transparent border-r-[7px] border-r-slate-50" />
            <span className="font-semibold text-slate-900">{ownerName}</span>
            <span>
              {" "}
              {actionLabel}
              {timestamp ? ` on ${timestamp}` : ""}
            </span>
          </div>
          <div className="p-3">
            <WorkflowMarkdownPreview content={content} />
          </div>
        </div>
      </div>
    </div>
  );
}
