import { useEffect, useRef, useState } from "react";

import { PRFeedbackSettingsFooter } from "./PRFeedbackSettingsChrome";
import {
  PRFeedbackChecksFields,
  PRFeedbackDiscussionFields,
  PRFeedbackHealthSection,
} from "./PRFeedbackSettingsFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";
import { FactoryAutomationHistory } from "./FactoryAutomationHistory";
import { useFactoryAutomationNavigation } from "@/hooks/useFactoryAutomationNavigation";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface PRFeedbackSettingsPopupProps {
  organizationId?: string;
  factoryId?: string;
  githubIntegrationId?: string;
  vcsProvider?: string;
  settings: PRFeedbackDraftSettings;
  healthy: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  canvasId?: string;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PRFeedbackSettingsTab;
}

export function PRFeedbackSettingsPopup({
  organizationId,
  factoryId,
  githubIntegrationId,
  vcsProvider,
  settings,
  healthy,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  onDelete,
  savePending = false,
  deletePending = false,
  saveError,
  canvasId,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
}: PRFeedbackSettingsPopupProps) {
  const [draft, setDraft] = useState(settings);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const history = useFactoryAutomationNavigation({
    organizationId,
    factoryId,
    graph: automationGraph,
    agent,
    parameter: "prFeedbackSettings",
    initialTab,
    hasAgent: Boolean(agent),
  });
  const { tab, setTab } = history;
  const settingsKey = JSON.stringify(settings);
  const syncedSettingsKey = useRef(settingsKey);
  useEffect(() => {
    if (syncedSettingsKey.current === settingsKey) return;
    syncedSettingsKey.current = settingsKey;
    setDraft(settings);
  }, [settings, settingsKey]);

  function update<K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <PopupShell testId="pr-feedback-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={settings.name} onClose={onClose}>
        <FactoryAutomationSettingsTabs
          tab={tab}
          onTabChange={setTab}
          hasAgent={Boolean(agent)}
          historyEnabled={history.enabled}
          label={PR_FEEDBACK_SETTINGS_COPY.tabsLabel}
          testId="pr-feedback-settings"
        />
      </PopupHeader>
      {tab === "runs" ? (
        <FactoryAutomationHistory
          organizationId={history.organizationId}
          factoryId={history.factoryId}
          canvasId={canvasId}
          selectedRunId={history.selectedRunId}
          onSelectRun={history.setSelectedRunId}
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
          factoryId={agent.factoryId}
          factoryKey={agent.factoryKey}
          automationId={agent.automationId}
          isLoading={agent.isLoading}
          showAutomationNote={false}
          showCancel={false}
        />
      ) : (
        <PRFeedbackGeneralTab
          organizationId={organizationId}
          githubIntegrationId={githubIntegrationId}
          vcsProvider={vcsProvider}
          draft={draft}
          healthy={healthy}
          confirmDelete={confirmDelete}
          savePending={savePending}
          deletePending={deletePending}
          saveError={saveError}
          onUpdate={(key, value) => update(key, value)}
          onSave={onSave}
          onDelete={onDelete}
          onConfirmDelete={(next) => setConfirmDelete(next)}
          onClose={onClose}
        />
      )}
    </PopupShell>
  );
}

function PRFeedbackGeneralTab({
  organizationId,
  githubIntegrationId,
  vcsProvider,
  draft,
  healthy,
  confirmDelete,
  savePending,
  deletePending,
  saveError,
  onUpdate,
  onSave,
  onDelete,
  onConfirmDelete,
  onClose,
}: {
  organizationId?: string;
  githubIntegrationId?: string;
  vcsProvider?: string;
  draft: PRFeedbackDraftSettings;
  healthy: boolean;
  confirmDelete: boolean;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onClose: () => void;
}) {
  const checks = draft.source === "checks";

  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <PRFeedbackHealthSection healthy={healthy} checks={checks} />
          {checks ? (
            <PRFeedbackChecksFields
              organizationId={organizationId}
              githubIntegrationId={githubIntegrationId}
              draft={draft}
              onUpdate={onUpdate}
            />
          ) : (
            <PRFeedbackDiscussionFields
              organizationId={organizationId}
              githubIntegrationId={githubIntegrationId}
              vcsProvider={vcsProvider}
              draft={draft}
              onUpdate={onUpdate}
            />
          )}
        </div>
      </div>
      <PRFeedbackSettingsFooter
        draft={draft}
        confirmDelete={confirmDelete}
        savePending={savePending}
        deletePending={deletePending}
        saveError={saveError}
        onSave={onSave}
        onDelete={onDelete}
        onConfirmDelete={onConfirmDelete}
        onClose={onClose}
      />
    </>
  );
}
