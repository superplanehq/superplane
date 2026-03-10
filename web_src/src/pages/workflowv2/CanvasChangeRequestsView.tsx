import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { cn } from "@/lib/utils";
import Editor from "@monaco-editor/react";
import type { Monaco } from "@monaco-editor/react";
import { AlertTriangle, ArrowLeft, Check, CheckCircle2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as yaml from "js-yaml";
import type { editor as MonacoEditor } from "monaco-editor";

type ChangeRequestFilter = "all" | "open" | "rejected" | "published";
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

type DiffLine = {
  kind: "add" | "remove";
  path: string;
  value: string;
};

type DiffGroup = {
  section: string;
  lines: DiffLine[];
};

interface CanvasChangeRequestsViewProps {
  changeRequests: CanvasesCanvasChangeRequest[];
  selectedChangeRequestId?: string;
  canUpdateCanvas: boolean;
  actionPending: boolean;
  resolvePending: boolean;
  liveCanvasVersion?: CanvasesCanvasVersion;
  canvasName: string;
  canvasDescription?: string;
  onSelectChangeRequest: (changeRequestId: string) => void;
  onApprove: (changeRequestId: string) => Promise<void>;
  onReject: (changeRequestId: string) => Promise<void>;
  onReopen: (changeRequestId: string) => Promise<void>;
  onResolve: (data: {
    changeRequestId: string;
    nodes: Record<string, unknown>[];
    edges: Record<string, unknown>[];
  }) => Promise<void>;
}

function normalizeStatus(status?: string): "open" | "published" | "rejected" | "unknown" {
  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish")) return "published";
  if (value.includes("reject")) return "rejected";
  return "unknown";
}

function formatStatusLabel(status: ReturnType<typeof normalizeStatus>): string {
  if (status === "open") return "Open";
  if (status === "published") return "Published";
  if (status === "rejected") return "Rejected";
  return "Unknown";
}

function statusBadgeVariant(
  status: ReturnType<typeof normalizeStatus>,
): "default" | "secondary" | "destructive" | "outline" {
  if (status === "published") return "default";
  if (status === "rejected") return "destructive";
  return "outline";
}

function isChangeRequestConflicted(changeRequest?: CanvasesCanvasChangeRequest): boolean {
  if (!changeRequest) {
    return false;
  }

  if (typeof changeRequest.metadata?.isConflicted === "boolean") {
    return changeRequest.metadata.isConflicted;
  }

  return (changeRequest.diff?.conflictingNodeIds || []).length > 0;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
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

function formatTimestamp(value?: string): string {
  if (!value) {
    return "unknown time";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "unknown time";
  }
  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
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
  const liveVersionLabel = liveCanvasVersion?.metadata?.id?.slice(0, 8) || "live";
  const incomingVersionLabel = changeRequest.version?.metadata?.id?.slice(0, 8) || "draft";
  const incomingOwnerName = changeRequest.metadata?.owner?.name || "Unknown owner";
  const currentConflictLabel = `Current Live (${liveVersionLabel})`;
  const incomingConflictLabel = `Incoming CR (${incomingVersionLabel}) (${incomingOwnerName})`;

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
      <div className="mx-auto max-w-7xl space-y-5 p-5 md:p-7">
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
                      <p className="break-all text-xs font-medium text-slate-900">{nodeID}</p>
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
                  <span className="break-all rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-800">
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

export function CanvasChangeRequestsView({
  changeRequests,
  selectedChangeRequestId,
  canUpdateCanvas,
  actionPending,
  resolvePending,
  liveCanvasVersion,
  canvasName,
  canvasDescription,
  onSelectChangeRequest,
  onApprove,
  onReject,
  onReopen,
  onResolve,
}: CanvasChangeRequestsViewProps) {
  const [filter, setFilter] = useState<ChangeRequestFilter>("open");
  const [resolvingChangeRequestID, setResolvingChangeRequestID] = useState("");
  const [selectedTab, setSelectedTab] = useState<"conversation" | "nodes">("conversation");
  const [showDetailView, setShowDetailView] = useState(Boolean(selectedChangeRequestId));

  const filteredRequests = useMemo(() => {
    if (filter === "all") {
      return changeRequests;
    }
    return changeRequests.filter((item) => normalizeStatus(item.metadata?.status) === filter);
  }, [changeRequests, filter]);

  const selectedChangeRequest = useMemo(() => {
    const selected = changeRequests.find((item) => item.metadata?.id === selectedChangeRequestId);
    if (selected) {
      return selected;
    }
    return filteredRequests[0];
  }, [changeRequests, filteredRequests, selectedChangeRequestId]);

  useEffect(() => {
    setSelectedTab("conversation");
  }, [selectedChangeRequest?.metadata?.id]);

  useEffect(() => {
    if (!selectedChangeRequestId) {
      setShowDetailView(false);
    }
  }, [selectedChangeRequestId]);

  const resolvingChangeRequest = useMemo(
    () => changeRequests.find((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID),
    [changeRequests, resolvingChangeRequestID],
  );

  useEffect(() => {
    if (!resolvingChangeRequestID) {
      return;
    }

    const stillExists = changeRequests.some((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID);
    if (!stillExists) {
      setResolvingChangeRequestID("");
    }
  }, [changeRequests, resolvingChangeRequestID]);

  const selectedStatus = normalizeStatus(selectedChangeRequest?.metadata?.status);
  const selectedChangeRequestIdSafe = selectedChangeRequest?.metadata?.id || "";
  const conflictingNodeIds = selectedChangeRequest?.diff?.conflictingNodeIds || [];
  const changedNodeIds = selectedChangeRequest?.diff?.changedNodeIds || [];
  const selectedHasConflicts = isChangeRequestConflicted(selectedChangeRequest);

  const canApprove = canUpdateCanvas && selectedStatus === "open" && !selectedHasConflicts;
  const canReject = canUpdateCanvas && selectedStatus === "open";
  const canReopen = canUpdateCanvas && selectedStatus === "rejected";
  const canResolveConflicts =
    canUpdateCanvas &&
    selectedStatus === "open" &&
    selectedHasConflicts &&
    !!selectedChangeRequest?.version?.spec?.nodes &&
    !!selectedChangeRequest?.version?.spec?.edges;

  const selectedChangeNodeDiffs = useMemo(() => {
    if (!selectedChangeRequest) {
      return [];
    }

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

    const allNodeIDs = Array.from(new Set([...liveNodeByID.keys(), ...crNodeByID.keys()])).sort((left, right) =>
      left.localeCompare(right),
    );

    return allNodeIDs
      .map((nodeID) => {
        const oldNode = liveNodeByID.get(nodeID);
        const newNode = crNodeByID.get(nodeID);
        const lines = buildNodeDiffLines(oldNode, newNode);
        if (lines.length === 0) {
          return null;
        }

        const groups = buildNodeDiffGroups(lines);
        const kind = !oldNode && newNode ? "added" : oldNode && !newNode ? "removed" : "updated";
        const nodeNameValue = newNode?.name ?? oldNode?.name;
        const nodeName =
          typeof nodeNameValue === "string" && nodeNameValue.trim() ? nodeNameValue.trim() : "Unnamed node";

        return {
          nodeID,
          nodeName,
          kind,
          isConflicting: conflictingNodeIDSet.has(nodeID),
          lines,
          groups,
        };
      })
      .filter(
        (
          item,
        ): item is {
          nodeID: string;
          nodeName: string;
          kind: "added" | "removed" | "updated";
          isConflicting: boolean;
          lines: DiffLine[];
          groups: DiffGroup[];
        } => Boolean(item),
      );
  }, [selectedChangeRequest, liveCanvasVersion?.spec?.nodes]);

  const addedNodes = selectedChangeNodeDiffs.filter((diff) => diff.kind === "added").length;
  const removedNodes = selectedChangeNodeDiffs.filter((diff) => diff.kind === "removed").length;
  const updatedNodes = selectedChangeNodeDiffs.filter((diff) => diff.kind === "updated").length;

  if (resolvingChangeRequest) {
    return (
      <CanvasChangeRequestConflictResolver
        liveCanvasVersion={liveCanvasVersion}
        changeRequest={resolvingChangeRequest}
        canvasName={canvasName}
        canvasDescription={canvasDescription}
        isSubmitting={resolvePending}
        onBack={() => setResolvingChangeRequestID("")}
        onSubmit={async (data) => {
          await onResolve(data);
          setResolvingChangeRequestID("");
        }}
      />
    );
  }

  if (!showDetailView) {
    return (
      <div className="h-full overflow-auto bg-slate-50">
        <div className="mx-auto max-w-6xl space-y-4 p-5 md:p-7">
          <section className="rounded-xl border border-slate-200 bg-white">
            <div className="border-b border-slate-200 px-4 py-3">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <p className="text-base font-semibold text-slate-900">Change Requests</p>
                  <p className="text-xs text-slate-600">Select a request to open it in a dedicated PR view.</p>
                </div>
                <Badge variant="outline">{changeRequests.length}</Badge>
              </div>
            </div>
            <div className="space-y-3 p-4">
              <Tabs value={filter} onValueChange={(value) => setFilter(value as ChangeRequestFilter)}>
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="open">Open</TabsTrigger>
                  <TabsTrigger value="rejected">Rejected</TabsTrigger>
                  <TabsTrigger value="published">Published</TabsTrigger>
                  <TabsTrigger value="all">All</TabsTrigger>
                </TabsList>
              </Tabs>

              <div className="max-h-[500px] overflow-auto rounded-md border border-slate-200 bg-white">
                {filteredRequests.length === 0 ? (
                  <p className="p-3 text-sm text-slate-600">No change requests in this filter.</p>
                ) : (
                  filteredRequests.map((item) => {
                    const itemId = item.metadata?.id || "";
                    const itemStatus = normalizeStatus(item.metadata?.status);
                    const conflictCount = item.diff?.conflictingNodeIds?.length || 0;
                    const hasConflicts = isChangeRequestConflicted(item);
                    const itemChangedCount = item.diff?.changedNodeIds?.length || 0;

                    return (
                      <button
                        key={itemId}
                        type="button"
                        className={cn(
                          "w-full border-b p-3 text-left last:border-b-0 hover:bg-slate-50",
                          hasConflicts ? "border-red-200 bg-red-50/40" : "border-slate-200",
                        )}
                        onClick={() => {
                          onSelectChangeRequest(itemId);
                          setShowDetailView(true);
                        }}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <p className="truncate text-sm font-semibold text-slate-900">
                            {item.metadata?.title?.trim() || "Untitled change request"}
                          </p>
                          <div className="flex items-center gap-2">
                            {hasConflicts ? (
                              <Badge variant="destructive" className="uppercase">
                                Conflicted
                              </Badge>
                            ) : null}
                            <Badge variant={statusBadgeVariant(itemStatus)}>{formatStatusLabel(itemStatus)}</Badge>
                          </div>
                        </div>
                        <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-slate-600">
                          <span>#{itemId.slice(0, 8)}</span>
                          <span>changed nodes: {itemChangedCount}</span>
                          <span className={hasConflicts ? "font-semibold text-red-700" : "text-emerald-700"}>
                            conflicts: {conflictCount}
                          </span>
                          <span>updated: {formatTimestamp(item.metadata?.updatedAt)}</span>
                        </div>
                      </button>
                    );
                  })
                )}
              </div>
            </div>
          </section>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto bg-slate-50">
      <div className="mx-auto max-w-6xl space-y-4 p-5 md:p-7">
        {!selectedChangeRequest ? (
          <section className="rounded-xl border border-slate-200 bg-white p-4">
            <Button variant="ghost" size="sm" className="px-1" onClick={() => setShowDetailView(false)}>
              <ArrowLeft className="h-4 w-4" />
              Back to Change Requests
            </Button>
            <p className="mt-2 text-sm text-slate-600">This change request is no longer available.</p>
          </section>
        ) : (
          <>
            <section className="rounded-xl border border-slate-200 bg-white p-4">
              <Button variant="ghost" size="sm" className="mb-3 px-1" onClick={() => setShowDetailView(false)}>
                <ArrowLeft className="h-4 w-4" />
                Back to Change Requests
              </Button>
              {selectedHasConflicts ? (
                <div className="mb-3 rounded-md border-2 border-red-300 bg-red-50 px-3 py-2 text-sm font-semibold text-red-800">
                  This change request is conflicted. It cannot be approved until conflicts are resolved.
                </div>
              ) : null}
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-lg font-semibold text-slate-900">
                      {selectedChangeRequest.metadata?.title?.trim() || "Untitled change request"}
                    </h3>
                    <span className="text-sm text-slate-500">#{selectedChangeRequestIdSafe.slice(0, 8)}</span>
                  </div>
                  <p className="text-sm text-slate-600">
                    {selectedChangeRequest.metadata?.owner?.name || "Unknown owner"} opened this change request on{" "}
                    {formatTimestamp(selectedChangeRequest.metadata?.createdAt)}.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {selectedHasConflicts ? (
                    <Badge variant="destructive" className="uppercase">
                      Conflicted
                    </Badge>
                  ) : null}
                  <Badge variant={statusBadgeVariant(selectedStatus)}>{formatStatusLabel(selectedStatus)}</Badge>
                </div>
              </div>
            </section>

            <div className="grid gap-4 lg:grid-cols-[1fr_280px]">
              <section className="space-y-4">
                <Tabs value={selectedTab} onValueChange={(value) => setSelectedTab(value as "conversation" | "nodes")}>
                  <TabsList className="w-full justify-start gap-1 rounded-md border border-slate-200 bg-white p-1">
                    <TabsTrigger value="conversation">Conversation</TabsTrigger>
                    <TabsTrigger value="nodes">Nodes changed ({selectedChangeNodeDiffs.length})</TabsTrigger>
                  </TabsList>

                  <TabsContent value="conversation" className="mt-3 space-y-3">
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <p className="text-sm font-semibold text-slate-900">Description</p>
                      <p className="mt-2 whitespace-pre-wrap text-sm text-slate-700">
                        {selectedChangeRequest.metadata?.description?.trim() || "No description provided."}
                      </p>
                    </div>

                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <p className="text-sm font-semibold text-slate-900">Summary</p>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs">
                        <span className="rounded border border-slate-200 bg-slate-50 px-2 py-1 text-slate-700">
                          changed nodes: {changedNodeIds.length}
                        </span>
                        <span
                          className={cn(
                            "rounded border px-2 py-1",
                            selectedHasConflicts
                              ? "border-red-200 bg-red-50 text-red-700"
                              : "border-emerald-200 bg-emerald-50 text-emerald-700",
                          )}
                        >
                          conflicts: {conflictingNodeIds.length}
                        </span>
                        <span className="rounded border border-slate-200 bg-slate-50 px-2 py-1 text-slate-700">
                          base version:{" "}
                          {selectedChangeRequest.metadata?.basedOnVersionId?.slice(0, 8) || "current live"}
                        </span>
                        <span className="rounded border border-slate-200 bg-slate-50 px-2 py-1 text-slate-700">
                          updated: {formatTimestamp(selectedChangeRequest.metadata?.updatedAt)}
                        </span>
                      </div>
                    </div>

                    {selectedHasConflicts ? (
                      <div className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">
                        <p className="font-semibold">Conflicts detected</p>
                        <p className="mt-1 text-xs">Resolve conflicts before approving this change request.</p>
                        <p className="mt-2 break-all text-xs">{conflictingNodeIds.join(", ")}</p>
                      </div>
                    ) : null}
                  </TabsContent>

                  <TabsContent value="nodes" className="mt-3">
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <div className="mb-3 flex flex-wrap items-center gap-2 text-[11px]">
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

                      {selectedChangeNodeDiffs.length === 0 ? (
                        <p className="text-sm text-slate-600">No node-level differences found.</p>
                      ) : (
                        <Accordion type="multiple" className="rounded-md border border-slate-200 bg-white px-3">
                          {selectedChangeNodeDiffs.map((nodeDiff) => (
                            <AccordionItem key={nodeDiff.nodeID} value={nodeDiff.nodeID} className="border-slate-200">
                              <AccordionTrigger className="py-3 text-xs hover:no-underline">
                                <div className="min-w-0 flex-1 text-left">
                                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                                    <span className="truncate font-semibold text-slate-900">{nodeDiff.nodeName}</span>
                                    <span className="truncate text-[11px] text-slate-400">{nodeDiff.nodeID}</span>
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
                    </div>
                  </TabsContent>
                </Tabs>
              </section>

              <aside className="space-y-3">
                <div className="rounded-xl border border-slate-200 bg-white p-4">
                  <p className="text-sm font-semibold text-slate-900">Review Actions</p>
                  <p className="mt-1 text-xs text-slate-600">Conflicted requests must be resolved or rejected first.</p>
                  <div className="mt-3 space-y-2">
                    <Button
                      className="w-full justify-center"
                      onClick={() => onApprove(selectedChangeRequestIdSafe)}
                      disabled={!canApprove || actionPending || !selectedChangeRequestIdSafe}
                    >
                      Approve
                    </Button>
                    <Button
                      className="w-full justify-center"
                      variant="destructive"
                      onClick={() => onReject(selectedChangeRequestIdSafe)}
                      disabled={!canReject || actionPending || !selectedChangeRequestIdSafe}
                    >
                      Reject
                    </Button>
                    {selectedStatus === "rejected" ? (
                      <Button
                        className="w-full justify-center"
                        variant="outline"
                        onClick={() => onReopen(selectedChangeRequestIdSafe)}
                        disabled={!canReopen || actionPending || !selectedChangeRequestIdSafe}
                      >
                        Reopen
                      </Button>
                    ) : null}
                  </div>
                </div>

                {selectedHasConflicts ? (
                  <div className="rounded-xl border border-slate-200 bg-white p-4">
                    <p className="text-sm font-semibold text-slate-900">Conflict Resolution</p>
                    <p className="mt-1 text-xs text-slate-600">
                      Conflicts found in this request. Open resolver to merge node changes.
                    </p>
                    <Button
                      className="mt-3 w-full justify-center"
                      variant="secondary"
                      onClick={() => setResolvingChangeRequestID(selectedChangeRequestIdSafe)}
                      disabled={!canResolveConflicts || resolvePending}
                    >
                      Resolve Conflicts
                    </Button>
                  </div>
                ) : null}
              </aside>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
