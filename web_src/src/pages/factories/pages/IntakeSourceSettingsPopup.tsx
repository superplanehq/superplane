import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useEffect, useState, type Dispatch, type SetStateAction } from "react";

import { IntakeConnectionFields, type IntakeConnectionFieldsProps } from "./IntakeConnectionFields";
import { IntakeSourceSettingsFooter } from "./IntakeSourceSettingsFooter";
import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsTabs,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { JiraIntakeFilterFields } from "./JiraIntakeFilterFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import {
  SettingsAutomationCanvasEdit,
  SettingsAutomationHeaderRow,
  SettingsAutomationWorkspace,
} from "./SettingsAutomationWorkspace";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeSettingsConnection = Omit<IntakeConnectionFieldsProps, "sourceId"> & {
  saveDisabled?: boolean;
};

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  sourceId?: LineIntakeSourceId;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  paused?: boolean;
  pausePending?: boolean;
  deletePending?: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  editAutomationHref?: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: IntakeSettingsTab;
  connection?: IntakeSettingsConnection;
}

export function IntakeSourceSettingsPopup({
  settings,
  sourceId = "github-issues",
  labelOptions,
  labelOptionsLoading,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  savePending = false,
  saveError,
  paused = false,
  pausePending = false,
  deletePending = false,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
  editAutomationHref,
  canvasId,
  runHrefFor,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
  connection,
}: IntakeSourceSettingsPopupProps) {
  const tabs = intakeSettingsTabs(Boolean(agent));
  const hasAgent = Boolean(agent);
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<IntakeSettingsTab>(() => (tabs.includes(initialTab) ? initialTab : "general"));

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  useEffect(() => {
    const next = intakeSettingsTabs(hasAgent);
    if (!next.includes(tab)) {
      setTab("general");
    }
  }, [tab, hasAgent]);

  return (
    <PopupShell testId="intake-source-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={`Intake ${settings.name}`} onClose={onClose}>
        <SettingsAutomationHeaderRow
          tabs={
            <Tabs value={tab} onValueChange={(value) => setTab(value as IntakeSettingsTab)}>
              <TabsList aria-label={INTAKE_SETTINGS_COPY.tabsLabel}>
                <TabsTrigger value="general" data-testid="intake-settings-tab-general">
                  <Settings />
                  {INTAKE_SETTINGS_COPY.generalTab}
                </TabsTrigger>
                {tabs.includes("agent") ? (
                  <TabsTrigger value="agent" data-testid="intake-settings-tab-agent">
                    <Bot />
                    {INTAKE_SETTINGS_COPY.agentTab}
                  </TabsTrigger>
                ) : null}
                <TabsTrigger value="automation" data-testid="intake-settings-tab-automation">
                  <Workflow />
                  {INTAKE_SETTINGS_COPY.automationTab}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          }
        />
      </PopupHeader>
      <IntakeSettingsTabPanel
        tab={tab}
        sourceId={sourceId}
        labelOptions={labelOptions}
        labelOptionsLoading={labelOptionsLoading}
        draft={draft}
        agent={agent}
        automationGraph={automationGraph}
        automationLoading={automationLoading}
        automationError={automationError}
        onRetryAutomation={onRetryAutomation}
        canvasId={canvasId}
        runHrefFor={runHrefFor}
        editAutomationHref={editAutomationHref}
        savePending={savePending}
        saveError={saveError}
        paused={paused}
        pausePending={pausePending}
        deletePending={deletePending}
        pauseError={pauseError}
        deleteError={deleteError}
        onPause={onPause}
        onResume={onResume}
        onDelete={onDelete}
        onDraftChange={setDraft}
        onSave={onSave}
        onClose={onClose}
        connection={connection}
      />
    </PopupShell>
  );
}

