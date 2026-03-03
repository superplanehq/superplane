import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CheckCircle2, ChevronLeft, ChevronRight, CircleDot, GitPullRequest, Plus, Rocket } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";

const PAGE_SIZE = 8;

type DiffLine = {
  kind: "add" | "remove";
  path: string;
  value: string;
};

type DiffGroup = {
  section: string;
  lines: DiffLine[];
};

interface CanvasVersioningViewProps {
  liveCanvasVersion?: CanvasesCanvasVersion;
  myVersions: CanvasesCanvasVersion[];
  activeCanvasVersionId?: string;
  changeRequests: CanvasesCanvasChangeRequest[];
  selectedChangeRequestId?: string;
  onUseVersion: (versionID: string) => void;
  onSelectChangeRequest: (changeRequestID: string) => void;
  onPublishChangeRequest: () => void;
  onCreateVersion: () => void;
  onCreateChangeRequest: () => void;
  createVersionDisabled: boolean;
  createChangeRequestDisabled: boolean;
  publishChangeRequestDisabled: boolean;
  createVersionPending: boolean;
  createChangeRequestPending: boolean;
  publishChangeRequestPending: boolean;
}

function formatVersionLabel(version?: CanvasesCanvasVersion): string {
  const revision = version?.metadata?.revision ?? "?";
  const dateRaw = version?.metadata?.updatedAt || version?.metadata?.createdAt || version?.metadata?.publishedAt;
  const date = dateRaw ? new Date(dateRaw) : null;
  const readable =
    date && !Number.isNaN(date.getTime())
      ? date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })
      : "";
  return readable ? `Revision ${revision} · ${readable}` : `Revision ${revision}`;
}

function normalizeChangeRequestStatus(status?: string | number): "open" | "published" | "conflicted" | "unknown" {
  if (typeof status === "number") {
    if (status === 1) return "open";
    if (status === 2) return "published";
    if (status === 3) return "conflicted";
    return "unknown";
  }

  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish")) return "published";
  if (value.includes("conflict")) return "conflicted";
  return "unknown";
}

function normalizeForCompare(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => normalizeForCompare(item));
  }

  if (!value || typeof value !== "object") {
    return value;
  }

  const entries = Object.entries(value as Record<string, unknown>).sort(([left], [right]) => left.localeCompare(right));
  const normalized: Record<string, unknown> = {};
  entries.forEach(([key, entryValue]) => {
    normalized[key] = normalizeForCompare(entryValue);
  });
  return normalized;
}

function serializeDiffValue(value: unknown): string {
  if (value === undefined) {
    return "undefined";
  }
  if (typeof value === "string") {
    return JSON.stringify(value);
  }
  return JSON.stringify(normalizeForCompare(value));
}

function flattenValue(value: unknown, basePath: string): Record<string, string> {
  if (value === undefined) {
    return {};
  }

  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return { [basePath]: serializeDiffValue(value) };
  }

  const objectValue = value as Record<string, unknown>;
  const keys = Object.keys(objectValue).sort((left, right) => left.localeCompare(right));
  if (keys.length === 0) {
    return { [basePath]: "{}" };
  }

  const flattened: Record<string, string> = {};
  keys.forEach((key) => {
    const nestedPath = `${basePath}.${key}`;
    const nestedValue = objectValue[key];
    if (nestedValue && typeof nestedValue === "object" && !Array.isArray(nestedValue)) {
      Object.assign(flattened, flattenValue(nestedValue, nestedPath));
      return;
    }
    flattened[nestedPath] = serializeDiffValue(nestedValue);
  });
  return flattened;
}

function buildNodeSnapshot(node?: Record<string, unknown>): Record<string, unknown> {
  if (!node) {
    return {};
  }

  return {
    name: node.name,
    type: node.type,
    position: node.position,
    isCollapsed: node.isCollapsed,
    integrationId: node.integrationId,
    ref: node.ref,
    configuration: node.configuration,
  };
}

