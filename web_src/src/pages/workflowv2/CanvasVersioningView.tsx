import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import Editor from "@monaco-editor/react";
import type { Monaco } from "@monaco-editor/react";
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  CircleDot,
  GitPullRequest,
  Plus,
  RefreshCw,
  Rocket,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import * as yaml from "js-yaml";
import type { editor as MonacoEditor } from "monaco-editor";

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
  canvasName: string;
  canvasDescription?: string;
  changeRequests: CanvasesCanvasChangeRequest[];
  selectedChangeRequestId?: string;
  currentUserId?: string;
  onUseVersion: (versionID: string) => void;
  onSelectChangeRequest: (changeRequestID: string) => void;
  onPublishChangeRequest: () => void;
  onCloseChangeRequest: (changeRequestID: string) => void;
  onResolveChangeRequest: (data: {
    changeRequestId: string;
    nodes: Record<string, unknown>[];
    edges: Record<string, unknown>[];
  }) => Promise<void>;
  onCreateVersion: () => void;
  onCreateChangeRequest: () => void;
  createVersionDisabled: boolean;
  createChangeRequestDisabled: boolean;
  publishChangeRequestDisabled: boolean;
  resolveChangeRequestPending: boolean;
  createVersionPending: boolean;
  createChangeRequestPending: boolean;
  publishChangeRequestPending: boolean;
  closeChangeRequestPending: boolean;
}

type CanvasNodeLike = Record<string, unknown>;
type CanvasEdgeLike = Record<string, unknown>;
type ConflictBlockResolution = "current" | "incoming" | "both";

type ConflictMarkerBlock = {
  startLine: number;
  separatorLine: number;
  endLine: number;
  currentLabel: string;
  incomingLabel: string;
};

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

type ChangeRequestStatus = "open" | "published" | "conflicted" | "closed" | "unknown";
type ChangeRequestFilter = "open" | "closed" | "merged" | "conflicted" | "all";

function normalizeChangeRequestStatus(status?: string | number): ChangeRequestStatus {
  if (typeof status === "number") {
    if (status === 1) return "open";
    if (status === 2) return "published";
    if (status === 3) return "conflicted";
    if (status === 4) return "closed";
    return "unknown";
  }

  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish")) return "published";
  if (value.includes("conflict")) return "conflicted";
  if (value.includes("close")) return "closed";
  return "unknown";
}

