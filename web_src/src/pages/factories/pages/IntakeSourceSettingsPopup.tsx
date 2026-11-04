import type { Dispatch, ReactNode, SetStateAction } from "react";

import { IntakeAutomaticIntakeSetting } from "./IntakeAutomaticIntakeSetting";
import { IntakeSettingsHeader } from "./IntakeSettingsHeader";
import { IntakeSourceSettingsFooter } from "./IntakeSourceSettingsFooter";
import { FactoryAutomationHistory } from "./FactoryAutomationHistory";
import { useFactoryAutomationNavigation } from "@/hooks/useFactoryAutomationNavigation";
import { NodeConfigRepositoryLine } from "./NodeConfigPanel";
import {
  intakeSourceSettingsTitle,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { DependabotIntakeFilterFields } from "./DependabotIntakeFilterFields";
import { DatadogIntakeFilterFields } from "./DatadogIntakeFilterFields";
import { LinearIntakeFilterFields } from "./LinearIntakeFilterFields";
import { IntakeSourceDefaultOwner } from "./IntakeDefaultOwnerField";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { JiraIntakeFilterFields } from "./JiraIntakeFilterFields";
import { ProductiveIntakeFilterFields } from "./ProductiveIntakeFilterFields";
import { SentryIntakeFilterFields } from "./SentryIntakeFilterFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { PopupShell } from "./work-order-popup-redesign/popupShared";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import type { LinearFilterChoices } from "./linearIntakeScope";
import { useIntakeSettingsDraft } from "./useIntakeSettingsDraft";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  canEdit: boolean;
  accessLoading: boolean;
  sourceId?: LineIntakeSourceId;
  organizationId?: string;
  factoryId?: string;
  repository?: string;
  integrationId?: string;
  resourceId?: string;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  paused?: boolean;
  onPausedChange?: (paused: boolean) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  deletePending?: boolean;
  deleteError?: string;
  onDelete?: () => Promise<void> | void;
  canvasId?: string;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  page?: boolean;
  initialTab?: IntakeSettingsTab;
}

export function IntakeSourceSettingsPopup({
  settings,
  canEdit,
  accessLoading,
  sourceId = "github-issues",
  organizationId,
  factoryId,
  repository,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  paused = false,
  onPausedChange,
  savePending = false,
  saveError,
  deletePending = false,
  deleteError,
  onDelete,
  canvasId,
  agent,
  onClose,
  fixed = true,
  page = false,
  initialTab = "general",
}: IntakeSourceSettingsPopupProps) {
  const { draft, onDraftChange, getDraft, linearFilterChoices, setLinearFilterChoices } =
    useIntakeSettingsDraft(settings);
  const editableAgent = editableAgentSlot(agent, canEdit);
  const history = useFactoryAutomationNavigation({
    organizationId,
    factoryId,
    graph: automationGraph,
    agent: editableAgent,
    parameter: "settings",
    initialTab,
    hasAgent: Boolean(editableAgent),
  });

  const content = (
    <>
      <IntakeSettingsHeader
        name={settings.name}
        repository={repository}
        sourceId={sourceId}
        history={history}
        hasAgent={Boolean(editableAgent)}
        onClose={onClose}
      />
      {history.tab === "runs" ? (
        <FactoryAutomationHistory
          organizationId={history.organizationId}
          factoryId={history.factoryId}
          canvasId={canvasId}
          intakeSource={sourceId}
          selectedRunId={history.selectedRunId}
          onSelectRun={history.setSelectedRunId}
          loading={automationLoading}
          error={automationError}
          onRetry={onRetryAutomation}
        />
      ) : (
        <IntakeSettingsTabPanel
          automaticIntakeSetting={
            sourceId === "github-issues" && onPausedChange ? (
              <IntakeAutomaticIntakeSetting
                paused={paused}
                disabled={savePending || deletePending}
                onPausedChange={onPausedChange}
              />
            ) : null
          }
          repository={repository}
          tab={history.tab}
          sourceId={sourceId}
          organizationId={organizationId}
          integrationId={integrationId}
          resourceId={resourceId}
          labelOptions={labelOptions}
          labelOptionsLoading={labelOptionsLoading}
          draft={draft}
          linearFilterChoices={linearFilterChoices}
          onLinearFilterChoicesChange={setLinearFilterChoices}
          canEdit={canEdit}
          accessLoading={accessLoading}
          agent={editableAgent}
          savePending={savePending}
          saveError={saveError}
          deletePending={deletePending}
          deleteError={deleteError}
          onDelete={onDelete}
          onDraftChange={onDraftChange}
          getDraft={getDraft}
          onSave={onSave}
          onClose={onClose}
        />
      )}
    </>
  );

  if (page) {
    return (
      <section
        className="flex h-full min-h-0 flex-col overflow-hidden bg-background pt-[env(safe-area-inset-top)] dark:bg-gray-900"
        data-testid="intake-source-settings"
        aria-label={intakeSourceSettingsTitle(sourceId, settings.name)}
      >
        {content}
      </section>
    );
  }

  return (
    <PopupShell testId="intake-source-settings" canvas fixed={fixed} onDismiss={onClose}>
      {content}
    </PopupShell>
  );
}

