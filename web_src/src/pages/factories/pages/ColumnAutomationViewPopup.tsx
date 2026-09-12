import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useState, type ReactNode } from "react";

import {
  COLUMN_AUTOMATIONS_COPY,
  columnAutomationViewTabs,
  type ColumnAutomationViewTab,
} from "../lib/columnAutomations";
import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
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
}: ColumnAutomationViewPopupProps) {
  const hasGeneral = Boolean(general);
  const hasAgent = Boolean(agent);
  const tabs = columnAutomationViewTabs({ hasGeneral, hasAgent });
  const [userTab, setUserTab] = useState<ColumnAutomationViewTab | undefined>(() =>
    initialTab && tabs.includes(initialTab) ? initialTab : undefined,
  );
  const tab = userTab && tabs.includes(userTab) ? userTab : (tabs[0] ?? "automation");

  return (
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
    </PopupShell>
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
        key={agent.draft?.components[0]?.id ?? "agent"}
        initialDraft={agent.draft}
        onSave={agent.onSave}
        organizationId={agent.organizationId}
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
  factoryKey,
  lineId,
  canvasId,
  title,
  onClose,
  general,
  initialTab,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  canvasId: string;
  title: string;
  onClose: () => void;
  general?: ReactNode;
  initialTab?: ColumnAutomationViewTab;
}) {
  const automation = useIntakeAutomationCanvas(organizationId, canvasId);
  const agent = useColumnCanvasAgentEditor(organizationId, canvasId);
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
      general={general}
      agent={
        agent.agentNode
          ? {
              draft: agent.draft ?? undefined,
              isLoading: agent.isLoading || !agent.draft,
              organizationId,
              onSave: agent.save,
            }
          : undefined
      }
      initialTab={initialTab}
    />
  );
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