function IntakeSettingsTabPanel({
  tab,
  sourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  agent,
  automationGraph,
  automationLoading,
  automationError,
  onRetryAutomation,
  canvasId,
  runHrefFor,
  editAutomationHref,
  savePending,
  saveError,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
  onDraftChange,
  onSave,
  onClose,
  connection,
}: {
  tab: IntakeSettingsTab;
  sourceId: LineIntakeSourceId;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  agent?: PlanningReviewAgentSlot;
  automationGraph?: IntakeAutomationGraph;
  automationLoading: boolean;
  automationError: boolean;
  onRetryAutomation?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editAutomationHref?: string;
  savePending?: boolean;
  saveError?: string;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
  connection?: IntakeSettingsConnection;
}) {
  if (tab === "automation") {
    return (
      <IntakeAutomationTab
        graph={automationGraph}
        canvasId={canvasId}
        runHrefFor={runHrefFor}
        editHref={editAutomationHref}
        loading={automationLoading}
        error={automationError}
        onRetry={onRetryAutomation}
      />
    );
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
    <IntakeGeneralTab
      sourceId={sourceId}
      labelOptions={labelOptions}
      labelOptionsLoading={labelOptionsLoading}
      draft={draft}
      savePending={savePending}
      saveError={saveError}
      paused={paused}
      pausePending={pausePending}
      deletePending={deletePending}
      pauseError={pauseError}
      deleteError={deleteError}
      onPause={onPause}
      onResume={onResume}
      onDelete={onDelete}
      onDraftChange={onDraftChange}
      onSave={onSave}
      onClose={onClose}
      connection={connection}
    />
  );
}

function IntakeGeneralTab({
  sourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  savePending,
  saveError,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
  onDraftChange,
  onSave,
  onClose,
  connection,
}: {
  sourceId: LineIntakeSourceId;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
  connection?: IntakeSettingsConnection;
}) {
  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          {connection ? (
            <IntakeConnectionFields
              sourceId={sourceId}
              health={connection.health}
              binding={connection.binding}
              integrations={connection.integrations}
              integrationsLoading={connection.integrationsLoading}
              projects={connection.projects}
              projectsLoading={connection.projectsLoading}
              projectsError={connection.projectsError}
              connecting={connection.connecting}
              connectError={connection.connectError}
              onBindingChange={connection.onBindingChange}
              onConnect={connection.onConnect}
              onReconnect={connection.onReconnect}
              onRetryProjects={connection.onRetryProjects}
            />
          ) : null}
          <GitHubIntakeFilterFields
            sourceId={sourceId}
            settings={draft}
            onSettingsChange={onDraftChange}
            labelOptions={labelOptions}
            labelOptionsLoading={labelOptionsLoading}
          />
          <JiraIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={onDraftChange} />
        </div>
      </div>
      <IntakeSourceSettingsFooter
        sourceId={sourceId}
        draft={draft}
        savePending={savePending}
        saveError={saveError}
        paused={paused}
        pausePending={pausePending}
        deletePending={deletePending}
        pauseError={pauseError}
        deleteError={deleteError}
        onPause={onPause}
        onResume={onResume}
        onDelete={onDelete}
        onSave={onSave}
        onClose={onClose}
        saveDisabled={connection?.saveDisabled}
      />
    </>
  );
}

function IntakeAutomationTab({
  graph,
  canvasId,
  runHrefFor,
  editHref,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editHref?: string;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <IntakeAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        onRetry={automationRetry(error, onRetry)}
        editHref={editHref}
      />
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="intake-source-automation"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      workflowNodes={graph.specNodes}
      editHref={editHref}
      editLabel={INTAKE_SETTINGS_COPY.editAutomation}
    />
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return INTAKE_SETTINGS_COPY.automationLoading;
  }
  return error ? INTAKE_SETTINGS_COPY.automationError : INTAKE_SETTINGS_COPY.automationEmpty;
}

function automationRetry(error: boolean, onRetry: (() => void) | undefined): (() => void) | undefined {
  return error ? onRetry : undefined;
}

function IntakeAutomationEmpty({
  message,
  onRetry,
  editHref,
}: {
  message: string;
  onRetry?: () => void;
  editHref?: string;
}) {
  return (
    <section
      className="relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {INTAKE_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
      {editHref ? (
        <SettingsAutomationCanvasEdit
          href={editHref}
          label={INTAKE_SETTINGS_COPY.editAutomation}
          testId="settings-automation-edit"
        />
      ) : null}
    </section>
  );
}
