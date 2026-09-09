import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useEffect, useState, type Dispatch, type SetStateAction } from "react";

import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsTabs,
  normalizeIntakeSourceSettings,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { SettingsAutomationHeaderRow, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { LineIntakeSourceId } from "./lineIntakeModel";

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
  editAutomationHref?: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: IntakeSettingsTab;
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
  editAutomationHref,
  canvasId,
  runHrefFor,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
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
          editHref={tab === "automation" ? editAutomationHref : undefined}
          editLabel={INTAKE_SETTINGS_COPY.editAutomation}
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
        savePending={savePending}
        saveError={saveError}
        onDraftChange={setDraft}
        onSave={onSave}
        onClose={onClose}
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
  savePending,
  saveError,
  onDraftChange,
  onSave,
  onClose,
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
  savePending?: boolean;
  saveError?: string;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  if (tab === "automation") {
    return (
      <IntakeAutomationTab
        graph={automationGraph}
        canvasId={canvasId}
        runHrefFor={runHrefFor}
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
      onDraftChange={onDraftChange}
      onSave={onSave}
      onClose={onClose}
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
  onDraftChange,
  onSave,
  onClose,
}: {
  sourceId: LineIntakeSourceId;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <GitHubIntakeFilterFields
            sourceId={sourceId}
            settings={draft}
            onSettingsChange={onDraftChange}
            labelOptions={labelOptions}
            labelOptionsLoading={labelOptionsLoading}
          />
        </div>
      </div>
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        {saveError ? (
          <p className="workspace-body-text text-destructive" role="alert">
            {saveError}
          </p>
        ) : (
          <span />
        )}
        <Button
          type="button"
          disabled={savePending}
          onClick={async () => {
            try {
              await onSave(normalizeIntakeSourceSettings(draft));
              onClose();
            } catch {
              // The parent supplies the actionable error message.
            }
          }}
          data-testid="intake-source-settings-save"
        >
          {savePending ? INTAKE_SETTINGS_COPY.saving : INTAKE_SETTINGS_COPY.save}
        </Button>
      </footer>
    </>
  );
}

function IntakeAutomationTab({
  graph,
  canvasId,
  runHrefFor,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <IntakeAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        onRetry={automationRetry(error, onRetry)}
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

function IntakeAutomationEmpty({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <section
      className="flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {INTAKE_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
    </section>
  );
}
