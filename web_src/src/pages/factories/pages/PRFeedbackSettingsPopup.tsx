import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";

import { PRFeedbackAutomationTab, PRFeedbackSettingsFooter } from "./PRFeedbackSettingsChrome";
import {
  PRFeedbackChecksFields,
  PRFeedbackDiscussionFields,
  PRFeedbackHealthSection,
} from "./PRFeedbackSettingsFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  prFeedbackSettingsTabs,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface PRFeedbackSettingsPopupProps {
  organizationId?: string;
  factoryId?: string;
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
  editAutomationHref?: string;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PRFeedbackSettingsTab;
}

export function PRFeedbackSettingsPopup({
  organizationId,
  factoryId,
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
  editAutomationHref,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
}: PRFeedbackSettingsPopupProps) {
  const tabs = prFeedbackSettingsTabs(Boolean(agent));
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<PRFeedbackSettingsTab>(() => (tabs.includes(initialTab) ? initialTab : "general"));
  const [confirmDelete, setConfirmDelete] = useState(false);
  const hasAgent = Boolean(agent);

  useEffect(() => {
    const next = prFeedbackSettingsTabs(hasAgent);
    if (!next.includes(tab)) {
      setTab("general");
    }
  }, [tab, hasAgent]);

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  function update<K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <PopupShell testId="pr-feedback-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={settings.name} onClose={onClose}>
        <Tabs value={tab} onValueChange={(value) => setTab(value as PRFeedbackSettingsTab)} className="mt-3">
          <TabsList aria-label={PR_FEEDBACK_SETTINGS_COPY.tabsLabel}>
            <TabsTrigger value="general" data-testid="pr-feedback-settings-tab-general">
              <Settings />
              {PR_FEEDBACK_SETTINGS_COPY.generalTab}
            </TabsTrigger>
            {tabs.includes("agent") ? (
              <TabsTrigger value="agent" data-testid="pr-feedback-settings-tab-agent">
                <Bot />
                {PR_FEEDBACK_SETTINGS_COPY.agentTab}
              </TabsTrigger>
            ) : null}
            <TabsTrigger value="automation" data-testid="pr-feedback-settings-tab-automation">
              <Workflow />
              {PR_FEEDBACK_SETTINGS_COPY.automationTab}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </PopupHeader>
      {tab === "automation" ? (
        <PRFeedbackAutomationTab
          graph={automationGraph}
          title={settings.name}
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
        />
      ) : (
        <PRFeedbackGeneralTab
          organizationId={organizationId}
          factoryId={factoryId}
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
  factoryId,
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
  factoryId?: string;
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
              factoryId={factoryId}
              draft={draft}
              onUpdate={onUpdate}
            />
          ) : (
            <PRFeedbackDiscussionFields
              organizationId={organizationId}
              factoryId={factoryId}
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
