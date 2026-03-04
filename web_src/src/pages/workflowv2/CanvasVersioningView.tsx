import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useInfiniteCanvasChangeRequests } from "@/hooks/useCanvasData";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import Editor from "@monaco-editor/react";
import ReactMarkdown from "react-markdown";
import type { Monaco } from "@monaco-editor/react";
import { AlertTriangle, ArrowLeft, Check, CheckCircle2, GitPullRequest, RefreshCw, Rocket } from "lucide-react";
import { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import * as yaml from "js-yaml";
import type { editor as MonacoEditor } from "monaco-editor";

const CHANGE_REQUEST_PAGE_SIZE = 10;

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
  organizationId: string;
  canvasId: string;
  liveCanvasVersion?: CanvasesCanvasVersion;
  liveVersions: CanvasesCanvasVersion[];
  liveVersionsTotalCount?: number;
  hasMoreLiveVersions?: boolean;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsPending?: boolean;
  myVersions: CanvasesCanvasVersion[];
  activeCanvasVersionId?: string;
  canvasName: string;
  canvasDescription?: string;
  changeRequests: CanvasesCanvasChangeRequest[];
  selectedChangeRequestId?: string;
  onSelectChangeRequest: (changeRequestID: string) => void;
  onPublishChangeRequest: () => void;
  onCloseChangeRequest: (changeRequestID: string) => void;
  onResolveChangeRequest: (data: {
    changeRequestId: string;
    nodes: Record<string, unknown>[];
    edges: Record<string, unknown>[];
  }) => Promise<void>;
  createChangeRequestMode: boolean;
  onCreateChangeRequestModeChange: (enabled: boolean) => void;
  onSubmitCreateChangeRequest: (data: { title: string; description: string }) => Promise<void>;
  sandboxModeEnabled: boolean;
  sandboxModeTooltip?: string;
  publishChangeRequestDisabled: boolean;
  publishChangeRequestDisabledTooltip?: string;
  resolveChangeRequestPending: boolean;
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

type ChangeRequestStatus = "open" | "conflicted" | "merged" | "rejected" | "unknown";
type ChangeRequestFilter = "open" | "rejected" | "merged" | "conflicted" | "all";

function normalizeChangeRequestStatus(status?: string | number): ChangeRequestStatus {
  if (typeof status === "number") {
    if (status === 1) return "open";
    if (status === 2) return "merged";
    if (status === 3) return "conflicted";
    if (status === 4) return "rejected";
    return "unknown";
  }

  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish") || value.includes("merge")) return "merged";
  if (value.includes("close") || value.includes("reject")) return "rejected";
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

function withTooltip(disabled: boolean, message: string | undefined, element: ReactNode): ReactNode {
  if (!disabled || !message) {
    return element;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="inline-flex">{element}</div>
      </TooltipTrigger>
      <TooltipContent>{message}</TooltipContent>
    </Tooltip>
  );
}

const MARKDOWN_PREVIEW_WRAPPER_CLASS =
  "[&_a]:underline [&_a]:underline-offset-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 [&_blockquote]:text-slate-700 [&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_hr]:my-3 [&_hr]:border-slate-200 [&_li]:mb-1 [&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_p]:mb-2 [&_p]:last:mb-0 [&_pre]:mb-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 [&_strong]:font-semibold [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc";

const markdownPreviewComponents = {
  h1: ({ children }: { children?: ReactNode }) => <h1 className="mb-2 text-base font-semibold">{children}</h1>,
  h2: ({ children }: { children?: ReactNode }) => <h2 className="mb-2 text-base font-semibold">{children}</h2>,
  h3: ({ children }: { children?: ReactNode }) => <h3 className="mb-1 text-sm font-semibold">{children}</h3>,
  h4: ({ children }: { children?: ReactNode }) => <h4 className="mb-1 text-sm font-medium">{children}</h4>,
  p: ({ children }: { children?: ReactNode }) => <p className="mb-2 last:mb-0">{children}</p>,
  ul: ({ children }: { children?: ReactNode }) => <ul className="mb-2 ml-5 list-disc">{children}</ul>,
  ol: ({ children }: { children?: ReactNode }) => <ol className="mb-2 ml-5 list-decimal">{children}</ol>,
  li: ({ children }: { children?: ReactNode }) => <li className="mb-1">{children}</li>,
  a: ({ children, href }: { children?: ReactNode; href?: string }) => (
    <a className="underline underline-offset-2" target="_blank" rel="noopener noreferrer" href={href}>
      {children}
    </a>
  ),
  code: ({ children }: { children?: ReactNode }) => (
    <code className="rounded bg-slate-100 px-1 py-0.5">{children}</code>
  ),
  pre: ({ children }: { children?: ReactNode }) => (
    <pre className="mb-2 overflow-auto rounded bg-slate-100 p-2">{children}</pre>
  ),
  strong: ({ children }: { children?: ReactNode }) => <strong className="font-semibold">{children}</strong>,
  em: ({ children }: { children?: ReactNode }) => <em className="italic">{children}</em>,
};

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
  organizationId,
  canvasId,
  liveCanvasVersion,
  liveVersions,
  liveVersionsTotalCount,
  hasMoreLiveVersions,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsPending,
  myVersions,
  activeCanvasVersionId,
  canvasName,
  canvasDescription,
  changeRequests,
  selectedChangeRequestId,
  onSelectChangeRequest,
  onPublishChangeRequest,
  onCloseChangeRequest,
  onResolveChangeRequest,
  createChangeRequestMode,
  onCreateChangeRequestModeChange,
  onSubmitCreateChangeRequest,
  sandboxModeEnabled,
  sandboxModeTooltip,
  publishChangeRequestDisabled,
  publishChangeRequestDisabledTooltip,
  resolveChangeRequestPending,
  createChangeRequestPending,
  publishChangeRequestPending,
  closeChangeRequestPending,
}: CanvasVersioningViewProps) {
  const [changeRequestFilter, setChangeRequestFilter] = useState<ChangeRequestFilter>("open");
  const [onlyMine, setOnlyMine] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [resolvingChangeRequestID, setResolvingChangeRequestID] = useState("");
  const [selectedLiveHistoryVersionId, setSelectedLiveHistoryVersionId] = useState(activeCanvasVersionId || "");
  const [createChangeRequestTitle, setCreateChangeRequestTitle] = useState("");
  const [createChangeRequestDescription, setCreateChangeRequestDescription] = useState("");
  const [createChangeRequestDescriptionMode, setCreateChangeRequestDescriptionMode] = useState<"write" | "preview">(
    "write",
  );

  const createChangeRequestVersion = useMemo(() => {
    if (activeCanvasVersionId) {
      const activeVersion = myVersions.find((version) => version.metadata?.id === activeCanvasVersionId);
      if (activeVersion && !activeVersion.metadata?.isPublished) {
        return activeVersion;
      }
    }

    return myVersions[0];
  }, [activeCanvasVersionId, myVersions]);

  useEffect(() => {
    if (!createChangeRequestMode) {
      return;
    }

    const revision = createChangeRequestVersion?.metadata?.revision ?? "?";
    const defaultTitle = `Update ${canvasName || "Canvas"} - Revision ${revision}`;
    setCreateChangeRequestTitle(defaultTitle);
    setCreateChangeRequestDescription("");
    setCreateChangeRequestDescriptionMode("write");
  }, [createChangeRequestMode, createChangeRequestVersion?.metadata?.revision, canvasName]);

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      setSearchQuery(searchInput.trim());
    }, 300);
    return () => window.clearTimeout(timeout);
  }, [searchInput]);

  const canvasChangeRequestsQuery = useInfiniteCanvasChangeRequests(organizationId, canvasId, {
    enabled: true,
    limit: CHANGE_REQUEST_PAGE_SIZE,
    statusFilter: changeRequestFilter,
    onlyMine,
    searchQuery,
  });

  const visibleChangeRequests = useMemo(
    () => (canvasChangeRequestsQuery.data?.pages || []).flatMap((page) => page?.changeRequests || []),
    [canvasChangeRequestsQuery.data?.pages],
  );
  const isLoadingChangeRequests =
    (canvasChangeRequestsQuery.isPending || canvasChangeRequestsQuery.isFetching) && visibleChangeRequests.length === 0;

  const selectedChangeRequest = useMemo(
    () =>
      visibleChangeRequests.find((changeRequest) => changeRequest.metadata?.id === selectedChangeRequestId) ||
      changeRequests.find((changeRequest) => changeRequest.metadata?.id === selectedChangeRequestId),
    [visibleChangeRequests, changeRequests, selectedChangeRequestId],
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
    () => visibleChangeRequests.find((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID),
    [visibleChangeRequests, resolvingChangeRequestID],
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

  const createChangeRequestNodeDiffs = useMemo(() => {
    if (!createChangeRequestVersion) {
      return [];
    }

    const liveNodes = (liveCanvasVersion?.spec?.nodes || []) as Record<string, unknown>[];
    const draftNodes = (createChangeRequestVersion.spec?.nodes || []) as Record<string, unknown>[];

    const liveNodeByID = new Map<string, Record<string, unknown>>();
    liveNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        liveNodeByID.set(nodeID, node);
      }
    });

    const draftNodeByID = new Map<string, Record<string, unknown>>();
    draftNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        draftNodeByID.set(nodeID, node);
      }
    });

    const allNodeIDs = Array.from(new Set([...liveNodeByID.keys(), ...draftNodeByID.keys()])).sort((left, right) =>
      left.localeCompare(right),
    );

    return allNodeIDs
      .map((nodeID) => {
        const oldNode = liveNodeByID.get(nodeID);
        const newNode = draftNodeByID.get(nodeID);
        const lines = buildNodeDiffLines(oldNode, newNode);
        if (lines.length === 0) {
          return null;
        }

        const kind = !oldNode && newNode ? "added" : oldNode && !newNode ? "removed" : "updated";
        return {
          nodeID,
          kind,
          lines,
          groups: buildNodeDiffGroups(lines),
        };
      })
      .filter(
        (
          item,
        ): item is { nodeID: string; kind: "added" | "removed" | "updated"; lines: DiffLine[]; groups: DiffGroup[] } =>
          Boolean(item),
      );
  }, [createChangeRequestVersion, liveCanvasVersion?.spec?.nodes]);

  useEffect(() => {
    const nextVersionID = activeCanvasVersionId || liveCanvasVersion?.metadata?.id || "";
    if (!nextVersionID) {
      return;
    }
    if (selectedLiveHistoryVersionId) {
      const stillVisible = liveVersions.some((version) => version.metadata?.id === selectedLiveHistoryVersionId);
      if (stillVisible) {
        return;
      }
    }
    setSelectedLiveHistoryVersionId(nextVersionID);
  }, [activeCanvasVersionId, liveCanvasVersion?.metadata?.id, liveVersions, selectedLiveHistoryVersionId]);

  const selectedLiveHistoryVersion = useMemo(
    () => liveVersions.find((version) => version.metadata?.id === selectedLiveHistoryVersionId),
    [liveVersions, selectedLiveHistoryVersionId],
  );

  const selectedLiveHistoryBaseVersion = useMemo(() => {
    if (!selectedLiveHistoryVersion) {
      return undefined;
    }

    const basedOnVersionID = selectedLiveHistoryVersion.metadata?.basedOnVersionId || "";
    if (basedOnVersionID) {
      return liveVersions.find((version) => version.metadata?.id === basedOnVersionID);
    }

    const index = liveVersions.findIndex((version) => version.metadata?.id === selectedLiveHistoryVersion.metadata?.id);
    if (index >= 0 && index + 1 < liveVersions.length) {
      return liveVersions[index + 1];
    }

    return undefined;
  }, [liveVersions, selectedLiveHistoryVersion]);

  const selectedLiveHistoryNodeDiffs = useMemo(() => {
    if (!selectedLiveHistoryVersion) {
      return [];
    }

    const baseNodes = (selectedLiveHistoryBaseVersion?.spec?.nodes || []) as Record<string, unknown>[];
    const targetNodes = (selectedLiveHistoryVersion.spec?.nodes || []) as Record<string, unknown>[];

    const baseNodeByID = new Map<string, Record<string, unknown>>();
    baseNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        baseNodeByID.set(nodeID, node);
      }
    });

    const targetNodeByID = new Map<string, Record<string, unknown>>();
    targetNodes.forEach((node) => {
      const nodeID = (node.id as string) || "";
      if (nodeID) {
        targetNodeByID.set(nodeID, node);
      }
    });

    const allNodeIDs = Array.from(new Set([...baseNodeByID.keys(), ...targetNodeByID.keys()])).sort((left, right) =>
      left.localeCompare(right),
    );

    return allNodeIDs
      .map((nodeID) => {
        const oldNode = baseNodeByID.get(nodeID);
        const newNode = targetNodeByID.get(nodeID);
        const lines = buildNodeDiffLines(oldNode, newNode);
        if (lines.length === 0) {
          return null;
        }

        const kind = !oldNode && newNode ? "added" : oldNode && !newNode ? "removed" : "updated";
        return {
          nodeID,
          kind,
          lines,
          groups: buildNodeDiffGroups(lines),
        };
      })
      .filter(
        (
          item,
        ): item is { nodeID: string; kind: "added" | "removed" | "updated"; lines: DiffLine[]; groups: DiffGroup[] } =>
          Boolean(item),
      );
  }, [selectedLiveHistoryBaseVersion?.spec?.nodes, selectedLiveHistoryVersion]);

  const handleSubmitCreateChangeRequest = useCallback(async () => {
    await onSubmitCreateChangeRequest({
      title: createChangeRequestTitle.trim(),
      description: createChangeRequestDescription,
    });
  }, [onSubmitCreateChangeRequest, createChangeRequestTitle, createChangeRequestDescription]);

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

  if (createChangeRequestMode) {
    const liveRevision = liveCanvasVersion?.metadata?.revision ?? "?";
    const addedNodes = createChangeRequestNodeDiffs.filter((nodeDiff) => nodeDiff.kind === "added").length;
    const removedNodes = createChangeRequestNodeDiffs.filter((nodeDiff) => nodeDiff.kind === "removed").length;
    const updatedNodes = createChangeRequestNodeDiffs.filter((nodeDiff) => nodeDiff.kind === "updated").length;

    return (
      <div className="h-full overflow-auto bg-slate-100">
        <div className="mx-auto max-w-6xl space-y-4 p-5 md:p-7">
          <section className="rounded-md border border-slate-300 bg-white">
            <div className="border-b border-slate-200 px-4 py-3">
              <p className="text-lg font-semibold text-slate-900">Open Change Request</p>
              <p className="mt-1 text-sm text-slate-600">Review your draft changes and open a request for review.</p>
            </div>
            <div className="flex flex-wrap items-center gap-2 px-4 py-3 text-xs">
              <span className="rounded border border-slate-300 bg-slate-50 px-2 py-1 text-slate-700">
                base: Live Revision {liveRevision}
              </span>
            </div>
          </section>

          {!createChangeRequestVersion ? (
            <section className="rounded-md border border-slate-300 bg-white p-4">
              <p className="text-sm text-slate-600">
                Enable edit mode and save your changes before creating a change request.
              </p>
            </section>
          ) : (
            <>
              <section className="rounded-md border border-slate-300 bg-white">
                <div className="border-b border-slate-200 px-4 py-3">
                  <input
                    value={createChangeRequestTitle}
                    onChange={(event) => setCreateChangeRequestTitle(event.target.value)}
                    placeholder="Title"
                    className="h-10 w-full rounded-md border border-slate-300 px-3 text-base text-slate-900 focus:border-sky-400 focus:outline-none"
                  />
                </div>

                <div className="px-4 py-3">
                  <Tabs
                    value={createChangeRequestDescriptionMode}
                    onValueChange={(value) => setCreateChangeRequestDescriptionMode(value as "write" | "preview")}
                    className="w-full"
                  >
                    <TabsList className="h-9 gap-1 rounded-md border border-slate-200 bg-slate-50 p-0.5">
                      <TabsTrigger value="write" className="px-3 text-xs">
                        Write
                      </TabsTrigger>
                      <TabsTrigger value="preview" className="px-3 text-xs">
                        Preview
                      </TabsTrigger>
                    </TabsList>
                    <TabsContent value="write" className="mt-2">
                      <textarea
                        value={createChangeRequestDescription}
                        onChange={(event) => setCreateChangeRequestDescription(event.target.value)}
                        rows={12}
                        placeholder="Describe what changed and why."
                        className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-sky-400 focus:outline-none"
                      />
                    </TabsContent>
                    <TabsContent value="preview" className="mt-2">
                      <div
                        className={cn(
                          "min-h-[280px] rounded-md border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900",
                          MARKDOWN_PREVIEW_WRAPPER_CLASS,
                        )}
                      >
                        {createChangeRequestDescription.trim() ? (
                          <ReactMarkdown components={markdownPreviewComponents}>
                            {createChangeRequestDescription}
                          </ReactMarkdown>
                        ) : (
                          <p className="text-xs text-slate-500">Nothing to preview.</p>
                        )}
                      </div>
                    </TabsContent>
                  </Tabs>
                  <div className="mt-3 flex flex-wrap items-center justify-end gap-2">
                    <Button variant="outline" onClick={() => onCreateChangeRequestModeChange(false)}>
                      Cancel
                    </Button>
                    {withTooltip(
                      !createChangeRequestVersion || createChangeRequestPending || sandboxModeEnabled,
                      sandboxModeEnabled ? sandboxModeTooltip : undefined,
                      <Button
                        onClick={handleSubmitCreateChangeRequest}
                        disabled={!createChangeRequestVersion || createChangeRequestPending || sandboxModeEnabled}
                      >
                        <GitPullRequest className="h-4 w-4" />
                        {createChangeRequestPending ? "Creating change request..." : "Create change request"}
                      </Button>,
                    )}
                  </div>
                </div>
              </section>

              <section className="rounded-md border border-slate-300 bg-white p-4">
                <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                  <p className="text-sm font-semibold text-slate-900">Nodes Changed</p>
                  <div className="flex flex-wrap items-center gap-2 text-[11px]">
                    <span className="rounded border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-emerald-700">
                      +{addedNodes} added
                    </span>
                    <span className="rounded border border-blue-200 bg-blue-50 px-2 py-0.5 text-blue-700">
                      ~{updatedNodes} updated
                    </span>
                    <span className="rounded border border-red-200 bg-red-50 px-2 py-0.5 text-red-700">
                      -{removedNodes} removed
                    </span>
                  </div>
                </div>
                {createChangeRequestNodeDiffs.length === 0 ? (
                  <p className="mt-2 text-xs text-slate-600">No changes between your edit version and live.</p>
                ) : (
                  <Accordion type="multiple" className="mt-2 rounded-md border border-slate-200 bg-white px-3">
                    {createChangeRequestNodeDiffs.map((nodeDiff) => (
                      <AccordionItem key={nodeDiff.nodeID} value={nodeDiff.nodeID} className="border-slate-200">
                        <AccordionTrigger className="py-3 text-xs hover:no-underline">
                          <div className="min-w-0 flex-1 text-left">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="break-all font-semibold text-slate-900">{nodeDiff.nodeID}</span>
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
                            </div>
                          </div>
                        </AccordionTrigger>
                        <AccordionContent className="pb-3">
                          <div className="space-y-3">
                            {nodeDiff.groups.map((group) => (
                              <div
                                key={`${nodeDiff.nodeID}-${group.section}`}
                                className="rounded-md border border-slate-200 bg-slate-50"
                              >
                                <div className="border-b border-slate-200 px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-slate-600">
                                  {group.section}
                                </div>
                                <div className="px-3 py-2 font-mono text-xs">
                                  {group.lines.map((line, index) => (
                                    <p
                                      key={`${nodeDiff.nodeID}-${group.section}-${line.kind}-${line.path}-${index}`}
                                      className={cn(
                                        "break-all",
                                        line.kind === "add" ? "text-emerald-700" : "text-red-700",
                                      )}
                                    >
                                      {line.kind === "add" ? "+" : "-"} {line.path}: {line.value}
                                    </p>
                                  ))}
                                </div>
                              </div>
                            ))}
                          </div>
                        </AccordionContent>
                      </AccordionItem>
                    ))}
                  </Accordion>
                )}
              </section>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto bg-slate-50">
      <div className="mx-auto max-w-6xl p-5 md:p-7 space-y-5">
        <section className="rounded-xl border border-slate-200 bg-white p-4">
          <div>
            <p className="text-sm font-semibold text-slate-900">Versioning</p>
            <p className="text-xs text-slate-600">Manage live history and change requests.</p>
          </div>
          {sandboxModeEnabled ? (
            <p className="mt-3 text-xs text-amber-700">{sandboxModeTooltip || "Sandbox mode is enabled."}</p>
          ) : null}
        </section>

        <section className="rounded-xl border border-slate-200 bg-white p-4">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Live History ({liveVersionsTotalCount ?? liveVersions.length})
          </p>
          {liveVersions.length === 0 ? (
            <p className="mt-3 text-xs text-slate-600">No published history yet.</p>
          ) : (
            <>
              <div className="mt-3 flex max-h-[320px] flex-col gap-2 overflow-y-auto pr-1">
                {liveVersions.map((version) => {
                  const versionID = version.metadata?.id || "";
                  const isActive = versionID !== "" && versionID === activeCanvasVersionId;
                  const isSelected = versionID !== "" && versionID === selectedLiveHistoryVersionId;

                  return (
                    <button
                      key={versionID}
                      type="button"
                      onClick={() => {
                        setSelectedLiveHistoryVersionId(versionID);
                      }}
                      className={cn(
                        "w-full rounded-md border px-3 py-2 text-left",
                        isSelected
                          ? "border-sky-300 bg-sky-50"
                          : isActive
                            ? "border-sky-200 bg-sky-50/50"
                            : "border-slate-200 bg-white hover:bg-slate-50",
                      )}
                    >
                      <p className="text-sm font-medium text-slate-900">{formatVersionLabel(version)}</p>
                    </button>
                  );
                })}
              </div>
              {hasMoreLiveVersions ? (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-3"
                  onClick={onLoadMoreLiveVersions}
                  disabled={!onLoadMoreLiveVersions || loadMoreLiveVersionsPending}
                >
                  {loadMoreLiveVersionsPending ? "Loading..." : "Load older revisions"}
                </Button>
              ) : null}
              {selectedLiveHistoryVersion ? (
                <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-3">
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                    Selected Revision Diff · {formatVersionLabel(selectedLiveHistoryVersion)}
                  </p>
                  <p className="mt-1 text-xs text-slate-600">
                    Showing changes applied against{" "}
                    {selectedLiveHistoryBaseVersion
                      ? formatVersionLabel(selectedLiveHistoryBaseVersion)
                      : "an empty baseline (first live revision)"}
                    .
                  </p>
                  {selectedLiveHistoryNodeDiffs.length === 0 ? (
                    <p className="mt-2 text-xs text-slate-600">No node-level changes detected for this revision.</p>
                  ) : (
                    <Accordion type="multiple" className="mt-2 rounded-md border border-slate-200 bg-white px-3">
                      {selectedLiveHistoryNodeDiffs.map((nodeDiff) => (
                        <AccordionItem
                          key={`history-${nodeDiff.nodeID}`}
                          value={`history-${nodeDiff.nodeID}`}
                          className="border-slate-200"
                        >
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
                              </div>
                            </div>
                          </AccordionTrigger>
                          <AccordionContent className="pb-3">
                            <div className="space-y-3">
                              {nodeDiff.groups.map((group) => (
                                <div
                                  key={`history-${nodeDiff.nodeID}-${group.section}`}
                                  className="rounded-md border border-slate-200 bg-slate-50"
                                >
                                  <div className="border-b border-slate-200 px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-slate-600">
                                    {group.section}
                                  </div>
                                  <div className="px-3 py-2 font-mono text-xs">
                                    {group.lines.map((line, index) => (
                                      <p
                                        key={`history-${nodeDiff.nodeID}-${group.section}-${line.kind}-${line.path}-${index}`}
                                        className={cn(
                                          "break-all",
                                          line.kind === "add" ? "text-emerald-700" : "text-red-700",
                                        )}
                                      >
                                        {line.kind === "add" ? "+" : "-"} {line.path}: {line.value}
                                      </p>
                                    ))}
                                  </div>
                                </div>
                              ))}
                            </div>
                          </AccordionContent>
                        </AccordionItem>
                      ))}
                    </Accordion>
                  )}
                </div>
              ) : null}
            </>
          )}
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
                  { key: "rejected", label: "Rejected" },
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
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="Search by title, owner, revision, or status"
              className="h-9 w-full rounded-md border border-slate-300 px-3 text-sm text-slate-900 focus:border-sky-400 focus:outline-none"
            />
          </div>

          {isLoadingChangeRequests ? (
            <p className="mt-3 text-xs text-slate-600">Loading change requests...</p>
          ) : visibleChangeRequests.length === 0 ? (
            <p className="mt-3 text-xs text-slate-600">No change requests found for this filter.</p>
          ) : (
            <div className="mt-3 space-y-2">
              {visibleChangeRequests.map((changeRequest) => {
                const changeRequestID = changeRequest.metadata?.id || "";
                const status = normalizeChangeRequestStatus(
                  changeRequest.metadata?.status as string | number | undefined,
                );
                const changedCount = changeRequest.diff?.changedNodeIds?.length || 0;
                const conflictCount = changeRequest.diff?.conflictingNodeIds?.length || 0;
                const canResolve =
                  (status === "open" || status === "conflicted") && conflictCount > 0 && changeRequestID !== "";
                const canClose = (status === "open" || status === "conflicted") && changeRequestID !== "";
                const versioningActionDisabled = sandboxModeEnabled;

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
                          {changeRequest.metadata?.title?.trim()
                            ? changeRequest.metadata.title
                            : `CR · Revision ${changeRequest.version?.metadata?.revision ?? "?"}`}
                        </p>
                        <span
                          className={cn(
                            "rounded px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                            status === "open" && "bg-blue-100 text-blue-800",
                            status === "conflicted" && "bg-red-100 text-red-700",
                            status === "merged" && "bg-emerald-100 text-emerald-700",
                            status === "rejected" && "bg-slate-200 text-slate-700",
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
                      {canResolve
                        ? withTooltip(
                            versioningActionDisabled,
                            sandboxModeTooltip,
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={versioningActionDisabled}
                              onClick={() => {
                                onSelectChangeRequest(changeRequestID);
                                setResolvingChangeRequestID(changeRequestID);
                              }}
                            >
                              Resolve
                            </Button>,
                          )
                        : null}
                      {canClose
                        ? withTooltip(
                            versioningActionDisabled,
                            sandboxModeTooltip,
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={versioningActionDisabled || closeChangeRequestPending}
                              onClick={() => onCloseChangeRequest(changeRequestID)}
                            >
                              {closeChangeRequestPending ? "Rejecting..." : "Reject"}
                            </Button>,
                          )
                        : null}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {canvasChangeRequestsQuery.hasNextPage ? (
            <div className="mt-4 flex items-center justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={() => canvasChangeRequestsQuery.fetchNextPage()}
                disabled={canvasChangeRequestsQuery.isFetchingNextPage}
              >
                {canvasChangeRequestsQuery.isFetchingNextPage ? "Loading..." : "Load more change requests"}
              </Button>
            </div>
          ) : null}

          {selectedChangeRequest ? (
            <div className="mt-4 rounded-md border border-slate-200 bg-slate-50 p-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Selected CR Diff</p>
                <div className="flex items-center gap-2">
                  {(selectedChangeRequest.diff?.conflictingNodeIds?.length || 0) > 0 ? (
                    withTooltip(
                      sandboxModeEnabled,
                      sandboxModeTooltip,
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={
                          sandboxModeEnabled ||
                          !["open", "conflicted"].includes(
                            normalizeChangeRequestStatus(
                              selectedChangeRequest.metadata?.status as string | number | undefined,
                            ),
                          )
                        }
                        onClick={() => setResolvingChangeRequestID(selectedChangeRequest.metadata?.id || "")}
                      >
                        <RefreshCw className="h-3.5 w-3.5" />
                        Resolve conflicts
                      </Button>,
                    )
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
                    const canPublishSelected = selectedStatus === "open";
                    if (!canPublishSelected) {
                      return null;
                    }

                    return withTooltip(
                      publishChangeRequestDisabled,
                      publishChangeRequestDisabledTooltip,
                      <Button
                        type="button"
                        size="sm"
                        disabled={publishChangeRequestDisabled}
                        onClick={onPublishChangeRequest}
                      >
                        <Rocket className="h-3.5 w-3.5" />
                        {publishChangeRequestPending ? "Publishing..." : "Publish"}
                      </Button>,
                    );
                  })()}
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

                    return withTooltip(
                      sandboxModeEnabled,
                      sandboxModeTooltip,
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={sandboxModeEnabled || closeChangeRequestPending}
                        onClick={() => onCloseChangeRequest(selectedChangeRequest.metadata?.id || "")}
                      >
                        {closeChangeRequestPending ? "Rejecting..." : "Reject CR"}
                      </Button>,
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
  const unresolvedNodeCount = useMemo(
    () => conflictingNodeIDs.filter((nodeID) => !resolvedNodeIDs.has(nodeID)).length,
    [conflictingNodeIDs, resolvedNodeIDs],
  );
  const allConflictsMarkedResolved = unresolvedNodeCount === 0;
  const selectedNodeHasConflictMarkers = useMemo(
    () => /^(<<<<<<< |=======|>>>>>>> )/m.test(finalDraftYAML),
    [finalDraftYAML],
  );
  const canMarkSelectedNodeAsResolved = Boolean(selectedNodeID) && !selectedNodeHasConflictMarkers;
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

  const onMarkNodeAsResolved = () => {
    if (!canMarkSelectedNodeAsResolved) {
      return;
    }

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
    if (!allConflictsMarkedResolved) {
      return;
    }

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
              {!allConflictsMarkedResolved ? (
                <p className="text-xs text-amber-700">
                  {unresolvedNodeCount} conflicting node{unresolvedNodeCount === 1 ? "" : "s"} still need to be marked
                  as resolved.
                </p>
              ) : null}
            </div>
            <Button onClick={handleResolve} disabled={isSubmitting || !canvasName || !allConflictsMarkedResolved}>
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
                  const isResolved = resolvedNodeIDs.has(nodeID);

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
                      <div className="mt-1 flex items-center gap-1.5 text-[11px]">
                        <span className="text-slate-600">final: {resolution}</span>
                        <span
                          className={cn(
                            "rounded px-1.5 py-0.5 uppercase tracking-wide",
                            isResolved ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700",
                          )}
                        >
                          {isResolved ? "resolved" : "pending"}
                        </span>
                      </div>
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
                  {(changeRequest.diff?.conflictingNodeIds || []).includes(selectedNodeID) &&
                  !resolvedNodeIDs.has(selectedNodeID) ? (
                    <span className="inline-flex items-center gap-1 rounded bg-red-100 px-2 py-0.5 text-xs text-red-700">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      conflict
                    </span>
                  ) : null}
                  {selectedNodeID && resolvedNodeIDs.has(selectedNodeID) ? (
                    <span className="inline-flex items-center gap-1 rounded bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      resolved
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
                      <Button
                        variant="default"
                        size="sm"
                        onClick={onMarkNodeAsResolved}
                        disabled={!canMarkSelectedNodeAsResolved}
                      >
                        Mark as resolved
                      </Button>
                    </div>
                  </div>

                  <p className="mb-2 text-[11px] text-slate-600">
                    Resolve markers directly in this editor using the inline conflict actions for each block (or edit
                    manually), then mark this node as resolved.
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
