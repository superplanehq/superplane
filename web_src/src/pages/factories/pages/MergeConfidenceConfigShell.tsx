import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { MergeConfidenceCanvasProvider } from "@/lib/mergeConfidenceCanvas";
import { cn } from "@/lib/utils";
import type { PointerEvent as ReactPointerEvent, ReactNode, RefObject } from "react";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import { MERGE_CONFIDENCE_CONFIG_COPY } from "./mergeConfidenceCopy";
import { MergeConfidenceActionsMenu, MergeConfidenceDeleteDialog } from "./MergeConfidenceConfigActions";
import { type SplitConfigDeleteCopy } from "./splitConfigDeleteCopy";
import { NodeConfigPanel } from "./NodeConfigPanel";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import { SettingsAutomationHeaderRow, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

type MergeConfidenceConfigTab = "automation" | "runs";

interface MergeConfidenceConfigShellProps {
  title: string;
  page: boolean;
  onClose: () => void;
  onDelete?: () => Promise<void> | void;
  deleteCopy: SplitConfigDeleteCopy;
  deletePending: boolean;
  deleteError?: string;
  confirmDelete: boolean;
  onConfirmDelete: () => void;
  onConfirmDeleteChange: (open: boolean) => void;
  tab: MergeConfidenceConfigTab;
  showRunsTab: boolean;
  onTabChange: (tab: MergeConfidenceConfigTab) => void;
  containerRef: RefObject<HTMLDivElement | null>;
  selectedNode?: SuperplaneComponentsNode;
  nodeBody?: ReactNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  widthPercent: number;
  isResizing: boolean;
  onResizeStart: (event: ReactPointerEvent<HTMLDivElement>) => void;
  onClosePanel?: () => void;
  onSaveNode: (update: NodeConfigurationUpdate) => Promise<void> | void;
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
}

export function MergeConfidenceConfigShell(props: MergeConfidenceConfigShellProps) {
  return (
    <MergeConfidenceCanvasProvider>
      {props.page ? (
        <section
          className="flex h-full min-h-0 flex-col overflow-hidden bg-background pt-[env(safe-area-inset-top)] dark:bg-gray-900"
          data-testid="merge-confidence-config"
          aria-label={props.title}
        >
          <MergeConfidenceConfigContent {...props} />
        </section>
      ) : (
        <PopupShell testId="merge-confidence-config" canvas fixed onDismiss={props.onClose}>
          <MergeConfidenceConfigContent {...props} />
        </PopupShell>
      )}
    </MergeConfidenceCanvasProvider>
  );
}

function MergeConfidenceConfigContent(props: MergeConfidenceConfigShellProps) {
  return (
    <>
      <PopupHeader
        title={props.title}
        onClose={props.onClose}
        actions={
          props.onDelete ? (
            <MergeConfidenceActionsMenu label={props.deleteCopy.menuLabel} onDelete={props.onConfirmDelete} />
          ) : null
        }
      >
        <SettingsAutomationHeaderRow
          tabs={<ConfigTabs tab={props.tab} showRunsTab={props.showRunsTab} onTabChange={props.onTabChange} />}
        />
      </PopupHeader>
      <MergeConfidenceConfigBody {...configBodyProps(props)} />
      {props.onDelete ? (
        <MergeConfidenceDeleteDialog
          open={props.confirmDelete}
          pending={props.deletePending}
          error={props.deleteError}
          copy={props.deleteCopy}
          onOpenChange={props.onConfirmDeleteChange}
          onConfirm={props.onDelete}
        />
      ) : null}
    </>
  );
}

function configBodyProps(props: MergeConfidenceConfigShellProps) {
  return {
    containerRef: props.containerRef,
    selectedNode: props.selectedNode,
    nodeBody: props.nodeBody,
    organizationId: props.organizationId,
    factoryId: props.factoryId,
    factoryKey: props.factoryKey,
    widthPercent: props.widthPercent,
    isResizing: props.isResizing,
    onResizeStart: props.onResizeStart,
    onClosePanel: props.onClosePanel,
    onSaveNode: props.onSaveNode,
    tab: props.tab,
    graph: props.graph,
    loading: props.loading,
    error: props.error,
    onRetry: props.onRetry,
    canvasId: props.canvasId,
    runHrefFor: props.runHrefFor,
    selectedRunId: props.selectedRunId,
    onSelectedRunIdChange: props.onSelectedRunIdChange,
    onNodeSelect: props.onNodeSelect,
    selectedNodeId: props.selectedNodeId,
    focusNonce: props.focusNonce,
    layoutFitNonce: props.layoutFitNonce,
  };
}

function ConfigTabs({
  tab,
  showRunsTab,
  onTabChange,
}: {
  tab: MergeConfidenceConfigTab;
  showRunsTab: boolean;
  onTabChange: (tab: MergeConfidenceConfigTab) => void;
}) {
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as MergeConfidenceConfigTab)}>
      <TabsList aria-label={MERGE_CONFIDENCE_CONFIG_COPY.tabsLabel}>
        <TabsTrigger value="automation" data-testid="merge-confidence-config-tab-automation">
          {MERGE_CONFIDENCE_CONFIG_COPY.automationTab}
        </TabsTrigger>
        {showRunsTab ? (
          <TabsTrigger value="runs" data-testid="merge-confidence-config-tab-runs">
            {MERGE_CONFIDENCE_CONFIG_COPY.runsTab}
          </TabsTrigger>
        ) : null}
      </TabsList>
    </Tabs>
  );
}

function MergeConfidenceConfigBody({
  containerRef,
  selectedNode,
  nodeBody,
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
}: ReturnType<typeof configBodyProps>) {
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
          body={nodeBody}
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

function viewEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return COLUMN_AUTOMATIONS_COPY.viewLoading;
  }
  return error ? COLUMN_AUTOMATIONS_COPY.viewError : COLUMN_AUTOMATIONS_COPY.viewEmpty;
}
