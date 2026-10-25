import { useEffect, useRef, useState } from "react";

import { PRFeedbackSettingsFooter } from "./PRFeedbackSettingsChrome";
import { PRFeedbackChecksFields, PRFeedbackDiscussionFields } from "./PRFeedbackSettingsFields";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { getApiErrorMessage } from "@/lib/errors";
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
  const [editedAgent, setEditedAgent] = useState<PlanningReviewDraft>();
  const [saving, setSaving] = useState(false);
  const [localSaveError, setLocalSaveError] = useState<string>();
  const agentDraft = editedAgent ?? agent?.draft;
  const agentUnavailable = Boolean(agent && (agent.isLoading || !agentDraft?.components.length));
  const busy = saving || savePending;

  const save = async (next: PRFeedbackDraftSettings) => {
    if (busy || agentUnavailable) return;
    setSaving(true);
    setLocalSaveError(undefined);
    try {
      if (editedAgent) {
        if (!agent?.onSave) throw new Error(PR_FEEDBACK_SETTINGS_COPY.saveError);
        await agent.onSave(editedAgent);
      }
      await onSave(next);
    } catch (error) {
      setLocalSaveError(getApiErrorMessage(error, PR_FEEDBACK_SETTINGS_COPY.saveError));
      throw error;
    } finally {
      setSaving(false);
    }
  };
  const history = useFactoryAutomationNavigation({
    organizationId,
    factoryId,
    graph: automationGraph,
    agent,
    parameter: "prFeedbackSettings",
    initialTab,
    hasAgent: false,
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
          hasAgent={false}
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
      ) : (
        <PRFeedbackGeneralTab
          organizationId={organizationId}
          githubIntegrationId={githubIntegrationId}
          vcsProvider={vcsProvider}
          draft={draft}
          confirmDelete={confirmDelete}
          agent={agent}
          agentDraft={agentDraft}
          onAgentChange={setEditedAgent}
          saveDisabled={agentUnavailable}
          savePending={busy}
          deletePending={deletePending}
          saveError={localSaveError || saveError}
          onUpdate={(key, value) => update(key, value)}
          onSave={save}
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
  confirmDelete,
  agent,
  agentDraft,
  onAgentChange,
  saveDisabled,
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
  confirmDelete: boolean;
  agent?: PlanningReviewAgentSlot;
  agentDraft?: PlanningReviewDraft;
  onAgentChange: (next: PlanningReviewDraft) => void;
  saveDisabled: boolean;
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
      <div inert={savePending || undefined} className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          {agent?.isLoading ? (
            <p role="status" className="text-sm text-muted-foreground">
              Loading model settings...
            </p>
          ) : agentDraft?.components.length ? (
            <PlanningReviewForm
              draft={agentDraft}
              onChange={onAgentChange}
              organizationId={agent?.organizationId}
              factoryId={agent?.factoryId}
              appearance="fields"
              view="pr-feedback"
              showResources={false}
              showConcurrency={false}
              disabled={savePending}
            />
          ) : agent ? (
            <p role="alert" className="text-sm text-destructive">
              Could not load model settings.
            </p>
          ) : null}
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
        saveDisabled={saveDisabled}
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
