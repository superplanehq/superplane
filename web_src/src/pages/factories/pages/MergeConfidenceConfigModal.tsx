import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { DEFAULT_SPLIT_CONFIG_DELETE_COPY, type SplitConfigDeleteCopy } from "./splitConfigDeleteCopy";
import { MergeConfidenceConfigShell } from "./MergeConfidenceConfigShell";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { useSplitRunPanePercent } from "./work-order-split-run/useSplitRunPanePercent";

interface MergeConfidenceConfigModalProps {
  title: string;
  graph?: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  canvasId?: string;
  factoryId?: string;
  factoryKey?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  onClose: () => void;
  onSaveNode: (update: NodeConfigurationUpdate) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
  deleteError?: string;
  deleteCopy?: SplitConfigDeleteCopy;
  page?: boolean;
  showRunsTab?: boolean;
  renderNodeBody?: (node: SuperplaneComponentsNode) => ReactNode;
  fallbackNode?: SuperplaneComponentsNode;
}

/** Merge confidence configuration. The first component starts selected. */
export function MergeConfidenceConfigModal({
  title,
  graph,
  loading = false,
  error = false,
  onRetry,
  canvasId,
  factoryId,
  factoryKey,
  runHrefFor,
  onClose,
  onSaveNode,
  onDelete,
  deletePending = false,
  deleteError,
  deleteCopy = DEFAULT_SPLIT_CONFIG_DELETE_COPY,
  page = false,
  showRunsTab = true,
  renderNodeBody,
  fallbackNode,
}: MergeConfidenceConfigModalProps) {
  const [tab, setTab] = useState<"automation" | "runs">("automation");
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const selection = useCanvasNodeSelection(graph);
  const focus = usePanelFocus(tab, selection.selectedNodeId);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const split = useSplitRunPanePercent({ defaultPercent: 50, minPercent: 28, maxPercent: 72 });
  const selectedNode = nodeForSelection(tab, selection.selectedNodeId, graph);
  const panelNode = visiblePanelNode(tab, selectedNode, selection.selectedNodeId, focus.panelWasOpen, fallbackNode);

  return (
    <MergeConfidenceConfigShell
      title={title}
      page={page}
      onClose={onClose}
      onDelete={onDelete}
      deleteCopy={deleteCopy}
      deletePending={deletePending}
      deleteError={deleteError}
      confirmDelete={confirmDelete}
      onConfirmDelete={() => setConfirmDelete(true)}
      onConfirmDeleteChange={setConfirmDelete}
      tab={tab}
      showRunsTab={showRunsTab}
      onTabChange={setTab}
      containerRef={split.containerRef}
      selectedNode={panelNode}
      nodeBody={panelNode ? renderNodeBody?.(panelNode) : undefined}
      organizationId={graph?.organizationId}
      factoryId={panelFactoryId(factoryId, graph)}
      factoryKey={factoryKey}
      widthPercent={split.percent}
      isResizing={split.isResizing}
      onResizeStart={split.startResize}
      onClosePanel={selectedNode ? () => selection.setSelectedNodeId(null) : undefined}
      onSaveNode={onSaveNode}
      graph={graph}
      loading={loading}
      error={error}
      onRetry={onRetry}
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      selectedRunId={selectedRunId}
      onSelectedRunIdChange={setSelectedRunId}
      onNodeSelect={selection.setSelectedNodeId}
      selectedNodeId={selection.selectedNodeId}
      focusNonce={focus.focusNonce}
      layoutFitNonce={focus.layoutFitNonce}
    />
  );
}

function useCanvasNodeSelection(graph: IntakeAutomationGraph | undefined) {
  const initialNodeId = firstComponentId(graph);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(initialNodeId);
  const appliedInitialNode = useRef(initialNodeId !== null);

  useEffect(() => {
    if (appliedInitialNode.current) {
      return;
    }
    const nextId = firstComponentId(graph);
    if (!nextId) {
      return;
    }
    appliedInitialNode.current = true;
    setSelectedNodeId(nextId);
  }, [graph]);

  return { selectedNodeId, setSelectedNodeId };
}

function usePanelFocus(tab: "automation" | "runs", selectedNodeId: string | null) {
  const [focusNonce, setFocusNonce] = useState(0);
  const [layoutFitNonce, setLayoutFitNonce] = useState<number | null>(null);
  const panelWasOpen = useRef(false);

  useEffect(() => {
    if (tab !== "automation") {
      return;
    }
    if (selectedNodeId) {
      panelWasOpen.current = true;
      const timeout = window.setTimeout(() => setFocusNonce((nonce) => nonce + 1), 80);
      return () => window.clearTimeout(timeout);
    }
    if (!panelWasOpen.current) {
      return;
    }
    const timeout = window.setTimeout(() => setLayoutFitNonce((nonce) => (nonce ?? 0) + 1), 80);
    return () => window.clearTimeout(timeout);
  }, [selectedNodeId, tab]);

  return { focusNonce, layoutFitNonce, panelWasOpen: panelWasOpen.current };
}

function nodeForSelection(
  tab: "automation" | "runs",
  selectedNodeId: string | null,
  graph: IntakeAutomationGraph | undefined,
) {
  if (tab !== "automation" || !selectedNodeId) {
    return undefined;
  }
  return graph?.specNodes?.find((node) => node.id === selectedNodeId);
}

function visiblePanelNode(
  tab: "automation" | "runs",
  selectedNode: SuperplaneComponentsNode | undefined,
  selectedNodeId: string | null,
  panelWasOpen: boolean,
  fallbackNode: SuperplaneComponentsNode | undefined,
) {
  if (selectedNode) {
    return selectedNode;
  }
  if (tab !== "automation" || (panelWasOpen && !selectedNodeId)) {
    return undefined;
  }
  return fallbackNode;
}

function panelFactoryId(factoryId: string | undefined, graph: IntakeAutomationGraph | undefined) {
  return factoryId ?? graph?.factoryId;
}

function firstComponentId(graph: IntakeAutomationGraph | undefined): string | null {
  const specNodes = graph?.specNodes ?? [];
  const trigger = specNodes.find((node) => node.type === "TYPE_TRIGGER" && node.id?.trim());
  if (trigger?.id) {
    return trigger.id;
  }

  const targets = new Set(graph?.edges?.map((edge) => edge.target).filter(Boolean));
  const root = specNodes.find((node) => node.id && !targets.has(node.id));
  return root?.id?.trim() || specNodes[0]?.id?.trim() || null;
}