function buildNodeDiffLines(oldNode?: Record<string, unknown>, newNode?: Record<string, unknown>): DiffLine[] {
  const oldSnapshot = buildNodeSnapshot(oldNode);
  const newSnapshot = buildNodeSnapshot(newNode);

  const oldFlat = Object.entries(oldSnapshot).reduce<Record<string, string>>((acc, [key, value]) => {
    Object.assign(acc, flattenValue(value, key));
    return acc;
  }, {});
  const newFlat = Object.entries(newSnapshot).reduce<Record<string, string>>((acc, [key, value]) => {
    Object.assign(acc, flattenValue(value, key));
    return acc;
  }, {});

  const allPaths = Array.from(new Set([...Object.keys(oldFlat), ...Object.keys(newFlat)])).sort((left, right) =>
    left.localeCompare(right),
  );

  const lines: DiffLine[] = [];
  allPaths.forEach((path) => {
    const oldValue = oldFlat[path];
    const newValue = newFlat[path];
    if (oldValue === newValue) {
      return;
    }

    if (oldValue !== undefined) {
      lines.push({ kind: "remove", path, value: oldValue });
    }
    if (newValue !== undefined) {
      lines.push({ kind: "add", path, value: newValue });
    }
  });

  return lines;
}

function getDiffSection(path: string): string {
  const firstSegment = path.split(".")[0] || "other";
  if (firstSegment === "configuration") {
    return "Configuration";
  }
  if (firstSegment === "position") {
    return "Position";
  }
  if (firstSegment === "ref") {
    return "Reference";
  }
  if (firstSegment === "name") {
    return "Name";
  }
  if (firstSegment === "type") {
    return "Type";
  }
  if (firstSegment === "isCollapsed") {
    return "Display";
  }
  if (firstSegment === "integrationId") {
    return "Integration";
  }
  return "Other";
}

function buildNodeDiffGroups(lines: DiffLine[]): DiffGroup[] {
  if (lines.length === 0) {
    return [];
  }

  const groupsMap = new Map<string, DiffLine[]>();
  lines.forEach((line) => {
    const section = getDiffSection(line.path);
    const current = groupsMap.get(section) || [];
    current.push(line);
    groupsMap.set(section, current);
  });

  const orderedSections = ["Configuration", "Position", "Reference", "Name", "Type", "Display", "Integration", "Other"];
  return orderedSections
    .filter((section) => groupsMap.has(section))
    .map((section) => ({
      section,
      lines: groupsMap.get(section) || [],
    }));
}

