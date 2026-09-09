import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useEffect, useState, type Dispatch, type SetStateAction } from "react";

import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsTabs,
  normalizeIntakeSourceSettings,
  type IntakeListenMode,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { SettingsAutomationHeaderRow, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { LineIntakeSourceId } from "./lineIntakeModel";

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  sourceId?: LineIntakeSourceId;
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

  const update = <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

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
        onUpdate={update}
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
  onUpdate,
  onDraftChange,
  onSave,
  onClose,
}: {
  tab: IntakeSettingsTab;
  sourceId: LineIntakeSourceId;
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
  onUpdate: <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => void;
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
      draft={draft}
      savePending={savePending}
      saveError={saveError}
      onUpdate={onUpdate}
      onDraftChange={onDraftChange}
      onSave={onSave}
      onClose={onClose}
    />
  );
}

function IntakeGeneralTab({
  sourceId,
  draft,
  savePending,
  saveError,
  onUpdate,
  onDraftChange,
  onSave,
  onClose,
}: {
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => void;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <section>
            <Label htmlFor="intake-source-name">{INTAKE_SETTINGS_COPY.nameLabel}</Label>
            <p className="workspace-body-text mt-1 text-muted-foreground">{INTAKE_SETTINGS_COPY.nameHelper}</p>
            <Input
              id="intake-source-name"
              className="mt-2"
              value={draft.name}
              onChange={(event) => onUpdate("name", event.target.value)}
              data-testid="intake-source-name"
            />
          </section>

          <fieldset className="min-w-0">
            <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
              {INTAKE_SETTINGS_COPY.listenLabel}
            </legend>
            <div className="mt-2 flex flex-col gap-2">
              <IntakeSettingsRadioOption
                name="intake-listen-mode"
                value="listen"
                checked={draft.listenMode === "listen"}
                title={INTAKE_SETTINGS_COPY.listenOption}
                helper={INTAKE_SETTINGS_COPY.listenHelper}
                onChange={() => onUpdate("listenMode", "listen" satisfies IntakeListenMode)}
              />
              <IntakeSettingsRadioOption
                name="intake-listen-mode"
                value="schedule"
                checked={draft.listenMode === "schedule"}
                title={INTAKE_SETTINGS_COPY.scheduleOption}
                helper={INTAKE_SETTINGS_COPY.scheduleHelper}
                disabled
                onChange={() => onUpdate("listenMode", "schedule" satisfies IntakeListenMode)}
              />
            </div>
          </fieldset>

          <GitHubIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={onDraftChange} />
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
