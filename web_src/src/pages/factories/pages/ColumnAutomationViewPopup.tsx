import { ScheduledIntakeOwnerSettings } from "./ScheduledIntakeOwnerSettings";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { useFactoryAutomationNavigation } from "@/hooks/useFactoryAutomationNavigation";
import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";
import type { RunHistoryIntakeSource } from "./automationRunHistory/intakeRunData";
import { FactoryAutomationHistory } from "./FactoryAutomationHistory";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Link } from "@/components/Link/link";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Pencil } from "lucide-react";
import { useState, type ReactNode } from "react";

import { ColumnAutomationFooterSlotProvider } from "./columnAutomationFooterSlot";

import { columnAutomationUsesSplitConfig, IMPLEMENT_AGENT_NODE_ID } from "../lib/columnAutomationHeadline";
import { COLUMN_AUTOMATIONS_COPY, type ColumnAutomationViewTab } from "../lib/columnAutomations";
import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { SettingsAutomationCanvasEdit, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import { useIntakeAutomationCanvas, type IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

interface ColumnAutomationViewPopupProps {
  organizationId?: string;
  factoryId?: string;
  historyEligible?: boolean;
  intakeSource?: RunHistoryIntakeSource;
  title: string;
  graph?: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  editHref?: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  onClose: () => void;
  general?: ReactNode;
  agent?: PlanningReviewAgentSlot;
  initialTab?: ColumnAutomationViewTab;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
}

/** Read-only automation canvas in the board popup. Edit opens the full editor. */
export function ColumnAutomationViewPopup({
  organizationId,
  factoryId,
  historyEligible = false,
  intakeSource,
  title,
  graph,
  loading = false,
  error = false,
  onRetry,
  editHref,
  canvasId,
  runHrefFor,
  onClose,
  general,
  agent,
  initialTab,
  onDelete,
  deletePending = false,
}: ColumnAutomationViewPopupProps) {
  const hasGeneral = Boolean(general);
  const hasAgent = Boolean(agent);
  const history = useFactoryAutomationNavigation({
    organizationId,
    factoryId,
    graph,
    agent,
    parameter: "automationTab",
    initialTab,
    hasGeneral,
    hasAgent,
    eligible: historyEligible,
  });
  const { tab, setTab } = history;
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [footerSlot, setFooterSlot] = useState<HTMLDivElement | null>(null);
  const footerSlotValue = onDelete ? (confirmDelete ? null : footerSlot) : undefined;

  return (
    <ColumnAutomationFooterSlotProvider slot={footerSlotValue}>
      <PopupShell testId="column-automation-view" canvas fixed onDismiss={onClose}>
        <PopupHeader
          title={title}
          onClose={onClose}
          actions={
            history.enabled && editHref ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button asChild variant="ghost" size="icon" className="size-7 text-muted-foreground">
                    <Link
                      href={editHref}
                      aria-label={COLUMN_AUTOMATIONS_COPY.editLabel}
                      data-testid="column-automation-view-edit"
                    >
                      <Pencil className="size-4" aria-hidden />
                    </Link>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{COLUMN_AUTOMATIONS_COPY.editLabel}</TooltipContent>
              </Tooltip>
            ) : null
          }
        >
          {hasGeneral || hasAgent ? (
            <FactoryAutomationSettingsTabs
              tab={tab}
              onTabChange={setTab}
              hasGeneral={hasGeneral}
              hasAgent={hasAgent}
              historyEnabled={history.enabled}
              label={COLUMN_AUTOMATIONS_COPY.tabsLabel}
              testId="column-automation-view"
            />
          ) : null}
        </PopupHeader>
        <ColumnAutomationViewBody
          intakeSource={intakeSource}
          history={history}
          tab={tab}
          graph={graph}
          loading={loading}
          error={error}
          onRetry={onRetry}
          canvasId={canvasId}
          runHrefFor={runHrefFor}
          editHref={editHref}
          general={general}
          agent={agent}
        />
        {onDelete ? (
          <ColumnAutomationViewFooter
            confirmDelete={confirmDelete}
            deletePending={deletePending}
            onDelete={onDelete}
            onConfirmDelete={setConfirmDelete}
            onActionsSlot={setFooterSlot}
          />
        ) : null}
      </PopupShell>
    </ColumnAutomationFooterSlotProvider>
  );
}

function ColumnAutomationViewFooter({
  confirmDelete,
  deletePending,
  onDelete,
  onConfirmDelete,
  onActionsSlot,
}: {
  confirmDelete: boolean;
  deletePending: boolean;
  onDelete: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onActionsSlot: (slot: HTMLDivElement | null) => void;
}) {
  return (
    <footer
      className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3"
      data-testid="column-automation-view-footer"
    >
      {confirmDelete ? (
        <>
          <p className="workspace-body-text text-destructive" role="alert">
            {COLUMN_AUTOMATIONS_COPY.confirmDelete}
          </p>
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => onConfirmDelete(false)}>
              {COLUMN_AUTOMATIONS_COPY.keepLabel}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={deletePending}
              onClick={() => void onDelete()}
              data-testid="column-automation-view-delete-confirm"
            >
              {deletePending ? COLUMN_AUTOMATIONS_COPY.deletingLabel : COLUMN_AUTOMATIONS_COPY.deleteLabel}
            </Button>
          </div>
        </>
      ) : (
        <>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onConfirmDelete(true)}
            data-testid="column-automation-view-delete"
          >
            {COLUMN_AUTOMATIONS_COPY.deleteLabel}
          </Button>
          <div ref={onActionsSlot} className="flex items-center gap-3" />
        </>
      )}
    </footer>
  );
}