export function CanvasVersioningView({
  liveCanvasVersion,
  myVersions,
  activeCanvasVersionId,
  changeRequests,
  selectedChangeRequestId,
  onUseVersion,
  onSelectChangeRequest,
  onPublishChangeRequest,
  onCreateVersion,
  onCreateChangeRequest,
  createVersionDisabled,
  createChangeRequestDisabled,
  publishChangeRequestDisabled,
  createVersionPending,
  createChangeRequestPending,
  publishChangeRequestPending,
}: CanvasVersioningViewProps) {
  const [showMerged, setShowMerged] = useState(false);
  const [page, setPage] = useState(1);

  const visibleChangeRequests = useMemo(() => {
    return changeRequests.filter((changeRequest) => {
      const status = normalizeChangeRequestStatus(changeRequest.metadata?.status as string | number | undefined);
      if (showMerged) {
        return true;
      }

      return status !== "published";
    });
  }, [changeRequests, showMerged]);

  const pageCount = Math.max(1, Math.ceil(visibleChangeRequests.length / PAGE_SIZE));

  useEffect(() => {
    if (page > pageCount) {
      setPage(pageCount);
    }
  }, [page, pageCount]);

  const paginatedChangeRequests = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE;
    return visibleChangeRequests.slice(start, start + PAGE_SIZE);
  }, [visibleChangeRequests, page]);

  const selectedChangeRequest = useMemo(
    () => visibleChangeRequests.find((changeRequest) => changeRequest.metadata?.id === selectedChangeRequestId),
    [visibleChangeRequests, selectedChangeRequestId],
  );

  useEffect(() => {
    if (!selectedChangeRequestId) {
      return;
    }

    const hasVisibleSelected = visibleChangeRequests.some(
      (changeRequest) => changeRequest.metadata?.id === selectedChangeRequestId,
    );
    if (!hasVisibleSelected) {
      onSelectChangeRequest("");
    }
  }, [selectedChangeRequestId, visibleChangeRequests, onSelectChangeRequest]);
  const selectedChangeNodeDiffs = useMemo(() => {
    if (!selectedChangeRequest) {
      return [];
    }

    const changedNodeIDs = selectedChangeRequest.diff?.changedNodeIds || [];
    const conflictingNodeIDSet = new Set(selectedChangeRequest.diff?.conflictingNodeIds || []);
    const liveNodes = (liveCanvasVersion?.spec?.nodes || []) as Record<string, unknown>[];
    const crNodes = (selectedChangeRequest.version?.spec?.nodes || []) as Record<string, unknown>[];

    const liveNodeByID = new Map<string, Record<string, unknown>>();
    liveNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        liveNodeByID.set(nodeID, node);
      }
    });

    const crNodeByID = new Map<string, Record<string, unknown>>();
    crNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        crNodeByID.set(nodeID, node);
      }
    });

    return changedNodeIDs.map((nodeID) => {
      const oldNode = liveNodeByID.get(nodeID);
      const newNode = crNodeByID.get(nodeID);
      const lines = buildNodeDiffLines(oldNode, newNode);
      const groups = buildNodeDiffGroups(lines);
      const kind = !oldNode && newNode ? "added" : oldNode && !newNode ? "removed" : "updated";

      return {
        nodeID,
        kind,
        isConflicting: conflictingNodeIDSet.has(nodeID),
        lines,
        groups,
      };
    });
  }, [selectedChangeRequest, liveCanvasVersion?.spec?.nodes]);

  return (
    <div className="h-full overflow-auto bg-slate-50">
      <div className="mx-auto max-w-6xl p-5 md:p-7 space-y-5">
        <section className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm font-semibold text-slate-900">Versioning</p>
              <p className="text-xs text-slate-600">Manage live, your working versions, and change requests.</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button onClick={onCreateVersion} disabled={createVersionDisabled}>
                <Plus className="h-4 w-4" />
                {createVersionPending ? "Creating version..." : "Create version from live"}
              </Button>
              <Button variant="outline" onClick={onCreateChangeRequest} disabled={createChangeRequestDisabled}>
                <GitPullRequest className="h-4 w-4" />
                {createChangeRequestPending ? "Creating change request..." : "Create change request"}
              </Button>
              <Button onClick={onPublishChangeRequest} disabled={publishChangeRequestDisabled}>
                <Rocket className="h-4 w-4" />
                {publishChangeRequestPending ? "Publishing..." : "Publish selected CR"}
              </Button>
            </div>
          </div>
        </section>

        <section className="grid gap-5 lg:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Live Version</p>
            {liveCanvasVersion?.metadata?.id ? (
              <button
                type="button"
                onClick={() => onUseVersion(liveCanvasVersion.metadata?.id || "")}
                className="mt-3 w-full rounded-md border border-emerald-200 bg-emerald-50 p-3 text-left"
              >
                <p className="text-sm font-medium text-slate-900">{formatVersionLabel(liveCanvasVersion)}</p>
                <p className="mt-1 text-xs text-emerald-700 inline-flex items-center gap-1">
                  <CircleDot className="h-3.5 w-3.5" />
                  Published
                </p>
              </button>
            ) : (
              <p className="mt-3 text-xs text-slate-600">No live version available.</p>
            )}
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 lg:col-span-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              My Versions ({myVersions.length})
            </p>
            {myVersions.length === 0 ? (
              <p className="mt-3 text-xs text-slate-600">You don&apos;t have working versions yet.</p>
            ) : (
              <div className="mt-3 grid gap-2 md:grid-cols-2">
                {myVersions.map((version) => {
                  const versionID = version.metadata?.id || "";
                  const isActive = versionID !== "" && versionID === activeCanvasVersionId;

                  return (
                    <button
                      key={versionID}
                      type="button"
                      onClick={() => onUseVersion(versionID)}
                      className={cn(
                        "rounded-md border px-3 py-2 text-left",
                        isActive ? "border-sky-300 bg-sky-50" : "border-slate-200 bg-white hover:bg-slate-50",
                      )}
                    >
                      <p className="text-sm font-medium text-slate-900">{formatVersionLabel(version)}</p>
                      {isActive ? <p className="mt-1 text-[11px] text-sky-700">Active</p> : null}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </section>

        <section className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Change Requests</p>
            <Button variant="outline" size="sm" onClick={() => setShowMerged((current) => !current)}>
              {showMerged ? "Hide merged" : "Show merged"}
            </Button>
          </div>

          {visibleChangeRequests.length === 0 ? (
            <p className="mt-3 text-xs text-slate-600">No open change requests.</p>
          ) : (
            <div className="mt-3 space-y-2">
              {paginatedChangeRequests.map((changeRequest) => {
                const changeRequestID = changeRequest.metadata?.id || "";
                const status = normalizeChangeRequestStatus(
                  changeRequest.metadata?.status as string | number | undefined,
                );
                const changedCount = changeRequest.diff?.changedNodeIds?.length || 0;
                const conflictCount = changeRequest.diff?.conflictingNodeIds?.length || 0;

                return (
                  <button
                    key={changeRequestID}
                    type="button"
                    onClick={() => onSelectChangeRequest(changeRequestID)}
                    className={cn(
                      "w-full rounded-md border px-3 py-2 text-left",
                      selectedChangeRequestId === changeRequestID
                        ? "border-sky-300 bg-sky-50"
                        : "border-slate-200 bg-white hover:bg-slate-50",
                    )}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-sm font-medium text-slate-900 break-words">
                        CR · Revision {changeRequest.version?.metadata?.revision ?? "?"}
                      </p>
                      <span
                        className={cn(
                          "rounded px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                          status === "open" && "bg-blue-100 text-blue-800",
                          status === "published" && "bg-emerald-100 text-emerald-700",
                          status === "conflicted" && "bg-red-100 text-red-700",
                          status === "unknown" && "bg-slate-100 text-slate-700",
                        )}
                      >
                        {status}
                      </span>
                    </div>

                    <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-600">
                      <span>{changeRequest.metadata?.owner?.name || "Unknown owner"}</span>
                      <span>changed: {changedCount}</span>
                      <span className={conflictCount > 0 ? "text-red-700" : "text-emerald-700"}>
                        conflicts: {conflictCount}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}

          {visibleChangeRequests.length > PAGE_SIZE ? (
            <div className="mt-4 flex items-center justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((current) => Math.max(1, current - 1))}
                disabled={page <= 1}
              >
                <ChevronLeft className="h-4 w-4" />
                Prev
              </Button>
              <span className="text-xs text-slate-600">
                Page {page} / {pageCount}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((current) => Math.min(pageCount, current + 1))}
                disabled={page >= pageCount}
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          ) : null}

          {selectedChangeRequest ? (
            <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Selected CR Diff</p>
                {(selectedChangeRequest.diff?.conflictingNodeIds?.length || 0) === 0 ? (
                  <span className="inline-flex items-center gap-1 text-xs text-emerald-700">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    No conflicts
                  </span>
                ) : null}
              </div>
              <div className="mt-2">
                {selectedChangeNodeDiffs.length === 0 ? (
                  <p className="text-xs text-slate-600">No changes</p>
                ) : (
                  <Accordion type="multiple" className="rounded-md border border-slate-200 bg-white px-3">
                    {selectedChangeNodeDiffs.map((nodeDiff) => (
                      <AccordionItem key={nodeDiff.nodeID} value={nodeDiff.nodeID} className="border-slate-200">
                        <AccordionTrigger className="py-3 text-xs hover:no-underline">
                          <div className="min-w-0 flex-1 text-left">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-semibold text-slate-900 break-all">{nodeDiff.nodeID}</span>
                              <span
                                className={cn(
                                  "rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide",
                                  nodeDiff.kind === "added" && "bg-emerald-100 text-emerald-700",
                                  nodeDiff.kind === "removed" && "bg-red-100 text-red-700",
                                  nodeDiff.kind === "updated" && "bg-blue-100 text-blue-700",
                                )}
                              >
                                {nodeDiff.kind}
                              </span>
                              {nodeDiff.isConflicting ? (
                                <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-red-700">
                                  conflict
                                </span>
                              ) : null}
                            </div>
                          </div>
                        </AccordionTrigger>
                        <AccordionContent className="pb-3">
                          {nodeDiff.lines.length === 0 ? (
                            <p className="text-xs text-slate-600">No field-level changes.</p>
                          ) : (
                            <div className="space-y-3">
                              {nodeDiff.groups.map((group) => (
                                <div key={`${nodeDiff.nodeID}-${group.section}`}>
                                  <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
                                    {group.section}
                                  </p>
                                  <div className="space-y-1">
                                    {group.lines.map((line, index) => (
                                      <div
                                        key={`${nodeDiff.nodeID}-${line.kind}-${line.path}-${index}`}
                                        className={cn(
                                          "rounded px-2 py-1 font-mono text-[11px] break-all",
                                          line.kind === "add" && "bg-emerald-50 text-emerald-700",
                                          line.kind === "remove" && "bg-red-50 text-red-700",
                                        )}
                                      >
                                        <span className="mr-2 font-bold">{line.kind === "add" ? "+" : "-"}</span>
                                        <span>{line.path}:</span>
                                        <span className="ml-1">{line.value}</span>
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </AccordionContent>
                      </AccordionItem>
                    ))}
                  </Accordion>
                )}
              </div>
            </div>
          ) : null}
        </section>
      </div>
    </div>
  );
}
