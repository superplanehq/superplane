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
import { Ellipsis } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import { MERGE_CONFIDENCE_CONFIG_COPY } from "./mergeConfidenceCopy";
import { NodeConfigPanel } from "./NodeConfigPanel";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import { SettingsAutomationHeaderRow, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

interface MergeConfidenceConfigModalProps {
  title: string;
  graph?: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  onClose: () => void;
  onSaveNode: (update: NodeConfigurationUpdate) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
}

type MergeConfidenceConfigTab = "automation" | "runs";

/** Merge confidence configuration. The automation stays visible until a step is selected. */
export function MergeConfidenceConfigModal({
  title,
  graph,
  loading = false,
  error = false,
  onRetry,
  canvasId,
  runHrefFor,
  onClose,
  onSaveNode,
  onDelete,
  deletePending = false,
}: MergeConfidenceConfigModalProps) {
  const [tab, setTab] = useState<MergeConfidenceConfigTab>("automation");
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [focusNonce, setFocusNonce] = useState(0);
  const [layoutFitNonce, setLayoutFitNonce] = useState<number | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const panelWasOpen = useRef(false);
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
    <PopupShell testId="merge-confidence-config" canvas fixed onDismiss={onClose}>
      <PopupHeader
        title={title}
        onClose={onClose}
        actions={onDelete ? <MergeConfidenceActionsMenu onDelete={() => setConfirmDelete(true)} /> : null}
      >
        <SettingsAutomationHeaderRow
          tabs={
            <Tabs value={tab} onValueChange={(value) => setTab(value as MergeConfidenceConfigTab)}>
              <TabsList aria-label={MERGE_CONFIDENCE_CONFIG_COPY.tabsLabel}>
                <TabsTrigger value="automation" data-testid="merge-confidence-config-tab-automation">
                  {MERGE_CONFIDENCE_CONFIG_COPY.automationTab}
                </TabsTrigger>
                <TabsTrigger value="runs" data-testid="merge-confidence-config-tab-runs">
                  {MERGE_CONFIDENCE_CONFIG_COPY.runsTab}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          }
        />
      </PopupHeader>
      <div
        className="flex min-h-0 min-w-0 flex-1"
        data-testid="merge-confidence-config-body"
        data-split={selectedNode ? "true" : "false"}
      >
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
              onNodeSelect={setSelectedNodeId}
              focusNodeId={selectedNodeId}
              focusNonce={focusNonce}
              layoutFitNonce={layoutFitNonce}
            />
          )}
        </div>
        {selectedNode ? (
          <NodeConfigPanel
            node={selectedNode}
            organizationId={graph?.organizationId}
            onClose={() => setSelectedNodeId(null)}
            onSave={onSaveNode}
          />
        ) : null}
      </div>
      {onDelete ? (
        <MergeConfidenceDeleteDialog
          open={confirmDelete}
          pending={deletePending}
          onOpenChange={setConfirmDelete}
          onConfirm={onDelete}
        />
      ) : null}
    </PopupShell>
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
  onNodeSelect,
  focusNodeId = null,
  focusNonce = 0,
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
  onNodeSelect?: (nodeId: string) => void;
  focusNodeId?: string | null;
  focusNonce?: number;
  layoutFitNonce?: number | null;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <section
        className="relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
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
      showStatusControls={false}
      showFindControls={false}
      focusNodeId={focusNodeId}
      focusNonce={focusNonce}
      layoutFitNonce={layoutFitNonce}
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
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
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