function ColumnAutomationViewBody({
  intakeSource,
  history,
  tab,
  graph,
  loading,
  error,
  onRetry,
  canvasId,
  runHrefFor,
  editHref,
  general,
  agent,
}: {
  history: ReturnType<typeof useFactoryAutomationNavigation>;
  intakeSource?: RunHistoryIntakeSource;
  tab: ColumnAutomationViewTab;
  graph?: IntakeAutomationGraph;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editHref?: string;
  general?: ReactNode;
  agent?: PlanningReviewAgentSlot;
}) {
  if (tab === "runs") {
    return (
      <FactoryAutomationHistory
        intakeSource={intakeSource}
        organizationId={history.organizationId}
        factoryId={history.factoryId}
        canvasId={canvasId}
        selectedRunId={history.selectedRunId}
        onSelectRun={history.setSelectedRunId}
        loading={loading}
        error={error}
        onRetry={onRetry}
      />
    );
  }
  if (tab === "general" && general) {
    return general;
  }
  if (tab === "agent" && agent) {
    return (
      <PlanningReviewEditor
        key={`${agent.draft?.components[0]?.id ?? "agent"}:${JSON.stringify(agent.draft?.components[0]?.configuration.steps ?? [])}`}
        initialDraft={agent.draft}
        onSave={agent.onSave}
        organizationId={agent.organizationId}
        factoryId={agent.factoryId}
        factoryKey={agent.factoryKey}
        automationId={agent.automationId}
        isLoading={agent.isLoading}
        showAutomationNote={false}
        showCancel={false}
      />
    );
  }
  return (
    <AutomationViewBody
      graph={graph}
      loading={loading}
      error={error}
      onRetry={onRetry}
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      editHref={editHref}
    />
  );
}

