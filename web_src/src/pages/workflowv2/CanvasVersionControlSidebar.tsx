import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { Avatar, AvatarFallback, AvatarImage } from "@/ui/avatar";
import { cn } from "@/lib/utils";
import { ChevronLeft, Eye, GitBranch, GitCompareArrows } from "lucide-react";
import { MouseEvent as ReactMouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as yaml from "js-yaml";
import ReactMarkdown from "react-markdown";
import { Diff, Hunk, parseDiff } from "react-diff-view";
import "react-diff-view/style/index.css";

const CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY = "canvasVersionControlSidebarWidth";
const DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 460;
const LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 340;
const MIN_CANVAS_VERSION_CONTROL_WIDTH = 280;
const MAX_CANVAS_VERSION_CONTROL_WIDTH = 640;

interface CanvasVersionControlSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  liveCanvasVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveVersions: CanvasesCanvasVersion[];
  liveVersionChangeRequestsByVersionId?: Map<string, CanvasesCanvasChangeRequest>;
  liveVersionOwnerProfilesById?: Map<string, { name: string; avatarUrl?: string }>;
  liveVersionsTotalCount?: number;
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
}

function buildInitials(name?: string): string {
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

function formatVersionTimestamp(version?: CanvasesCanvasVersion): string | undefined {
  const raw = version?.metadata?.updatedAt || version?.metadata?.publishedAt || version?.metadata?.createdAt;
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function formatVersionLabel(version?: CanvasesCanvasVersion): string {
  if (version?.metadata?.isPublished) {
    return "Published version";
  }

  return "Draft version";
}

function formatVersionLabelWithTimestamp(version?: CanvasesCanvasVersion): string {
  const label = formatVersionLabel(version);
  const timestamp = formatVersionTimestamp(version);
  if (!timestamp) {
    return label;
  }

  return `${label} · ${timestamp}`;
}

function formatTimestamp(raw?: string): string | undefined {
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

type VersionNodeDiffItem = {
  id: string;
  name: string;
  changeType: "added" | "updated" | "removed";
  lines: DiffLine[];
};

type VersionNodeDiffSummary = {
  items: VersionNodeDiffItem[];
  addedCount: number;
  updatedCount: number;
  removedCount: number;
};

function summarizeNodeDiff(
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

export function CanvasVersionControlSidebar({
  isOpen,
  onToggle,
  liveCanvasVersionId,
  selectedCanvasVersion,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  liveVersionOwnerProfilesById,
  liveVersionsTotalCount,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  onUseVersion,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
}: CanvasVersionControlSidebarProps) {
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";

  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    const stored = window.localStorage.getItem(CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY);
    const parsed = stored ? Number.parseInt(stored, 10) : NaN;
    if (!Number.isFinite(parsed)) {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    if (parsed === LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH) {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    return Math.max(MIN_CANVAS_VERSION_CONTROL_WIDTH, Math.min(MAX_CANVAS_VERSION_CONTROL_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);
  const sidebarRef = useRef<HTMLElement>(null);
  const [diffContext, setDiffContext] = useState<{
    version: CanvasesCanvasVersion;
    previousVersion: CanvasesCanvasVersion;
    changeRequest?: CanvasesCanvasChangeRequest;
  } | null>(null);

  const diffSummary = useMemo(() => {
    if (!diffContext) {
      return null;
    }
    return summarizeNodeDiff(diffContext.version, diffContext.previousVersion);
  }, [diffContext]);
  const diffOwner = useMemo(() => {
    const changeRequestOwner = diffContext?.changeRequest?.metadata?.owner;
    if (!changeRequestOwner) {
      return null;
    }

    const profile = changeRequestOwner.id ? liveVersionOwnerProfilesById?.get(changeRequestOwner.id) : undefined;
    const name = changeRequestOwner.name || profile?.name || "Unknown user";

    return {
      name,
      avatarUrl: profile?.avatarUrl,
      initials: buildInitials(name),
    };
  }, [diffContext, liveVersionOwnerProfilesById]);
  const diffCommentTimestamp = useMemo(
    () =>
      formatTimestamp(
        diffContext?.changeRequest?.metadata?.updatedAt || diffContext?.changeRequest?.metadata?.createdAt,
      ),
    [diffContext],
  );

  const handleMouseDown = useCallback((event: ReactMouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (event: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const sidebarLeft = sidebarRef.current?.getBoundingClientRect().left ?? 0;
      const newWidth = event.clientX - sidebarLeft;
      const clampedWidth = Math.max(
        MIN_CANVAS_VERSION_CONTROL_WIDTH,
        Math.min(MAX_CANVAS_VERSION_CONTROL_WIDTH, newWidth),
      );
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.localStorage.setItem(CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  if (!isOpen) {
    return null;
  }

  return (
    <aside
      ref={sidebarRef}
      className="z-20 h-full border-r border-slate-950/10 bg-white relative"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      <div
        onMouseDown={handleMouseDown}
        className={`absolute right-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-slate-100 transition-colors flex items-center justify-center group z-30 ${
          isResizing ? "bg-sky-50" : ""
        }`}
        style={{ marginRight: "-8px" }}
      >
        <div
          className={`h-14 w-2 rounded-full bg-slate-300 transition-colors ${
            isResizing ? "bg-sky-500" : "group-hover:bg-slate-600"
          }`}
        />
      </div>
      <div className="flex h-full flex-col">
        <div className="flex h-12 items-center justify-between border-b border-slate-200 px-3">
          <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
            <GitBranch className="h-4 w-4" />
            Version Control
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            className="h-7 w-7"
            onClick={() => onToggle(false)}
            aria-label="Collapse version control"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex-1 overflow-auto p-3">
          {!canUpdateCanvas && !canvasDeletedRemotely ? (
            <p className="text-xs text-slate-600">You do not have permission to edit this canvas.</p>
          ) : null}
          {canvasDeletedRemotely ? (
            <p className="text-xs text-red-700">This canvas was deleted from another session.</p>
          ) : null}
          {isTemplate ? <p className="text-xs text-slate-600">Template canvases are read-only.</p> : null}

          <section className="mt-3 rounded-md">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
              Live History ({liveVersionsTotalCount ?? liveVersions.length})
            </p>
            {liveVersions.length === 0 ? (
              <p className="mt-2 text-xs text-slate-600">No published history yet.</p>
            ) : (
              <>
                <div className="mt-2 space-y-2">
                  {liveVersions.map((version, index) => {
                    const versionID = version.metadata?.id || "";
                    const isActive = versionID === selectedVersionId;
                    const isCurrentLive = liveCanvasVersionId === versionID;
                    const previousVersion = liveVersions[index + 1];
                    const changeRequest = versionID ? liveVersionChangeRequestsByVersionId?.get(versionID) : undefined;

                    return (
                      <VersionRow
                        key={versionID}
                        version={version}
                        changeRequest={changeRequest}
                        isActive={isActive}
                        subtitle={isCurrentLive ? "Current live" : "Live history"}
                        previousVersion={previousVersion}
                        onUseVersion={onUseVersion}
                        onViewDiff={(selectedVersion, selectedPreviousVersion, selectedChangeRequest) =>
                          setDiffContext({
                            version: selectedVersion,
                            previousVersion: selectedPreviousVersion,
                            changeRequest: selectedChangeRequest,
                          })
                        }
                      />
                    );
                  })}
                </div>
                {onLoadMoreLiveVersions ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 w-full"
                    onClick={onLoadMoreLiveVersions}
                    disabled={loadMoreLiveVersionsDisabled}
                  >
                    {loadMoreLiveVersionsPending ? "Loading..." : "Load older versions"}
                  </Button>
                ) : null}
              </>
            )}
          </section>
        </div>
      </div>

      <Dialog open={!!diffContext} onOpenChange={(open) => !open && setDiffContext(null)}>
        <DialogContent className="min-w-[60vw] max-w-5xl max-h-[92vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{diffContext?.changeRequest?.metadata?.title?.trim() || "Version Node Diff"}</DialogTitle>
            <DialogDescription>
              Comparing {formatVersionLabelWithTimestamp(diffContext?.version)} against the previous published version.
            </DialogDescription>
          </DialogHeader>

          {!diffSummary ? null : (
            <div className="space-y-3">
              {diffContext?.changeRequest?.metadata?.description?.trim() ? (
                <div className="flex items-start gap-3">
                  {diffOwner ? (
                    <Avatar className="h-8 w-8 mt-1">
                      <AvatarImage src={diffOwner.avatarUrl} alt={diffOwner.name} />
                      <AvatarFallback className="text-[10px] font-medium">{diffOwner.initials}</AvatarFallback>
                    </Avatar>
                  ) : null}
                  <div className="relative min-w-0 flex-1">
                    <div className="rounded-md border border-slate-200 bg-white">
                      <div className="relative border-b border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
                        <span className="pointer-events-none absolute -left-2 top-1/2 z-10 h-0 w-0 -translate-y-1/2 border-y-[8px] border-y-transparent border-r-[8px] border-r-slate-200" />
                        <span className="pointer-events-none absolute -left-[7px] top-1/2 z-10 h-0 w-0 -translate-y-1/2 border-y-[7px] border-y-transparent border-r-[7px] border-r-slate-50" />
                        <span className="font-semibold text-slate-900">{diffOwner?.name || "Unknown user"}</span>
                        <span>
                          {" "}
                          commented
                          {diffCommentTimestamp ? ` on ${diffCommentTimestamp}` : ""}
                        </span>
                      </div>
                      <div className="p-3">
                        <div className="prose prose-sm max-w-none text-slate-800 prose-p:my-2 prose-ul:my-2 prose-ol:my-2 prose-headings:my-2">
                          <ReactMarkdown>{diffContext.changeRequest.metadata?.description?.trim() || ""}</ReactMarkdown>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              ) : null}
              <div className="ml-11 flex flex-col gap-2">
                <p className="text-sm text-slate-700">
                  Added: {diffSummary.addedCount} · Updated: {diffSummary.updatedCount} · Removed:{" "}
                  {diffSummary.removedCount}
                </p>
                {diffSummary.items.length === 0 ? (
                  <p className="text-xs text-slate-600">No node changes found between these versions.</p>
                ) : (
                  <Accordion type="multiple" className="w-full rounded-md border border-slate-200 px-3">
                    {diffSummary.items.map((item, index) => (
                      <AccordionItem
                        key={`${item.id}-${item.changeType}-${index}`}
                        value={`${item.id}-${item.changeType}-${index}`}
                        className="border-slate-200"
                      >
                        <AccordionTrigger className="py-3 hover:no-underline">
                          <div className="flex items-center gap-2 min-w-0">
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
            </div>
          )}
        </DialogContent>
      </Dialog>
    </aside>
  );
}

function VersionRow({
  version,
  changeRequest,
  previousVersion,
  isActive = false,
  subtitle,
  onUseVersion,
  onViewDiff,
}: {
  version: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
  previousVersion?: CanvasesCanvasVersion;
  isActive?: boolean;
  subtitle?: string;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  const versionID = version.metadata?.id;
  const ownerName = version.metadata?.owner?.name || "Unknown owner";
  const changeRequestTitle = changeRequest?.metadata?.title?.trim();
  const versionLabel = changeRequestTitle || formatVersionLabelWithTimestamp(version);
  const versionTimestamp = formatVersionTimestamp(version);
  const versionSubtitle = changeRequestTitle
    ? [subtitle || ownerName, versionTimestamp].filter(Boolean).join(" · ")
    : subtitle || ownerName;

  if (!versionID) {
    return null;
  }

  return (
    <div
      className={cn(
        "w-full rounded-md border px-2.5 py-2 text-left transition",
        isActive ? "border-sky-300 bg-sky-50" : "border-slate-200 bg-white",
        "hover:border-slate-300",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-900 truncate">{versionLabel}</p>
          <p className="mt-0.5 text-xs text-slate-600 truncate">{versionSubtitle}</p>
        </div>
        <div className="flex items-center gap-1.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="h-7 w-7"
                onClick={() => onUseVersion(versionID)}
                aria-label="Visualize version"
              >
                <Eye className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top">Preview this version</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="h-7 w-7"
                  disabled={!previousVersion}
                  onClick={() => {
                    if (!previousVersion) {
                      return;
                    }
                    onViewDiff(version, previousVersion, changeRequest);
                  }}
                  aria-label="View diff with previous version"
                >
                  <GitCompareArrows className="h-4 w-4" />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              {previousVersion ? "View node diff with previous version" : "No previous version to compare"}
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}
