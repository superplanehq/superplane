import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";

import { PRFeedbackAutomationTab, PRFeedbackSettingsFooter } from "./PRFeedbackSettingsChrome";
import {
  PRFeedbackChecksFields,
  PRFeedbackDiscussionFields,
  PRFeedbackHealthSection,
  PRFeedbackTextField,
} from "./PRFeedbackSettingsFields";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  appendUniqueTrimmedString,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface PRFeedbackSettingsPopupProps {
  organizationId?: string;
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
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PRFeedbackSettingsTab;
}

export function PRFeedbackSettingsPopup({
  organizationId,
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
  onClose,
  fixed = true,
  initialTab = "general",
}: PRFeedbackSettingsPopupProps) {
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<PRFeedbackSettingsTab>(initialTab);
  const [confirmDelete, setConfirmDelete] = useState(false);

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
      ) : (
        <PRFeedbackGeneralTab
          organizationId={organizationId}
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
  const [checkNameInput, setCheckNameInput] = useState("");
  const addCheckName = () => {
    onUpdate("checkNames", appendUniqueTrimmedString(draft.checkNames, checkNameInput));
    setCheckNameInput("");
  };

  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <PRFeedbackHealthSection healthy={healthy} checks={checks} />
          <PRFeedbackTextField
            id="pr-feedback-name"
            label={PR_FEEDBACK_SETTINGS_COPY.nameLabel}
            helper={PR_FEEDBACK_SETTINGS_COPY.nameHelper}
            value={draft.name}
            onChange={(value) => onUpdate("name", value)}
          />
          <PRFeedbackTextField
            id="pr-feedback-repository"
            label={PR_FEEDBACK_SETTINGS_COPY.repositoryLabel}
            helper={
              checks ? PR_FEEDBACK_SETTINGS_COPY.checksRepositoryHelper : PR_FEEDBACK_SETTINGS_COPY.repositoryHelper
            }
            value={draft.repository}
            onChange={(value) => onUpdate("repository", value)}
          />
          {checks ? (
            <PRFeedbackChecksFields
              organizationId={organizationId}
              draft={draft}
              checkNameInput={checkNameInput}
              onUpdate={onUpdate}
              onInputChange={setCheckNameInput}
              onAdd={addCheckName}
            />
          ) : (
            <PRFeedbackDiscussionFields draft={draft} onUpdate={onUpdate} />
          )}
        </div>
      </div>
      <PRFeedbackSettingsFooter
        draft={draft}
        pendingCheckName={checkNameInput}
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
