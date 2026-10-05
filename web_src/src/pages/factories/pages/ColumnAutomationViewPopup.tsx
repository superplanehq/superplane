import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_RISK_SCORE } from "@/lib/experimentalFeatures";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useState, type ReactNode } from "react";

import { ColumnAutomationFooterSlotProvider } from "./columnAutomationFooterSlot";

import {
  COLUMN_AUTOMATIONS_COPY,
  columnAutomationViewTabs,
  type ColumnAutomationViewTab,
} from "../lib/columnAutomations";
import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { mergeConfidenceFromDraft } from "./mergeConfidenceChecks";
import { MergeConfidenceSettingsForm } from "./MergeConfidenceSettingsForm";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { riskScoreCategoriesFromDraft } from "./riskScoreCategories";
import { RiskScoreSettingsForm } from "./RiskScoreSettingsForm";
import {
  SettingsAutomationCanvasEdit,
  SettingsAutomationHeaderRow,
  SettingsAutomationWorkspace,
} from "./SettingsAutomationWorkspace";
import { useIntakeAutomationCanvas, type IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

interface ColumnAutomationViewPopupProps {
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
  const tabs = columnAutomationViewTabs({ hasGeneral, hasAgent });
  const [userTab, setUserTab] = useState<ColumnAutomationViewTab | undefined>(() =>
    initialTab && tabs.includes(initialTab) ? initialTab : undefined,
  );
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [footerSlot, setFooterSlot] = useState<HTMLDivElement | null>(null);
  const tab = userTab && tabs.includes(userTab) ? userTab : (tabs[0] ?? "automation");
  const footerSlotValue = onDelete ? (confirmDelete ? null : footerSlot) : undefined;

  return (
    <ColumnAutomationFooterSlotProvider slot={footerSlotValue}>
      <PopupShell testId="column-automation-view" canvas fixed onDismiss={onClose}>
        <PopupHeader title={title} onClose={onClose}>
          <SettingsAutomationHeaderRow
            tabs={<ColumnAutomationViewTabs tabs={tabs} tab={tab} onTabChange={setUserTab} />}
          />
        </PopupHeader>
        <ColumnAutomationViewBody
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

function ColumnAutomationViewTabs({
  tabs,
  tab,
  onTabChange,
}: {
  tabs: ColumnAutomationViewTab[];
  tab: ColumnAutomationViewTab;
  onTabChange: (tab: ColumnAutomationViewTab) => void;
}) {
  if (tabs.length <= 1) {
    return null;
  }
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as ColumnAutomationViewTab)}>
      <TabsList aria-label={COLUMN_AUTOMATIONS_COPY.tabsLabel}>
        {tabs.includes("general") ? (
          <TabsTrigger value="general" data-testid="column-automation-view-tab-general">
            <Settings />
            {COLUMN_AUTOMATIONS_COPY.generalTab}
          </TabsTrigger>
        ) : null}
        {tabs.includes("agent") ? (
          <TabsTrigger value="agent" data-testid="column-automation-view-tab-agent">
            <Bot />
            {COLUMN_AUTOMATIONS_COPY.agentTab}
          </TabsTrigger>
        ) : null}
        <TabsTrigger value="automation" data-testid="column-automation-view-tab-automation">
          <Workflow />
          {COLUMN_AUTOMATIONS_COPY.automationTab}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

function ColumnAutomationViewBody({
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
        showVisualEvidenceSetting={agent.showVisualEvidenceSetting}
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
  title: string;
  onClose: () => void;
  general?: ReactNode;
  initialTab?: ColumnAutomationViewTab;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
}) {
  const automation = useIntakeAutomationCanvas(organizationId, canvasId);
  const agent = useColumnCanvasAgentEditor(organizationId, canvasId);
  const allowRiskScore = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_RISK_SCORE);
  const isRiskScoreCanvas = automation.graph?.specNodes?.some((node) => node.id === "on-pr-risk") ?? false;
  const riskScoreGeneral = mergeConfidenceGeneral({
    allow: allowRiskScore,
    isCanvas: isRiskScoreCanvas,
    draft: agent.draft,
    onSave: agent.save,
  });
  return (
    <ColumnAutomationViewPopup
      title={automation.name?.trim() || title}
      graph={automation.graph}
      loading={automation.isLoading}
      error={automation.isError}
      onRetry={() => void automation.refetch()}
      editHref={factoryAppConfigurePath(organizationId, factoryKey, canvasId, { from: "lines", lineId })}
      canvasId={canvasId}
      runHrefFor={(runId) => factoryAppRunPath(organizationId, factoryKey, canvasId, runId, { from: "lines", lineId })}
      onClose={onClose}
      general={general ?? riskScoreGeneral}
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
              showVisualEvidenceSetting: agent.showVisualEvidenceSetting,
            }
          : undefined
      }
      initialTab={initialTab}
      onDelete={onDelete}
      deletePending={deletePending}
    />
  );
}

function mergeConfidenceGeneral({
  allow,
  isCanvas,
  draft,
  onSave,
}: {
  allow: boolean;
  isCanvas: boolean;
  draft: PlanningReviewDraft | null | undefined;
  onSave: (next: PlanningReviewDraft) => Promise<void> | void;
}): ReactNode {
  if (!allow || !isCanvas || !draft) {
    return undefined;
  }
  if (mergeConfidenceFromDraft(draft)) {
    return <MergeConfidenceSettingsForm draft={draft} onSave={onSave} />;
  }
  if (riskScoreCategoriesFromDraft(draft)) {
    return <RiskScoreSettingsForm draft={draft} onSave={onSave} />;
  }
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