function editableAgentSlot(agent: PlanningReviewAgentSlot | undefined, canEdit: boolean) {
  if (!canEdit) {
    return undefined;
  }
  return agent;
}

function IntakeSettingsTabPanel({
  automaticIntakeSetting,
  repository,
  tab,
  sourceId,
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  linearFilterChoices,
  onLinearFilterChoicesChange,
  canEdit,
  accessLoading,
  agent,
  savePending,
  saveError,
  deletePending,
  deleteError,
  onDelete,
  onDraftChange,
  getDraft,
  onSave,
  onClose,
}: IntakeGeneralTabProps & {
  tab: IntakeSettingsTab;
  agent?: PlanningReviewAgentSlot;
}) {
  if (tab === "agent" && agent) {
    return (
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
    );
  }
  return (
    <IntakeGeneralTab
      automaticIntakeSetting={automaticIntakeSetting}
      repository={repository}
      sourceId={sourceId}
      organizationId={organizationId}
      integrationId={integrationId}
      resourceId={resourceId}
      labelOptions={labelOptions}
      labelOptionsLoading={labelOptionsLoading}
      draft={draft}
      linearFilterChoices={linearFilterChoices}
      onLinearFilterChoicesChange={onLinearFilterChoicesChange}
      canEdit={canEdit}
      accessLoading={accessLoading}
      savePending={savePending}
      saveError={saveError}
      deletePending={deletePending}
      deleteError={deleteError}
      onDelete={onDelete}
      onDraftChange={onDraftChange}
      getDraft={getDraft}
      onSave={onSave}
      onClose={onClose}
    />
  );
}

type IntakeGeneralTabProps = {
  automaticIntakeSetting?: ReactNode;
  repository?: string;
  sourceId: LineIntakeSourceId;
  organizationId?: string;
  integrationId?: string;
  resourceId?: string;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  linearFilterChoices: LinearFilterChoices;
  onLinearFilterChoicesChange: (choices: LinearFilterChoices) => void;
  canEdit: boolean;
  accessLoading: boolean;
  savePending?: boolean;
  saveError?: string;
  deletePending: boolean;
  deleteError?: string;
  onDelete?: () => Promise<void> | void;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  getDraft: () => IntakeSourceSettings;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
};