function matchesChangeRequestFilter(status: ChangeRequestStatus, filter: ChangeRequestFilter): boolean {
  if (filter === "all") {
    return true;
  }
  if (filter === "open") {
    return status === "open" || status === "conflicted";
  }
  if (filter === "merged") {
    return status === "published";
  }
  if (filter === "closed") {
    return status === "closed";
  }
  if (filter === "conflicted") {
    return status === "conflicted";
  }
  return true;
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

function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

function prettyYAML(value: unknown): string {
  const normalized = normalizeForCompare(value === undefined ? null : value);
  return yaml.dump(normalized, {
    noRefs: true,
    lineWidth: 120,
    sortKeys: true,
  });
}

function parseNodeYAML(input: string, nodeID: string): { node: CanvasNodeLike | null; error?: string } {
  const trimmed = input.trim();
  if (!trimmed) {
    return { node: null };
  }

  if (trimmed.includes("<<<<<<<") || trimmed.includes("=======") || trimmed.includes(">>>>>>>")) {
    return { node: null, error: "Resolve conflict markers before applying YAML." };
  }

  try {
    const parsed = yaml.load(trimmed);
    if (parsed === null || parsed === undefined) {
      return { node: null };
    }

    if (typeof parsed !== "object" || Array.isArray(parsed)) {
      return { node: null, error: "Final Result must be a YAML object or null." };
    }

    return { node: { ...(parsed as CanvasNodeLike), id: nodeID } };
  } catch {
    return { node: null, error: "Invalid YAML format." };
  }
}

function buildConflictMarkerYAML(
  currentNode: CanvasNodeLike | undefined,
  incomingNode: CanvasNodeLike | undefined,
  currentLabel: string,
  incomingLabel: string,
): string {
  const currentObject = isPlainObject(currentNode) ? (normalizeForCompare(currentNode) as Record<string, unknown>) : {};
  const incomingObject = isPlainObject(incomingNode)
    ? (normalizeForCompare(incomingNode) as Record<string, unknown>)
    : {};

  const keys = Array.from(new Set([...Object.keys(currentObject), ...Object.keys(incomingObject)])).sort(
    (left, right) => left.localeCompare(right),
  );

  const lines: string[] = [];
  keys.forEach((key) => {
    const currentHasKey = Object.prototype.hasOwnProperty.call(currentObject, key);
    const incomingHasKey = Object.prototype.hasOwnProperty.call(incomingObject, key);

    if (!currentHasKey && !incomingHasKey) {
      return;
    }

    const currentValue = currentObject[key];
    const incomingValue = incomingObject[key];
    const valuesAreEqual =
      currentHasKey &&
      incomingHasKey &&
      JSON.stringify(normalizeForCompare(currentValue)) === JSON.stringify(normalizeForCompare(incomingValue));

    if (valuesAreEqual) {
      lines.push(...renderTopLevelFieldYAMLLines(key, currentValue, true));
      return;
    }

    lines.push(`<<<<<<< ${currentLabel}`);
    lines.push(...renderTopLevelFieldYAMLLines(key, currentValue, currentHasKey));
    lines.push("=======");
    lines.push(...renderTopLevelFieldYAMLLines(key, incomingValue, incomingHasKey));
    lines.push(`>>>>>>> ${incomingLabel}`);
  });

  return `${lines.join("\n").trimEnd()}\n`;
}

function findConflictMarkerBlocks(model: MonacoEditor.ITextModel): ConflictMarkerBlock[] {
  const blocks: ConflictMarkerBlock[] = [];
  const lineCount = model.getLineCount();
  let line = 1;

  while (line <= lineCount) {
    const lineContent = model.getLineContent(line);
    if (!lineContent.startsWith("<<<<<<< ")) {
      line += 1;
      continue;
    }

    const startLine = line;
    let separatorLine = -1;
    let endLine = -1;

    for (let searchLine = line + 1; searchLine <= lineCount; searchLine += 1) {
      const searchContent = model.getLineContent(searchLine);
      if (separatorLine < 0 && searchContent.startsWith("=======")) {
        separatorLine = searchLine;
        continue;
      }

      if (searchContent.startsWith(">>>>>>> ")) {
        endLine = searchLine;
        break;
      }
    }

    if (separatorLine > 0 && endLine > 0) {
      const currentLabel =
        model
          .getLineContent(startLine)
          .replace(/^<<<<<<<\s*/, "")
          .trim() || "Current";
      const incomingLabel =
        model
          .getLineContent(endLine)
          .replace(/^>>>>>>>+\s*/, "")
          .trim() || "Incoming";
      blocks.push({
        startLine,
        separatorLine,
        endLine,
        currentLabel,
        incomingLabel,
      });
      line = endLine + 1;
      continue;
    }

    line += 1;
  }

  return blocks;
}

function renderTopLevelFieldYAMLLines(key: string, value: unknown, hasKey: boolean): string[] {
  if (!hasKey) {
    return [`# ${key} is absent`];
  }

  const dumped = yaml.dump({ [key]: value }, { noRefs: true, lineWidth: 120, sortKeys: false }).trimEnd();
  if (!dumped) {
    return [];
  }
  return dumped.split("\n");
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function deepMergeObjects(current: unknown, incoming: unknown): unknown {
  if (!isPlainObject(current) || !isPlainObject(incoming)) {
    return incoming;
  }

  const merged: Record<string, unknown> = {};
  const keys = new Set([...Object.keys(current), ...Object.keys(incoming)]);
  keys.forEach((key) => {
    const currentValue = current[key];
    const incomingValue = incoming[key];

    if (incomingValue === undefined) {
      merged[key] = currentValue;
      return;
    }

    if (currentValue === undefined) {
      merged[key] = incomingValue;
      return;
    }

    merged[key] = deepMergeObjects(currentValue, incomingValue);
  });
  return merged;
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
  canvasName,
  canvasDescription,
  changeRequests,
  selectedChangeRequestId,
  currentUserId,
  onUseVersion,
  onSelectChangeRequest,
  onPublishChangeRequest,
  onCloseChangeRequest,
  onResolveChangeRequest,
  onCreateVersion,
  onCreateChangeRequest,
  createVersionDisabled,
  createChangeRequestDisabled,
  publishChangeRequestDisabled,
  resolveChangeRequestPending,
  createVersionPending,
  createChangeRequestPending,
  publishChangeRequestPending,
  closeChangeRequestPending,
}: CanvasVersioningViewProps) {
  const [changeRequestFilter, setChangeRequestFilter] = useState<ChangeRequestFilter>("open");
  const [onlyMine, setOnlyMine] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [page, setPage] = useState(1);
  const [resolvingChangeRequestID, setResolvingChangeRequestID] = useState("");

  const visibleChangeRequests = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    return changeRequests.filter((changeRequest) => {
      const status = normalizeChangeRequestStatus(changeRequest.metadata?.status as string | number | undefined);
      if (!matchesChangeRequestFilter(status, changeRequestFilter)) {
        return false;
      }

      if (onlyMine) {
        if (!currentUserId) {
          return false;
        }
        if ((changeRequest.metadata?.owner?.id || "").toLowerCase() !== currentUserId.toLowerCase()) {
          return false;
        }
      }

      if (!query) {
        return true;
      }

      const ownerName = (changeRequest.metadata?.owner?.name || "").toLowerCase();
      const ownerID = (changeRequest.metadata?.owner?.id || "").toLowerCase();
      const revision = String(changeRequest.version?.metadata?.revision || "").toLowerCase();
      const statusText = status.toLowerCase();
      return (
        ownerName.includes(query) || ownerID.includes(query) || revision.includes(query) || statusText.includes(query)
      );
    });
  }, [changeRequests, changeRequestFilter, onlyMine, currentUserId, searchQuery]);

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

  useEffect(() => {
    if (!resolvingChangeRequestID) {
      return;
    }

    const stillExists = changeRequests.some((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID);
    if (!stillExists) {
      setResolvingChangeRequestID("");
    }
  }, [changeRequests, resolvingChangeRequestID]);

  const resolvingChangeRequest = useMemo(
    () => changeRequests.find((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID),
    [changeRequests, resolvingChangeRequestID],
  );

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

  if (resolvingChangeRequest) {
    return (
      <CanvasChangeRequestConflictResolver
        liveCanvasVersion={liveCanvasVersion}
        changeRequest={resolvingChangeRequest}
        canvasName={canvasName}
        canvasDescription={canvasDescription}
        isSubmitting={resolveChangeRequestPending}
        onBack={() => setResolvingChangeRequestID("")}
        onSubmit={async (data) => {
          await onResolveChangeRequest(data);
          setResolvingChangeRequestID("");
        }}
      />
    );
  }

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
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Change Requests</p>
              <label className="inline-flex items-center gap-2 text-xs text-slate-700">
                <input
                  type="checkbox"
                  checked={onlyMine}
                  onChange={(event) => setOnlyMine(event.target.checked)}
                  className="h-3.5 w-3.5 rounded border-slate-300"
                />
                My CRs
              </label>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {(
                [
                  { key: "open", label: "Open" },
                  { key: "conflicted", label: "Conflicted" },
                  { key: "closed", label: "Closed" },
                  { key: "merged", label: "Merged" },
                  { key: "all", label: "All" },
                ] as { key: ChangeRequestFilter; label: string }[]
              ).map((filter) => (
                <Button
                  key={filter.key}
                  type="button"
                  size="sm"
                  variant={changeRequestFilter === filter.key ? "default" : "outline"}
                  onClick={() => setChangeRequestFilter(filter.key)}
                >
                  {filter.label}
                </Button>
              ))}
            </div>
            <input
              type="text"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Search by owner, revision, or status"
              className="h-9 w-full rounded-md border border-slate-300 px-3 text-sm text-slate-900 focus:border-sky-400 focus:outline-none"
            />
          </div>

          {visibleChangeRequests.length === 0 ? (
            <p className="mt-3 text-xs text-slate-600">No change requests found for this filter.</p>
          ) : (
            <div className="mt-3 space-y-2">
              {paginatedChangeRequests.map((changeRequest) => {
                const changeRequestID = changeRequest.metadata?.id || "";
                const status = normalizeChangeRequestStatus(
                  changeRequest.metadata?.status as string | number | undefined,
                );
                const changedCount = changeRequest.diff?.changedNodeIds?.length || 0;
                const conflictCount = changeRequest.diff?.conflictingNodeIds?.length || 0;
                const canResolve = status === "conflicted" && changeRequestID !== "";
                const canClose = (status === "open" || status === "conflicted") && changeRequestID !== "";

                return (
                  <div key={changeRequestID} className="flex items-start gap-2">
                    <button
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
                            status === "closed" && "bg-slate-200 text-slate-700",
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
                    <div className="flex shrink-0 flex-col gap-1">
                      {canResolve ? (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            onSelectChangeRequest(changeRequestID);
                            setResolvingChangeRequestID(changeRequestID);
                          }}
                        >
                          Resolve
                        </Button>
                      ) : null}
                      {canClose ? (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={closeChangeRequestPending}
                          onClick={() => onCloseChangeRequest(changeRequestID)}
                        >
                          {closeChangeRequestPending ? "Closing..." : "Close"}
                        </Button>
                      ) : null}
                    </div>
                  </div>
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
                <div className="flex items-center gap-2">
                  {(selectedChangeRequest.diff?.conflictingNodeIds?.length || 0) > 0 ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={
                        normalizeChangeRequestStatus(
                          selectedChangeRequest.metadata?.status as string | number | undefined,
                        ) !== "conflicted"
                      }
                      onClick={() => setResolvingChangeRequestID(selectedChangeRequest.metadata?.id || "")}
                    >
                      <RefreshCw className="h-3.5 w-3.5" />
                      Resolve conflicts
                    </Button>
                  ) : (
                    <span className="inline-flex items-center gap-1 text-xs text-emerald-700">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      No conflicts
                    </span>
                  )}
                  {(() => {
                    const selectedStatus = normalizeChangeRequestStatus(
                      selectedChangeRequest.metadata?.status as string | number | undefined,
                    );
                    const canCloseSelected =
                      (selectedStatus === "open" || selectedStatus === "conflicted") &&
                      !!selectedChangeRequest.metadata?.id;
                    if (!canCloseSelected) {
                      return null;
                    }

                    return (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={closeChangeRequestPending}
                        onClick={() => onCloseChangeRequest(selectedChangeRequest.metadata?.id || "")}
                      >
                        {closeChangeRequestPending ? "Closing..." : "Close CR"}
                      </Button>
                    );
                  })()}
                </div>
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

function upsertNode(nodes: CanvasNodeLike[], nodeID: string, node: CanvasNodeLike | null): CanvasNodeLike[] {
  const index = nodes.findIndex((item) => String(item.id || "") === nodeID);
  if (!node) {
    if (index < 0) {
      return nodes;
    }
    const next = [...nodes];
    next.splice(index, 1);
    return next;
  }

  if (index < 0) {
    return [...nodes, node];
  }

  const next = [...nodes];
  next[index] = node;
  return next;
}

function getNodeID(node: CanvasNodeLike | undefined): string {
  return String(node?.id || "");
}

function buildNodeMap(nodes: CanvasNodeLike[]): Map<string, CanvasNodeLike> {
  const result = new Map<string, CanvasNodeLike>();
  nodes.forEach((node) => {
    const id = getNodeID(node);
    if (id) {
      result.set(id, node);
    }
  });
  return result;
}

function pruneEdgesByNodes(edges: CanvasEdgeLike[], nodes: CanvasNodeLike[]): CanvasEdgeLike[] {
  const nodeIDSet = new Set(nodes.map((node) => getNodeID(node)).filter(Boolean));
  return edges.filter((edge) => {
    const sourceID = String(edge.sourceId || "");
    const targetID = String(edge.targetId || "");
    if (!sourceID || !targetID) {
      return false;
    }
    return nodeIDSet.has(sourceID) && nodeIDSet.has(targetID);
  });
}

function localResolutionLabel(
  currentNode: CanvasNodeLike | undefined,
  incomingNode: CanvasNodeLike | undefined,
  finalNode: CanvasNodeLike | undefined,
): string {
  if (!finalNode) {
    return "excluded";
  }

  if (JSON.stringify(normalizeForCompare(finalNode)) === JSON.stringify(normalizeForCompare(currentNode))) {
    return "current";
  }

  if (JSON.stringify(normalizeForCompare(finalNode)) === JSON.stringify(normalizeForCompare(incomingNode))) {
    return "incoming";
  }

  return "custom";
}

function CanvasChangeRequestConflictResolver({
  liveCanvasVersion,
  changeRequest,
  canvasName,
  canvasDescription,
  isSubmitting,
  onBack,
  onSubmit,
}: {
  liveCanvasVersion?: CanvasesCanvasVersion;
  changeRequest: CanvasesCanvasChangeRequest;
  canvasName: string;
  canvasDescription?: string;
  isSubmitting: boolean;
  onBack: () => void;
  onSubmit: (data: { changeRequestId: string; nodes: CanvasNodeLike[]; edges: CanvasEdgeLike[] }) => Promise<void>;
}) {
  const conflictingNodeIDs = useMemo(
    () => (changeRequest.diff?.conflictingNodeIds || []).filter(Boolean),
    [changeRequest.diff?.conflictingNodeIds],
  );

  const [selectedNodeID, setSelectedNodeID] = useState(conflictingNodeIDs[0] || "");
  const [finalNodes, setFinalNodes] = useState<CanvasNodeLike[]>(() =>
    cloneJSON((changeRequest.version?.spec?.nodes || []) as CanvasNodeLike[]),
  );
  const [finalEdges, setFinalEdges] = useState<CanvasEdgeLike[]>(() =>
    cloneJSON((changeRequest.version?.spec?.edges || []) as CanvasEdgeLike[]),
  );
  const [finalDraftYAML, setFinalDraftYAML] = useState("");
  const [finalDraftError, setFinalDraftError] = useState("");
  const [resolvedNodeIDs, setResolvedNodeIDs] = useState<Set<string>>(() => new Set());
  const resolverEditorRef = useRef<MonacoEditor.IStandaloneCodeEditor | null>(null);
  const resolverMonacoRef = useRef<Monaco | null>(null);
  const resolverDecorationsRef = useRef<string[]>([]);
  const resolverViewZonesRef = useRef<string[]>([]);

  const liveNodes = useMemo(
    () => cloneJSON((liveCanvasVersion?.spec?.nodes || []) as CanvasNodeLike[]),
    [liveCanvasVersion?.spec?.nodes],
  );
  const incomingNodes = useMemo(
    () => cloneJSON((changeRequest.version?.spec?.nodes || []) as CanvasNodeLike[]),
    [changeRequest.version?.spec?.nodes],
  );

  const liveNodeByID = useMemo(() => buildNodeMap(liveNodes), [liveNodes]);
  const incomingNodeByID = useMemo(() => buildNodeMap(incomingNodes), [incomingNodes]);
  const finalNodeByID = useMemo(() => buildNodeMap(finalNodes), [finalNodes]);

  useEffect(() => {
    setFinalNodes(cloneJSON((changeRequest.version?.spec?.nodes || []) as CanvasNodeLike[]));
    setFinalEdges(cloneJSON((changeRequest.version?.spec?.edges || []) as CanvasEdgeLike[]));
    setResolvedNodeIDs(new Set());
    const nextSelected = (changeRequest.diff?.conflictingNodeIds || [])[0] || "";
    setSelectedNodeID(nextSelected);
    setFinalDraftYAML("");
    setFinalDraftError("");
  }, [changeRequest.metadata?.id, changeRequest.version?.spec?.nodes, changeRequest.version?.spec?.edges]);

  const currentNode = selectedNodeID ? liveNodeByID.get(selectedNodeID) : undefined;
  const incomingNode = selectedNodeID ? incomingNodeByID.get(selectedNodeID) : undefined;
  const finalNode = selectedNodeID ? finalNodeByID.get(selectedNodeID) : undefined;
  const liveRevision = liveCanvasVersion?.metadata?.revision ?? "?";
  const incomingRevision = changeRequest.version?.metadata?.revision ?? "?";
  const incomingOwnerName = changeRequest.metadata?.owner?.name || "Unknown owner";
  const currentConflictLabel = `Current Live r${liveRevision}`;
  const incomingConflictLabel = `Incoming CR r${incomingRevision} (${incomingOwnerName})`;

  const applyConflictResolutionForBlock = useCallback(
    (block: ConflictMarkerBlock, resolution: ConflictBlockResolution) => {
      const editor = resolverEditorRef.current;
      const monaco = resolverMonacoRef.current;
      if (!editor || !monaco) {
        return;
      }

      const model = editor.getModel();
      if (!model) {
        return;
      }

      if (block.startLine < 1 || block.endLine > model.getLineCount() || block.startLine >= block.endLine) {
        return;
      }

      const currentStartLine = block.startLine + 1;
      const currentEndLine = block.separatorLine - 1;
      const incomingStartLine = block.separatorLine + 1;
      const incomingEndLine = block.endLine - 1;

      const currentLines =
        currentStartLine <= currentEndLine ? model.getLinesContent().slice(currentStartLine - 1, currentEndLine) : [];
      const incomingLines =
        incomingStartLine <= incomingEndLine
          ? model.getLinesContent().slice(incomingStartLine - 1, incomingEndLine)
          : [];

      let replacementLines: string[] = [];
      if (resolution === "current") {
        replacementLines = currentLines;
      } else if (resolution === "incoming") {
        replacementLines = incomingLines;
      } else if (currentLines.length === 0) {
        replacementLines = incomingLines;
      } else if (incomingLines.length === 0) {
        replacementLines = currentLines;
      } else {
        replacementLines = [...currentLines, ...incomingLines];
      }

      editor.pushUndoStop();
      editor.executeEdits("canvas-conflict-inline-resolver", [
        {
          range: new monaco.Range(block.startLine, 1, block.endLine, model.getLineMaxColumn(block.endLine)),
          text: replacementLines.join("\n"),
          forceMoveMarkers: true,
        },
      ]);
      editor.pushUndoStop();
      setFinalDraftYAML(model.getValue());
      setFinalDraftError("");
      editor.focus();
    },
    [],
  );

  const applyConflictDecorations = useCallback(() => {
    const editor = resolverEditorRef.current;
    const monaco = resolverMonacoRef.current;
    if (!editor || !monaco) {
      return;
    }

    const model = editor.getModel();
    if (!model) {
      return;
    }

    const blocks = findConflictMarkerBlocks(model);
    const nextDecorations: MonacoEditor.IModelDeltaDecoration[] = [];
    blocks.forEach((block) => {
      nextDecorations.push(
        {
          range: new monaco.Range(block.startLine, 1, block.separatorLine, 1),
          options: {
            isWholeLine: true,
            className: "canvas-conflict-current-line",
          },
        },
        {
          range: new monaco.Range(block.separatorLine, 1, block.endLine, 1),
          options: {
            isWholeLine: true,
            className: "canvas-conflict-incoming-line",
          },
        },
        {
          range: new monaco.Range(block.startLine, 1, block.startLine, 1),
          options: {
            isWholeLine: true,
            className: "canvas-conflict-marker-current",
          },
        },
        {
          range: new monaco.Range(block.separatorLine, 1, block.separatorLine, 1),
          options: {
            isWholeLine: true,
            className: "canvas-conflict-marker-separator",
          },
        },
        {
          range: new monaco.Range(block.endLine, 1, block.endLine, 1),
          options: {
            isWholeLine: true,
            className: "canvas-conflict-marker-incoming",
          },
        },
      );
    });

    resolverDecorationsRef.current = editor.deltaDecorations(resolverDecorationsRef.current, nextDecorations);
    editor.changeViewZones((accessor) => {
      resolverViewZonesRef.current.forEach((zoneID) => accessor.removeZone(zoneID));
      resolverViewZonesRef.current = [];

      blocks.forEach((block) => {
        const zoneNode = document.createElement("div");
        zoneNode.className = "canvas-conflict-inline-actions";

        const buttonContainer = document.createElement("div");
        buttonContainer.className = "canvas-conflict-inline-actions__buttons";

        const createActionButton = (title: string, resolution: ConflictBlockResolution) => {
          const button = document.createElement("button");
          button.type = "button";
          button.className = "canvas-conflict-inline-action-button";
          button.textContent = title;
          button.onpointerdown = (event) => {
            event.preventDefault();
            event.stopPropagation();
            applyConflictResolutionForBlock(block, resolution);
          };
          button.onclick = (event) => {
            event.preventDefault();
            event.stopPropagation();
          };
          return button;
        };

        buttonContainer.appendChild(createActionButton("Accept Current", "current"));
        buttonContainer.appendChild(createActionButton("Accept Incoming", "incoming"));
        buttonContainer.appendChild(createActionButton("Accept Both", "both"));
        zoneNode.appendChild(buttonContainer);

        const zoneID = accessor.addZone({
          afterLineNumber: block.startLine - 1,
          heightInPx: 26,
          domNode: zoneNode,
          suppressMouseDown: true,
        });
        resolverViewZonesRef.current.push(zoneID);
      });
    });
  }, [applyConflictResolutionForBlock]);

  useEffect(() => {
    return () => {
      const editor = resolverEditorRef.current;
      if (!editor) {
        return;
      }

      editor.deltaDecorations(resolverDecorationsRef.current, []);
      editor.changeViewZones((accessor) => {
        resolverViewZonesRef.current.forEach((zoneID) => accessor.removeZone(zoneID));
        resolverViewZonesRef.current = [];
      });
    };
  }, []);

  useEffect(() => {
    if (!selectedNodeID) {
      setFinalDraftYAML("");
      setFinalDraftError("");
      return;
    }

    if (resolvedNodeIDs.has(selectedNodeID)) {
      const resolvedNode = finalNodeByID.get(selectedNodeID);
      setFinalDraftYAML(prettyYAML(resolvedNode || null));
      setFinalDraftError("");
      return;
    }

    setFinalDraftYAML(buildConflictMarkerYAML(currentNode, incomingNode, currentConflictLabel, incomingConflictLabel));
    setFinalDraftError("");
  }, [
    selectedNodeID,
    currentNode,
    incomingNode,
    currentConflictLabel,
    incomingConflictLabel,
    resolvedNodeIDs,
    finalNodeByID,
  ]);

  useEffect(() => {
    applyConflictDecorations();
  }, [finalDraftYAML, applyConflictDecorations]);

  const onApplyFinalYAML = () => {
    if (!selectedNodeID) {
      return;
    }

    const { node, error } = parseNodeYAML(finalDraftYAML, selectedNodeID);
    if (error) {
      setFinalDraftError(error);
      return;
    }

    if (!node) {
      setFinalNodes((current) => upsertNode(current, selectedNodeID, null));
      setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
      setFinalDraftError("");
      return;
    }

    setFinalNodes((current) => upsertNode(current, selectedNodeID, cloneJSON(node)));
    setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
    setFinalDraftError("");
  };

  const onUseCurrentNode = () => {
    if (!selectedNodeID) {
      return;
    }

    const node = liveNodeByID.get(selectedNodeID);
    setFinalNodes((current) => upsertNode(current, selectedNodeID, node ? cloneJSON(node) : null));
    setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
    setFinalDraftYAML(prettyYAML(node || null));
    setFinalDraftError("");
  };

  const onUseIncomingNode = () => {
    if (!selectedNodeID) {
      return;
    }

    const node = incomingNodeByID.get(selectedNodeID);
    setFinalNodes((current) => upsertNode(current, selectedNodeID, node ? cloneJSON(node) : null));
    setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
    setFinalDraftYAML(prettyYAML(node || null));
    setFinalDraftError("");
  };

  const onUseBothNode = () => {
    if (!selectedNodeID) {
      return;
    }

    if (!currentNode && !incomingNode) {
      setFinalNodes((current) => upsertNode(current, selectedNodeID, null));
      setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
      setFinalDraftYAML("null\n");
      setFinalDraftError("");
      return;
    }

    const merged = cloneJSON(
      deepMergeObjects(currentNode || {}, incomingNode || {}) as CanvasNodeLike,
    ) as CanvasNodeLike;
    merged.id = selectedNodeID;
    setFinalNodes((current) => upsertNode(current, selectedNodeID, merged));
    setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
    setFinalDraftYAML(prettyYAML(merged));
    setFinalDraftError("");
  };

  const onToggleIncludeNode = () => {
    if (!selectedNodeID) {
      return;
    }

    if (finalNode) {
      setFinalNodes((current) => upsertNode(current, selectedNodeID, null));
      setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
      setFinalDraftYAML("null\n");
      setFinalDraftError("");
      return;
    }

    const fallbackNode = incomingNode || currentNode;
    setFinalNodes((current) => upsertNode(current, selectedNodeID, fallbackNode ? cloneJSON(fallbackNode) : null));
    setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
    setFinalDraftYAML(prettyYAML(fallbackNode || null));
    setFinalDraftError("");
  };

  const handleResolve = async () => {
    const prunedEdges = pruneEdgesByNodes(finalEdges, finalNodes);
    const changeRequestID = changeRequest.metadata?.id || "";
    if (!changeRequestID) {
      return;
    }

    await onSubmit({
      changeRequestId: changeRequestID,
      nodes: finalNodes,
      edges: prunedEdges,
    });
  };

  const yamlEditorOptions = useMemo(
    () => ({
      minimap: { enabled: false },
      fontSize: 12,
      lineNumbers: "on" as const,
      wordWrap: "on" as const,
      folding: true,
      autoIndent: "advanced" as const,
      formatOnPaste: true,
      formatOnType: true,
      tabSize: 2,
      insertSpaces: true,
      scrollBeyondLastLine: false,
      renderWhitespace: "selection" as const,
      smoothScrolling: true,
      cursorBlinking: "smooth" as const,
      bracketPairColorization: {
        enabled: true,
      },
      automaticLayout: true,
    }),
    [],
  );

  return (
    <div className="h-full overflow-auto bg-slate-50">
      <div className="mx-auto max-w-7xl p-5 md:p-7 space-y-5">
        <section className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="space-y-1">
              <Button variant="ghost" size="sm" className="px-1" onClick={onBack}>
                <ArrowLeft className="h-4 w-4" />
                Back to Versioning
              </Button>
              <p className="text-sm font-semibold text-slate-900">Resolve Change Request Conflicts</p>
              <p className="text-xs text-slate-600">
                Review Current vs Incoming, then edit Final Result and save the resolved version.
              </p>
            </div>
            <Button onClick={handleResolve} disabled={isSubmitting || !canvasName}>
              <Check className="h-4 w-4" />
              {isSubmitting ? "Resolving..." : "Save resolved result"}
            </Button>
          </div>
          <p className="mt-2 text-[11px] text-slate-600">
            Canvas: {canvasName}
            {canvasDescription ? ` · ${canvasDescription}` : ""}
          </p>
        </section>

        <section className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
          <div className="rounded-xl border border-slate-200 bg-white p-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Conflicting Nodes ({conflictingNodeIDs.length})
            </p>
            {conflictingNodeIDs.length === 0 ? (
              <p className="mt-2 text-xs text-emerald-700">No conflicts left in this CR.</p>
            ) : (
              <div className="mt-2 space-y-1.5">
                {conflictingNodeIDs.map((nodeID) => {
                  const resolution = localResolutionLabel(
                    liveNodeByID.get(nodeID),
                    incomingNodeByID.get(nodeID),
                    finalNodeByID.get(nodeID),
                  );

                  return (
                    <button
                      key={nodeID}
                      type="button"
                      onClick={() => setSelectedNodeID(nodeID)}
                      className={cn(
                        "w-full rounded-md border px-2 py-2 text-left",
                        selectedNodeID === nodeID
                          ? "border-sky-300 bg-sky-50"
                          : "border-slate-200 bg-white hover:bg-slate-50",
                      )}
                    >
                      <p className="text-xs font-medium text-slate-900 break-all">{nodeID}</p>
                      <p className="mt-1 text-[11px] text-slate-600">final: {resolution}</p>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-3">
            {!selectedNodeID ? (
              <p className="text-sm text-slate-600">Select a conflicting node to resolve.</p>
            ) : (
              <div className="space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-slate-500">Node</span>
                  <span className="rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-800 break-all">
                    {selectedNodeID}
                  </span>
                  {(changeRequest.diff?.conflictingNodeIds || []).includes(selectedNodeID) ? (
                    <span className="inline-flex items-center gap-1 rounded bg-red-100 px-2 py-0.5 text-xs text-red-700">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      conflict
                    </span>
                  ) : null}
                </div>

                <div className="rounded-md border border-slate-200 bg-slate-50 p-2">
                  <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                    <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                      YAML Conflict Resolver (VS Code Style)
                    </p>
                    <div className="flex flex-wrap gap-2">
                      <Button variant="outline" size="sm" onClick={onUseCurrentNode}>
                        Accept Current
                      </Button>
                      <Button variant="outline" size="sm" onClick={onUseIncomingNode}>
                        Accept Incoming
                      </Button>
                      <Button variant="outline" size="sm" onClick={onUseBothNode}>
                        Accept Both
                      </Button>
                      <Button variant="outline" size="sm" onClick={onToggleIncludeNode}>
                        {finalNode ? "Exclude node" : "Include node"}
                      </Button>
                      <Button variant="default" size="sm" onClick={onApplyFinalYAML}>
                        Apply YAML
                      </Button>
                    </div>
                  </div>

                  <p className="mb-2 text-[11px] text-slate-600">
                    Resolve markers directly in this editor using the inline conflict actions for each block (or edit
                    manually), then apply YAML.
                  </p>
                  <div className="h-[520px] overflow-hidden rounded border border-slate-200 bg-white">
                    <Editor
                      defaultLanguage="yaml"
                      value={finalDraftYAML}
                      onChange={(value) => setFinalDraftYAML(value || "")}
                      onMount={(editor, monaco) => {
                        resolverEditorRef.current = editor;
                        resolverMonacoRef.current = monaco;
                        editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Alt | monaco.KeyCode.KeyZ, () => {
                          editor.trigger("keyboard", "undo", null);
                        });
                        applyConflictDecorations();
                      }}
                      theme="vs"
                      options={{
                        ...yamlEditorOptions,
                        glyphMargin: true,
                      }}
                    />
                  </div>
                  {finalDraftError ? <p className="mt-2 text-xs text-red-700">{finalDraftError}</p> : null}
                </div>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
