import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";

import { type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { FactoryAutomationHistory } from "./FactoryAutomationHistory";
import { useFactoryAutomationNavigation } from "@/hooks/useFactoryAutomationNavigation";
import { PlanningSettingsFooter } from "./PlanningSettingsConfiguration";
import { PlanningSettingsMenu } from "./PlanningSettingsMenu";
import { PlanningSettingsConfiguration } from "./PlanningSettingsConfiguration";
import { usePlanningSettingsDraft } from "./usePlanningSettingsDraft";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PLANNING_SETTINGS_COPY,
  type PlanningAutoStartLine,
  type PlanningDraftSettings,
  type PlanningSettingsTab,
} from "./planningSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface PlanningSettingsPopupProps {
  organizationId?: string;
  factoryId?: string;
  settings: PlanningDraftSettings;
  lines?: PlanningAutoStartLine[];
  title?: string;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: PlanningDraftSettings) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  canvasId?: string;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PlanningSettingsTab;
}

export function PlanningSettingsPopup({
  organizationId,
  factoryId,
  settings,
  lines = [],
  title = PLANNING_SETTINGS_COPY.title,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  savePending = false,
  saveError,
  canvasId,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
}: PlanningSettingsPopupProps) {
  const editor = usePlanningSettingsDraft({ settings, agent, savePending, onSave, onClose });
  const history = useFactoryAutomationNavigation({
    organizationId,
    factoryId,
    graph: automationGraph,
    agent,
    parameter: "planningSettings",
    initialTab,
    hasAgent: false,
  });
  const { tab, setTab } = history;

  return (
    <PopupShell testId="planning-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader
        title={title}
        onClose={onClose}
        actions={
          tab === "general" ? (
            <PlanningSettingsMenu
              enabled={editor.draft.enabled}
              disabled={editor.busy}
              onEnabledChange={(enabled) => void editor.setPlanningEnabled(enabled)}
            />
          ) : undefined
        }
      >
        <FactoryAutomationSettingsTabs
          tab={tab}
          onTabChange={setTab}
          hasAgent={false}
          generalLabel="Configuration"
          historyEnabled={history.enabled}
          label={PLANNING_SETTINGS_COPY.tabsLabel}
          testId="planning-settings"
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
        <PlanningSettingsConfiguration
          draft={editor.draft}
          lines={lines}
          agent={agent}
          agentDraft={editor.agentDraft}
          onAgentChange={editor.setEditedAgent}
          onUpdate={editor.update}
          onRetry={onRetryAutomation}
          saving={editor.busy}
        />
      )}
      {tab === "general" ? (
        <PlanningSettingsFooter
          error={editor.saveError || saveError}
          saving={editor.busy}
          disabled={editor.saveDisabled}
          onSave={() => void editor.save()}
        />
      ) : null}
    </PopupShell>
  );
}
