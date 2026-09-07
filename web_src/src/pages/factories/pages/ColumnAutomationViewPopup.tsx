import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useState, type ReactNode } from "react";

import {
  COLUMN_AUTOMATIONS_COPY,
  columnAutomationViewTabs,
  type ColumnAutomationViewTab,
} from "../lib/columnAutomations";
import { factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { SettingsAutomationCanvas } from "./SettingsAutomationCanvas";
import { useIntakeAutomationCanvas, type IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

interface ColumnAutomationViewPopupProps {
  title: string;
  graph?: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  editHref?: string;
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
        {tabs.length > 1 ? (
          <Tabs value={tab} onValueChange={(value) => setUserTab(value as ColumnAutomationViewTab)} className="mt-3">
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
        ) : null}
      </PopupHeader>
      {tab === "general" && general ? (
        general
      ) : tab === "agent" && agent ? (
        <PlanningReviewEditor
          key={agent.draft?.components[0]?.id ?? "agent"}
          initialDraft={agent.draft}
          onSave={agent.onSave}
          organizationId={agent.organizationId}
          isLoading={agent.isLoading}
          showAutomationNote={false}
          showCancel={false}
        />
      ) : (
        <AutomationViewBody
          title={title}
          graph={graph}
          loading={loading}
          error={error}
          onRetry={onRetry}
          editHref={editHref}
        />
      )}
    </PopupShell>
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
  title,
  graph,
  loading,
  error,
  onRetry,
  editHref,
}: {
  title: string;
  graph?: IntakeAutomationGraph;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  editHref?: string;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <section
        className="flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
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
          <Link href={editHref} className={buttonVariants({ size: "sm" })}>
            {COLUMN_AUTOMATIONS_COPY.editLabel}
          </Link>
        ) : null}
      </section>
    );
  }

  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Automation"
      data-testid="column-automation-view-canvas"
    >
      <div className="flex shrink-0 items-center justify-between gap-2 px-5 pt-3 pb-2">
        <p className="min-w-0 truncate text-[15px] font-semibold tracking-[-0.02em] text-foreground">{title}</p>
        {editHref ? (
          <Link href={editHref} className={buttonVariants({ size: "sm" })} data-testid="column-automation-view-edit">
            {COLUMN_AUTOMATIONS_COPY.editLabel}
          </Link>
        ) : null}
      </div>
      <div className="min-h-[18rem] flex-1">
        <SettingsAutomationCanvas graph={graph} />
      </div>
    </section>
  );
}

function viewEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return COLUMN_AUTOMATIONS_COPY.viewLoading;
  }
  return error ? COLUMN_AUTOMATIONS_COPY.viewError : COLUMN_AUTOMATIONS_COPY.viewEmpty;
}
