import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PlanningHealthSection, PlanningSettingsFields } from "./PlanningSettingsFields";
import {
  SettingsAutomationCanvasEdit,
  SettingsAutomationHeaderRow,
  SettingsAutomationWorkspace,
} from "./SettingsAutomationWorkspace";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PLANNING_SETTINGS_COPY,
  planningSettingsTabs,
  type PlanningDraftSettings,
  type PlanningSettingsTab,
} from "./planningSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface PlanningSettingsPopupProps {
  settings: PlanningDraftSettings;
  title?: string;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: PlanningDraftSettings) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  editAutomationHref?: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PlanningSettingsTab;
}

export function PlanningSettingsPopup({
  settings,
  title = PLANNING_SETTINGS_COPY.title,
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
}: PlanningSettingsPopupProps) {
  const tabs = planningSettingsTabs(Boolean(agent));
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<PlanningSettingsTab>(() => (tabs.includes(initialTab) ? initialTab : "general"));
  const hasAgent = Boolean(agent);

  useEffect(() => {
    const next = planningSettingsTabs(hasAgent);
    if (!next.includes(tab)) {
      setTab("general");
    }
  }, [tab, hasAgent]);

  // Reset the draft only when the stored values change. Hosts often build a
  // new settings object on every render, and that alone must not erase
  // toggles the user changed before Save.
  useEffect(() => {
    setDraft({ enabled: settings.enabled, clarity: settings.clarity, confidence: settings.confidence });
  }, [settings.enabled, settings.clarity, settings.confidence]);

  const update = useCallback(<K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => {
    setDraft((current) => ({ ...current, [key]: value }));
  }, []);

  return (
    <PopupShell testId="planning-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={title} onClose={onClose}>
        <SettingsAutomationHeaderRow
          tabs={
            <Tabs value={tab} onValueChange={(value) => setTab(value as PlanningSettingsTab)}>
              <TabsList aria-label={PLANNING_SETTINGS_COPY.tabsLabel}>
                <TabsTrigger value="general" data-testid="planning-settings-tab-general">
                  <Settings />
                  {PLANNING_SETTINGS_COPY.generalTab}
                </TabsTrigger>
                {tabs.includes("agent") ? (
                  <TabsTrigger value="agent" data-testid="planning-settings-tab-agent">
                    <Bot />
                    {PLANNING_SETTINGS_COPY.agentTab}
                  </TabsTrigger>
                ) : null}
                <TabsTrigger value="automation" data-testid="planning-settings-tab-automation">
                  <Workflow />
                  {PLANNING_SETTINGS_COPY.automationTab}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          }
        />
      </PopupHeader>
      {tab === "automation" ? (
        <PlanningAutomationTab
          graph={automationGraph}
          canvasId={canvasId}
          runHrefFor={runHrefFor}
          editHref={editAutomationHref}
          loading={automationLoading}
          error={automationError}
          onRetry={onRetryAutomation}
        />
      ) : tab === "agent" && agent ? (
        <PlanningReviewEditor
          key={agent.draft?.components[0]?.id ?? "agent"}
          initialDraft={agent.draft}
          onSave={agent.onSave}
          organizationId={agent.organizationId}
          isLoading={agent.isLoading}
          showAutomationNote={false}
          showCancel={false}
          showVisualEvidenceSetting={agent.showVisualEvidenceSetting}
        />
      ) : (
        <PlanningGeneralTab
          draft={draft}
          savePending={savePending}
          saveError={saveError}
          onUpdate={update}
          onSave={onSave}
          onClose={onClose}
        />
      )}
    </PopupShell>
  );
}

function PlanningGeneralTab({
  draft,
  savePending,
  saveError,
  onUpdate,
  onSave,
  onClose,
}: {
  draft: PlanningDraftSettings;
  savePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => void;
  onSave: (next: PlanningDraftSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <PlanningHealthSection enabled={draft.enabled} />
          <PlanningSettingsFields draft={draft} onUpdate={onUpdate} />
        </div>
      </div>
      <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
        {saveError ? (
          <p className="workspace-body-text mr-auto text-destructive" role="alert">
            {saveError}
          </p>
        ) : null}
        <Button
          type="button"
          disabled={savePending}
          onClick={async () => {
            try {
              await onSave(draft);
              onClose();
            } catch {
              // The parent supplies the actionable error message.
            }
          }}
          data-testid="planning-settings-save"
        >
          {savePending ? PLANNING_SETTINGS_COPY.saving : PLANNING_SETTINGS_COPY.save}
        </Button>
      </footer>
    </>
  );
}

function PlanningAutomationTab({
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
      <section
        className="relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
        aria-label="Automation"
        data-testid="planning-automation"
      >
        <p className="workspace-body-text text-muted-foreground">
          {loading
            ? PLANNING_SETTINGS_COPY.automationLoading
            : error
              ? PLANNING_SETTINGS_COPY.automationError
              : PLANNING_SETTINGS_COPY.automationEmpty}
        </p>
        {error && onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {PLANNING_SETTINGS_COPY.retry}
          </Button>
        ) : null}
        {editHref ? (
          <SettingsAutomationCanvasEdit href={editHref} label="Edit automation" testId="settings-automation-edit" />
        ) : null}
      </section>
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="planning-automation"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      workflowNodes={graph.specNodes}
      editHref={editHref}
      editLabel="Edit automation"
    />
  );
}
