import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MergeConfidenceCanvasProvider } from "@/lib/mergeConfidenceCanvas";
import { cn } from "@/lib/utils";
import { Ellipsis } from "lucide-react";
import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent, type RefObject } from "react";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import { MERGE_CONFIDENCE_CONFIG_COPY } from "./mergeConfidenceCopy";
import { NodeConfigPanel } from "./NodeConfigPanel";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import { SettingsAutomationHeaderRow, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
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
}

type MergeConfidenceConfigTab = "automation" | "runs";

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
}: MergeConfidenceConfigModalProps) {
  const [tab, setTab] = useState<MergeConfidenceConfigTab>("automation");
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const initialNodeId = firstComponentId(graph);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(initialNodeId);
  const appliedInitialNode = useRef(initialNodeId !== null);
  const [focusNonce, setFocusNonce] = useState(0);
  const [layoutFitNonce, setLayoutFitNonce] = useState<number | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const panelWasOpen = useRef(false);
  const split = useSplitRunPanePercent({ defaultPercent: 50, minPercent: 28, maxPercent: 72 });

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
  const selectedNode =
    tab === "automation" && selectedNodeId ? graph?.specNodes?.find((node) => node.id === selectedNodeId) : undefined;

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

  return (
    <MergeConfidenceCanvasProvider>
      <PopupShell testId="merge-confidence-config" canvas fixed onDismiss={onClose}>
        <PopupHeader
          title={title}
          onClose={onClose}
          actions={onDelete ? <MergeConfidenceActionsMenu onDelete={() => setConfirmDelete(true)} /> : null}
        >
          <SettingsAutomationHeaderRow tabs={<ConfigTabs tab={tab} onTabChange={setTab} />} />
        </PopupHeader>
        <MergeConfidenceConfigBody
          containerRef={split.containerRef}
          selectedNode={selectedNode}
          organizationId={graph?.organizationId}
          factoryId={panelFactoryId(factoryId, graph)}
          factoryKey={factoryKey}
          widthPercent={split.percent}
          isResizing={split.isResizing}
          onResizeStart={split.startResize}
          onClosePanel={() => setSelectedNodeId(null)}
          onSaveNode={onSaveNode}
          tab={tab}
          graph={graph}
          loading={loading}
          error={error}
          onRetry={onRetry}
          canvasId={canvasId}
          runHrefFor={runHrefFor}
          selectedRunId={selectedRunId}
          onSelectedRunIdChange={setSelectedRunId}
          onNodeSelect={setSelectedNodeId}
          selectedNodeId={selectedNodeId}
          focusNonce={focusNonce}
          layoutFitNonce={layoutFitNonce}
        />
        {onDelete ? (
          <MergeConfidenceDeleteDialog
            open={confirmDelete}
            pending={deletePending}
            error={deleteError}
            onOpenChange={setConfirmDelete}
            onConfirm={onDelete}
          />
        ) : null}
      </PopupShell>
    </MergeConfidenceCanvasProvider>
  );
}

function MergeConfidenceConfigBody({
  containerRef,
  selectedNode,
  organizationId,
  factoryId,
  factoryKey,
  widthPercent,
  isResizing,
  onResizeStart,
  onClosePanel,
  onSaveNode,
  tab,
  graph,
  loading,
  error,
  onRetry,
  canvasId,
  runHrefFor,
  selectedRunId,
  onSelectedRunIdChange,
  onNodeSelect,
  selectedNodeId,
  focusNonce,
  layoutFitNonce,
}: {
  containerRef: RefObject<HTMLDivElement | null>;
  selectedNode?: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  widthPercent: number;
  isResizing: boolean;
  onResizeStart: (event: ReactPointerEvent<HTMLDivElement>) => void;
  onClosePanel: () => void;
  onSaveNode: (update: NodeConfigurationUpdate) => Promise<void> | void;
  tab: MergeConfidenceConfigTab;
  graph?: IntakeAutomationGraph;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  selectedRunId: string | null;
  onSelectedRunIdChange: (runId: string | null) => void;
  onNodeSelect: (nodeId: string) => void;
  selectedNodeId: string | null;
  focusNonce: number;
  layoutFitNonce: number | null;
}) {
  return (
    <div
      ref={containerRef}
      className="flex min-h-0 min-w-0 flex-1"
      data-testid="merge-confidence-config-body"
      data-split={selectedNode ? "true" : "false"}
    >
      {selectedNode ? (
        <NodeConfigPanel
          node={selectedNode}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          widthPercent={widthPercent}
          onClose={onClosePanel}
          onSave={onSaveNode}
        />
      ) : null}
      {selectedNode ? <ConfigResizeHandle isResizing={isResizing} onPointerDown={onResizeStart} /> : null}
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        {tab === "runs" ? (
          <MergeConfidenceCanvas
            graph={graph}
            loading={loading}
            error={error}
            onRetry={onRetry}
            canvasId={canvasId}
            runHrefFor={runHrefFor}
            showRuns
            selectLatestRun
            selectedRunId={selectedRunId}
            onSelectedRunIdChange={onSelectedRunIdChange}
            onNodeSelect={() => undefined}
          />
        ) : (
          <MergeConfidenceCanvas
            graph={graph}
            loading={loading}
            error={error}
            onRetry={onRetry}
            canvasId={canvasId}
            runHrefFor={runHrefFor}
            showRuns={false}
            onNodeSelect={onNodeSelect}
            focusNodeId={selectedNodeId}
            focusNonce={focusNonce}
            focusFit={false}
            lockNativeZoom
            layoutFitNonce={layoutFitNonce}
          />
        )}
      </div>
    </div>
  );
}

