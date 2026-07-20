import { useCallback, useMemo, useRef } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { RightSideControls } from "@/ui/CanvasPage/RightSideControls";
import type { DraftConsoleDiffSummary } from "../draftConsoleDiff";

import type {
  ConsoleLayoutItem,
  ConsolePanel,
  CanvasConsoleQueryResult,
  UpdateCanvasConsoleMutationResult,
} from "@/hooks/useCanvasData";

import { ConsoleView } from "./ConsoleView";
import { ConsoleYamlModal } from "./ConsoleYamlModal";
import type { ConsoleContextValue, ConsoleNodeStatus } from "./ConsoleContext";
import { ConsoleContextProvider } from "./ConsoleContextProvider";

export type ConsoleOverlayProps = {
  /**
   * Aggregated read-only flag: true when the viewer cannot edit the dashboard
   * structure (panels, layout, markdown body) for any reason — missing the
   * `canvases:update` permission, viewing a template, or the canvas being
   * deleted remotely. Mirrors the server-side guard in
   * `UpdateCanvasDashboard`; the server remains the source of truth.
   */
  readOnly: boolean;
  /**
   * True when the viewer can import/replace the dashboard with YAML. Falls
   * back to false when `readOnly` is true. Server-side authorization remains
   * the source of truth; this is for UX only.
   */
  canImportYaml: boolean;
  /**
   * True when the viewer can invoke runtime actions on the underlying canvas:
   * trigger chips, run-trigger row actions, approve/cancel/push-through
   * execution hooks. This maps 1:1 to the `canvases:update` permission used
   * by the `InvokeNodeTriggerHook` / `InvokeNodeExecutionHook` interceptor
   * rules; the same backend rules apply even if the UI is bypassed.
   */
  canRunNodes: boolean;
  consoleQuery: CanvasConsoleQueryResult;
  updateConsoleMutation: UpdateCanvasConsoleMutationResult;
  addPanelDialogOpen: boolean;
  onAddPanelDialogOpenChange: (open: boolean) => void;
  /** Controlled state for the YAML modal — owned by the canvas page header. */
  yamlModalOpen: boolean;
  onYamlModalOpenChange: (open: boolean) => void;
  canvasId?: string;
  canvasName?: string;
  /** Organization id for chip navigation. Required when chips should be live. */
  organizationId?: string;
  /** Canvas nodes used to resolve chip references by id or name. */
  canvasNodes?: SuperplaneComponentsNode[];
  /** True while the canvas nodes are still being fetched. */
  canvasNodesLoading?: boolean;
  /** Latest known node status per node id; powers the status chip. */
  nodeStatuses?: Record<string, ConsoleNodeStatus | undefined>;
  /** Callback invoked when a manual-run chip is clicked. */
  onTriggerNode?: ConsoleContextValue["onTriggerNode"];
  showConsoleEditControls?: boolean;
  onConsoleAddPanel?: () => void;
  onConsoleOpenYaml?: () => void;
  consoleYamlReadOnly?: boolean;
  visualDiff?: {
    enabled: boolean;
    summary?: DraftConsoleDiffSummary;
  };
  onEffectiveConsoleChange?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void;
};

export function ConsoleOverlay({
  readOnly,
  canImportYaml,
  canRunNodes,
  consoleQuery,
  updateConsoleMutation,
  addPanelDialogOpen,
  onAddPanelDialogOpenChange,
  yamlModalOpen,
  onYamlModalOpenChange,
  canvasId,
  canvasName,
  organizationId,
  canvasNodes,
  canvasNodesLoading,
  nodeStatuses,
  onTriggerNode,
  showConsoleEditControls = false,
  onConsoleAddPanel,
  onConsoleOpenYaml,
  consoleYamlReadOnly,
  visualDiff,
  onEffectiveConsoleChange,
}: ConsoleOverlayProps) {
  const updateConsoleMutationRef = useRef(updateConsoleMutation);
  updateConsoleMutationRef.current = updateConsoleMutation;

  const panels: ConsolePanel[] = useMemo(() => consoleQuery.data?.panels ?? [], [consoleQuery.data?.panels]);

  const layout: ConsoleLayoutItem[] = useMemo(() => consoleQuery.data?.layout ?? [], [consoleQuery.data?.layout]);

  const handleChange = useCallback((next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => {
    updateConsoleMutationRef.current.mutate(next);
  }, []);

  const handleImportYaml = useCallback(async (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => {
    // Use mutateAsync so the modal can await before closing/showing toasts.
    await updateConsoleMutationRef.current.mutateAsync(next);
  }, []);

  const contextNodes = canvasNodes ?? [];
  const leftOffset = useEffectiveLeftSidebarWidth();
  // The YAML modal is opened from the canvas page header (next to "Add panel").
  // The dashboard overlay no longer renders its own toolbar so the white
  // strip is gone and the grid fills the available area.
  const overlayContent = (
    <>
      <div
        className="absolute bottom-0 top-[5rem] z-10 flex flex-row bg-slate-100 dark:bg-gray-900"
        style={{ left: leftOffset, right: 0 }}
        data-testid="console-overlay"
      >
        <div className="min-h-0 flex-1 overflow-auto">
          <ConsoleView
            panels={panels}
            layout={layout}
            isLoading={consoleQuery.isLoading}
            errorMessage={consoleQuery.error ? String(consoleQuery.error) : undefined}
            readOnly={readOnly}
            onChange={handleChange}
            onEffectiveChange={onEffectiveConsoleChange}
            addPanelDialogOpen={addPanelDialogOpen}
            onAddPanelDialogOpenChange={onAddPanelDialogOpenChange}
            visualDiff={visualDiff}
          />
        </div>
        {showConsoleEditControls ? (
          <RightSideControls
            mode="edit"
            layout="embedded"
            consoleEditControls
            onConsoleAddPanel={onConsoleAddPanel}
            onConsoleOpenYaml={onConsoleOpenYaml}
            consoleYamlReadOnly={consoleYamlReadOnly}
          />
        ) : null}
      </div>

      <ConsoleYamlModal
        open={yamlModalOpen}
        onOpenChange={onYamlModalOpenChange}
        panels={panels}
        layout={layout}
        canvasId={canvasId}
        canvasName={canvasName}
        onImport={canImportYaml ? handleImportYaml : undefined}
        isImporting={updateConsoleMutation.isPending}
      />
    </>
  );

  if (!canvasId || !organizationId) {
    return overlayContent;
  }

  return (
    <ConsoleContextProvider
      canvasId={canvasId}
      organizationId={organizationId}
      nodes={contextNodes}
      nodesLoading={canvasNodesLoading}
      nodeStatuses={nodeStatuses}
      canRunNodes={canRunNodes}
      onTriggerNode={canRunNodes ? onTriggerNode : undefined}
    >
      {overlayContent}
    </ConsoleContextProvider>
  );
}