function IntakeGeneralTab({
  automaticIntakeSetting,
  repository,
  sourceId,
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  linearFilterChoices,
  onLinearFilterChoicesChange,
  canEdit,
  accessLoading,
  savePending,
  saveError,
  deletePending,
  deleteError,
  onDelete,
  onDraftChange,
  getDraft,
  onSave,
  onClose,
}: IntakeGeneralTabProps) {
  const ownerField = (
    <IntakeSourceDefaultOwner
      sourceId={sourceId}
      organizationId={organizationId}
      draft={draft}
      disabled={savePending}
      onDraftChange={onDraftChange}
    />
  );
  const isGitHub = sourceId === "github-issues";
  const ownerAtBottom = intakeOwnerAfterFilters(sourceId);

  return (
    <>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          {!canEdit ? (
            <p
              role={accessLoading ? "status" : "alert"}
              className="rounded-md border border-border bg-muted p-3 text-sm"
            >
              {accessLoading
                ? "Checking access…"
                : "You do not have permission to change intake settings. Ask an organization admin for access."}
            </p>
          ) : null}
          <fieldset disabled={!canEdit} className="flex min-w-0 flex-col gap-6 border-0 p-0">
            {sourceId === "dependabot-alerts" && repository ? <NodeConfigRepositoryLine value={repository} /> : null}
            {isGitHub && automaticIntakeSetting ? (
              <div className="border-b border-border pb-4">{automaticIntakeSetting}</div>
            ) : null}
            {!ownerAtBottom ? ownerField : null}
            <IntakeFilterFields
              sourceId={sourceId}
              draft={draft}
              onDraftChange={onDraftChange}
              organizationId={organizationId}
              integrationId={integrationId}
              resourceId={resourceId}
              labelOptions={labelOptions}
              labelOptionsLoading={labelOptionsLoading}
              canEdit={canEdit}
              linearFilterChoices={linearFilterChoices}
              onLinearFilterChoicesChange={onLinearFilterChoicesChange}
            />
            {ownerAtBottom && organizationId ? <div className="border-t border-border pt-4">{ownerField}</div> : null}
          </fieldset>
        </div>
      </div>
      <IntakeSourceSettingsFooter
        sourceId={sourceId}
        draft={draft}
        linearLabelsRequired={
          sourceId === "linear-issues" && linearFilterChoices.labels && draft.linearLabels.length === 0
        }
        linearProjectsRequired={
          sourceId === "linear-issues" && linearFilterChoices.projects && draft.linearProjectIds.length === 0
        }
        canEdit={canEdit}
        savePending={savePending}
        saveError={saveError}
        deletePending={deletePending}
        deleteError={deleteError}
        onDelete={onDelete}
        getDraft={getDraft}
        onSave={onSave}
        onClose={onClose}
      />
    </>
  );
}

function intakeOwnerAfterFilters(sourceId: LineIntakeSourceId): boolean {
  return sourceId === "github-issues" || sourceId === "linear-issues";
}

function IntakeFilterFields({
  sourceId,
  draft,
  onDraftChange,
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  canEdit,
  linearFilterChoices,
  onLinearFilterChoicesChange,
}: Pick<
  IntakeGeneralTabProps,
  | "sourceId"
  | "draft"
  | "onDraftChange"
  | "organizationId"
  | "integrationId"
  | "resourceId"
  | "labelOptions"
  | "labelOptionsLoading"
  | "canEdit"
  | "linearFilterChoices"
  | "onLinearFilterChoicesChange"
>) {
  return (
    <>
      <GitHubIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        labelOptions={labelOptions}
        labelOptionsLoading={labelOptionsLoading}
        disabled={!canEdit}
      />
      <JiraIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        organizationId={organizationId}
        integrationId={integrationId}
        projectId={resourceId}
      />
      <SentryIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        organizationId={organizationId}
        integrationId={integrationId}
      />
      <DependabotIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={onDraftChange} />
      <ProductiveIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        organizationId={organizationId}
        integrationId={integrationId}
        projectId={resourceId}
      />
      <DatadogIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        organizationId={organizationId}
        integrationId={integrationId}
        resourceId={resourceId}
      />
      <LinearIntakeFilterFields
        sourceId={sourceId}
        settings={draft}
        onSettingsChange={onDraftChange}
        organizationId={organizationId}
        integrationId={integrationId}
        filterChoices={linearFilterChoices}
        onFilterChoicesChange={onLinearFilterChoicesChange}
      />
    </>
  );
}