function ConfigResizeHandle({
  isResizing,
  onPointerDown,
}: {
  isResizing: boolean;
  onPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize step settings"
      data-testid="merge-confidence-config-resize"
      onPointerDown={onPointerDown}
      className="group relative z-10 w-2 shrink-0 cursor-col-resize touch-none"
    >
      <div
        aria-hidden
        className={cn(
          "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-foreground/30",
          isResizing && "bg-foreground/30",
        )}
      />
    </div>
  );
}

function ConfigTabs({
  tab,
  onTabChange,
}: {
  tab: MergeConfidenceConfigTab;
  onTabChange: (tab: MergeConfidenceConfigTab) => void;
}) {
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as MergeConfidenceConfigTab)}>
      <TabsList aria-label={MERGE_CONFIDENCE_CONFIG_COPY.tabsLabel}>
        <TabsTrigger value="automation" data-testid="merge-confidence-config-tab-automation">
          {MERGE_CONFIDENCE_CONFIG_COPY.automationTab}
        </TabsTrigger>
        <TabsTrigger value="runs" data-testid="merge-confidence-config-tab-runs">
          {MERGE_CONFIDENCE_CONFIG_COPY.runsTab}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
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

function MergeConfidenceCanvas({
  graph,
  loading,
  error,
  onRetry,
  canvasId,
  runHrefFor,
  showRuns,
  selectLatestRun = false,
  selectedRunId,
  onSelectedRunIdChange,
  onNodeSelect,
  focusNodeId = null,
  focusNonce = 0,
  focusFit = true,
  lockNativeZoom = false,
  layoutFitNonce = null,
}: {
  graph?: IntakeAutomationGraph;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  showRuns: boolean;
  selectLatestRun?: boolean;
  selectedRunId?: string | null;
  onSelectedRunIdChange?: (runId: string | null) => void;
  onNodeSelect?: (nodeId: string) => void;
  focusNodeId?: string | null;
  focusNonce?: number;
  focusFit?: boolean;
  lockNativeZoom?: boolean;
  layoutFitNonce?: number | null;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <section
        className="settings-graph relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
        aria-label="Automation"
        data-testid="merge-confidence-config-canvas"
      >
        <p className="workspace-body-text text-muted-foreground">{viewEmptyMessage(loading, error)}</p>
        {error && onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {COLUMN_AUTOMATIONS_COPY.viewRetry}
          </Button>
        ) : null}
      </section>
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="merge-confidence-config-canvas"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      onNodeSelect={onNodeSelect}
      showRuns={showRuns}
      selectLatestRun={selectLatestRun}
      selectedRunId={selectedRunId}
      onSelectedRunIdChange={onSelectedRunIdChange}
      showStatusControls={false}
      showFindControls={false}
      focusNodeId={focusNodeId}
      focusNonce={focusNonce}
      focusFit={focusFit}
      lockNativeZoom={lockNativeZoom}
      layoutFitNonce={layoutFitNonce}
      className="settings-graph"
    />
  );
}

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full text-foreground hover:bg-slate-950/5 dark:hover:bg-white/10";

function MergeConfidenceActionsMenu({ onDelete }: { onDelete: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={HEADER_ICON_BUTTON}
          aria-label="More actions"
          data-testid="merge-confidence-config-actions"
        >
          <Ellipsis className="h-4 w-4" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuItem
          className="text-destructive focus:text-destructive"
          onSelect={onDelete}
          data-testid="merge-confidence-config-delete"
        >
          {COLUMN_AUTOMATIONS_COPY.deleteLabel}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MergeConfidenceDeleteDialog({
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void> | void;
}) {
  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!pending) {
          onOpenChange(next);
        }
      }}
    >
      <AlertDialogContent data-testid="merge-confidence-config-delete-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>{COLUMN_AUTOMATIONS_COPY.deleteLabel}</AlertDialogTitle>
          <AlertDialogDescription>{COLUMN_AUTOMATIONS_COPY.confirmDelete}</AlertDialogDescription>
          {error ? (
            <p className="text-sm text-destructive" data-testid="merge-confidence-config-delete-error">
              {error}
            </p>
          ) : null}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{COLUMN_AUTOMATIONS_COPY.keepLabel}</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-white hover:bg-destructive/90"
            disabled={pending}
            data-testid="merge-confidence-config-delete-confirm"
            onClick={(event) => {
              event.preventDefault();
              void onConfirm();
            }}
          >
            {pending ? COLUMN_AUTOMATIONS_COPY.deletingLabel : COLUMN_AUTOMATIONS_COPY.deleteLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function viewEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return COLUMN_AUTOMATIONS_COPY.viewLoading;
  }
  return error ? COLUMN_AUTOMATIONS_COPY.viewError : COLUMN_AUTOMATIONS_COPY.viewEmpty;
}
