import { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import Editor from "@monaco-editor/react";
import type { Monaco } from "@monaco-editor/react";
import { AlertTriangle, ArrowLeft, Check, CheckCircle2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { editor as MonacoEditor } from "monaco-editor";
import {
  buildConflictMarkerYAML,
  buildNodeMap,
  cloneJSON,
  concatenateBothNodes,
  concatenateConflictBlockLines,
  localResolutionLabel,
  parseNodeYAML,
  prettyYAML,
  pruneEdgesByNodes,
  upsertNode,
} from "./conflictResolverUtils";

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

interface CanvasChangeRequestConflictResolverProps {
  liveCanvasVersion?: CanvasesCanvasVersion;
  changeRequest: CanvasesCanvasChangeRequest;
  canvasName: string;
  canvasDescription?: string;
  isSubmitting: boolean;
  onBack: () => void;
  onSubmit: (data: { changeRequestId: string; nodes: CanvasNodeLike[]; edges: CanvasEdgeLike[] }) => Promise<void>;
}

export function CanvasChangeRequestConflictResolver({
  liveCanvasVersion,
  changeRequest,
  canvasName,
  canvasDescription,
  isSubmitting,
  onBack,
  onSubmit,
}: CanvasChangeRequestConflictResolverProps) {
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
      } else {
        replacementLines = concatenateConflictBlockLines(currentLines, incomingLines);
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
      if (resolution === "both") {
        setFinalDraftError(
          "Both changes were written. The resulting YAML has duplicate keys and is invalid — please edit it manually before marking as resolved.",
        );
      } else {
        setFinalDraftError("");
      }
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

    const selectedNode = incomingNode || currentNode;
    if (!currentNode && !incomingNode) {
      setFinalNodes((current) => upsertNode(current, selectedNodeID, null));
      setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
      setFinalDraftYAML("null\n");
      setFinalDraftError("");
      return;
    }

    const bothYAML = concatenateBothNodes(currentNode, incomingNode);
    setFinalDraftYAML(bothYAML);

    if (!currentNode || !incomingNode) {
      setFinalNodes((current) => upsertNode(current, selectedNodeID, selectedNode ? cloneJSON(selectedNode) : null));
      setResolvedNodeIDs((current) => new Set(current).add(selectedNodeID));
      setFinalDraftError("");
      return;
    }

    setFinalDraftError(
      "Both changes were written. The resulting YAML has duplicate keys and is invalid — please edit it manually before marking as resolved.",
    );
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
                Back
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
            <LoadingButton
              onClick={handleResolve}
              disabled={!canvasName || !allConflictsMarkedResolved}
              loading={isSubmitting}
              loadingText="Resolving..."
            >
              <Check className="h-4 w-4" />
              Save resolved result
            </LoadingButton>
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