export function ColumnAutomationViewHost({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  canvasId,
  columnKey,
  title,
  onClose,
  general,
  initialTab,
  onDelete,
  deletePending,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  canvasId: string;
  columnKey?: string;
  title: string;
  onClose: () => void;
  general?: ReactNode;
  initialTab?: ColumnAutomationViewTab;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
}) {
  const automation = useIntakeAutomationCanvas(organizationId, canvasId);
  const agent = useColumnCanvasAgentEditor(organizationId, canvasId);
  const nodeIds = automation.graph?.specNodes?.map((node) => node.id) ?? [];
  const intakeSource = scheduledIntakeHistorySource(automation.graph);
  const generalSettings = useScheduledOwnerSettings({
    organizationId,
    factoryId,
    intakeSource,
    graph: automation.graph,
    saveNode: agent.saveNode,
    general,
  });

  if (columnAutomationUsesSplitConfig(nodeIds)) {
    return (
      <MergeConfidenceConfigModal
        title={automation.name?.trim() || title}
        graph={automation.graph}
        loading={automation.isLoading}
        error={automation.isError}
        onRetry={() => void automation.refetch()}
        canvasId={canvasId}
        runHrefFor={(runId) =>
          factoryAppRunPath(organizationId, factoryKey, canvasId, runId, { from: "lines", lineId })
        }
        onClose={onClose}
        factoryId={factoryId}
        factoryKey={factoryKey}
        onSaveNode={(update) => agent.saveNode(update, { notify: false })}
        onDelete={onDelete}
        deletePending={deletePending}
        initialNodeId={nodeIds.includes(IMPLEMENT_AGENT_NODE_ID) ? IMPLEMENT_AGENT_NODE_ID : undefined}
      />
    );
  }
  return (
    <ColumnAutomationViewPopup
      organizationId={organizationId}
      factoryId={factoryId}
      historyEligible={Boolean(intakeSource) || columnKey === "verify" || columnKey === "done"}
      intakeSource={intakeSource}
      title={automation.name?.trim() || title}
      graph={automation.graph}
      loading={automation.isLoading}
      error={automation.isError}
      onRetry={() => void automation.refetch()}
      editHref={factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "lines", lineId })}
      canvasId={canvasId}
      runHrefFor={(runId) => factoryAppRunPath(organizationId, factoryKey, canvasId, runId, { from: "lines", lineId })}
      onClose={onClose}
      general={generalSettings}
      agent={
        agent.agentNode
          ? {
              draft: agent.draft ?? undefined,
              isLoading: agent.isLoading || !agent.draft,
              organizationId,
              factoryId,
              factoryKey,
              automationId: canvasId,
              onSave: agent.save,
            }
          : undefined
      }
      initialTab={initialTab}
      onDelete={onDelete}
      deletePending={deletePending}
    />
  );
}

function scheduledIntakeHistorySource(graph: IntakeAutomationGraph): RunHistoryIntakeSource | undefined {
  const nodes = graph.specNodes ?? [];
  if (nodes.some((node) => node.id === "scheduled-agent") && nodes.some((node) => node.component === "schedule"))
    return "scheduled-intake";
  return undefined;
}

function AutomationViewBody({
  graph,
  loading,
  error,
  onRetry,
  canvasId,
  runHrefFor,
  editHref,
}: {
  graph?: IntakeAutomationGraph;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editHref?: string;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <section
        className="relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
        aria-label="Automation"
        data-testid="column-automation-view-canvas"
      >
        <p className="workspace-body-text text-muted-foreground">{viewEmptyMessage(loading, error)}</p>
        {error && onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {COLUMN_AUTOMATIONS_COPY.viewRetry}
          </Button>
        ) : null}
        {editHref ? (
          <SettingsAutomationCanvasEdit
            href={editHref}
            label={COLUMN_AUTOMATIONS_COPY.editLabel}
            testId="column-automation-view-edit"
          />
        ) : null}
      </section>
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="column-automation-view-canvas"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      workflowNodes={graph.specNodes}
      editHref={editHref}
      editLabel={COLUMN_AUTOMATIONS_COPY.editLabel}
      editTestId="column-automation-view-edit"
    />
  );
}

function viewEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return COLUMN_AUTOMATIONS_COPY.viewLoading;
  }
  return error ? COLUMN_AUTOMATIONS_COPY.viewError : COLUMN_AUTOMATIONS_COPY.viewEmpty;
}

function useScheduledOwnerSettings({
  organizationId,
  factoryId,
  intakeSource,
  graph,
  saveNode,
  general,
}: {
  organizationId: string;
  factoryId: string;
  intakeSource: RunHistoryIntakeSource | undefined;
  graph: IntakeAutomationGraph;
  saveNode: ReturnType<typeof useColumnCanvasAgentEditor>["saveNode"];
  general?: ReactNode;
}) {
  const layout = useOptionalFactoriesLayout();
  if (general) return general;
  const node = graph.specNodes?.find((node) => node.component === "createWorkOrder");
  if (!layout?.factory?.onboarding?.codexSubscription) return undefined;
  if (intakeSource !== "scheduled-intake" || !node) return undefined;
  return (
    <ScheduledIntakeOwnerSettings
      organizationId={organizationId}
      factoryId={factoryId}
      node={node}
      onSave={(update) => saveNode(update, { notify: false })}
    />
  );
}
