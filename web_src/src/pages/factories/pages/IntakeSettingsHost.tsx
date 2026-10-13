import type { SuperplaneComponentsNode } from "@/api-client";
import { useDeleteFactoryIntake, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback, useEffect, useMemo, useState } from "react";

import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomationsCopy";
import { DependabotIntakeConfigBody } from "./DependabotIntakeConfigBody";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import { type SplitConfigDeleteCopy } from "./splitConfigDeleteCopy";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsToApi,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";

interface IntakeSettingsHostProps {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  intake: ConfiguredLineIntakeSource;
  /** Repository the intake watches, used to offer the labels that exist in it. */
  repository?: string;
  /** GitHub integration that has access to that repository. */
  vcsIntegrationId?: string;
  initialTab?: IntakeSettingsTab;
  page?: boolean;
  onClose: () => void;
}

/** Settings for one intake, opened from its row at the foot of Backlog. */
export function IntakeSettingsHost(props: IntakeSettingsHostProps) {
  if (props.intake.source.id === "dependabot-alerts") {
    return <DependabotIntakeSettings {...props} />;
  }
  return <SourceIntakeSettings {...props} />;
}

function DependabotIntakeSettings({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  intake,
  repository,
  page = false,
  onClose,
}: IntakeSettingsHostProps) {
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const agent = useColumnCanvasAgentEditor(organizationId, intake.appId);
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
  const deleteIntake = useDeleteFactoryIntake(organizationId, factoryId);
  const actions = useIntakeSettingsActions({
    intake,
    updateIntake,
    deleteIntake,
    automationRefetch: automation.refetch,
    onClose,
  });
  const [severityDraft, setSeverityDraft] = useState(intake.settings);
  const saveError = updateIntake.error
    ? getApiErrorMessage(updateIntake.error, INTAKE_SETTINGS_COPY.saveError)
    : undefined;

  useEffect(() => {
    setSeverityDraft(intake.settings);
  }, [intake.settings]);

  return (
    <MergeConfidenceConfigModal
      title={intake.settings.name.trim() || intake.source.name}
      graph={automation.graph}
      loading={automation.isLoading}
      error={automation.isError}
      onRetry={() => void automation.refetch()}
      canvasId={intake.appId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      runHrefFor={intakeRunHref(organizationId, factoryKey, intake.appId, lineId)}
      onClose={onClose}
      onSaveNode={(update) => agent.saveNode(update, { notify: false })}
      onDelete={actions.removeIntake}
      deletePending={deleteIntake.isPending}
      deleteError={actions.deleteError}
      deleteCopy={DEPENDABOT_INTAKE_DELETE_COPY}
      page={page}
      showRunsTab={Boolean(intake.appId)}
      fallbackNode={DEPENDABOT_SETTINGS_NODE}
      renderNodeBody={(node) =>
        node.component === "github.onDependabotAlert" ? (
          <DependabotIntakeConfigBody
            node={node}
            fallbackRepository={repository}
            settings={severityDraft}
            savePending={updateIntake.isPending}
            saveError={saveError}
            onSettingsChange={setSeverityDraft}
            onSave={actions.saveSettings}
          />
        ) : null
      }
    />
  );
}

function SourceIntakeSettings({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  intake,
  repository,
  vcsIntegrationId,
  initialTab = "general",
  page = false,
  onClose,
}: IntakeSettingsHostProps) {
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const agent = useColumnCanvasAgentEditor(organizationId, intake.appId);
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
  const deleteIntake = useDeleteFactoryIntake(organizationId, factoryId);
  const actions = useIntakeSettingsActions({
    intake,
    updateIntake,
    deleteIntake,
    automationRefetch: automation.refetch,
    onClose,
  });
  // An empty integration id keeps the query idle, so we never ask for labels
  // without knowing the repository they belong to.
  const repositoryLabels = useIntegrationResources(
    organizationId,
    repository ? (vcsIntegrationId ?? "") : "",
    "label",
    repository ? { repository } : undefined,
  );
  const labelOptions = useMemo(
    () => (repositoryLabels.data ?? []).map((resource) => resource.name ?? "").filter((name) => name.length > 0),
    [repositoryLabels.data],
  );
  const editAutomationHref = intake.appId
    ? factoryAppConfigurePath(organizationId, factoryKey, intake.appId, { from: "lines", lineId })
    : undefined;
  const saveError = updateIntake.error
    ? getApiErrorMessage(updateIntake.error, INTAKE_SETTINGS_COPY.saveError)
    : undefined;

  return (
    <IntakeSourceSettingsPopup
      settings={intake.settings}
      sourceId={intake.source.id}
      organizationId={organizationId}
      integrationId={intake.integrationId}
      resourceId={intake.resourceId}
      labelOptions={labelOptions}
      labelOptionsLoading={repositoryLabels.isLoading}
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={automation.refetch}
      onSave={actions.saveSettings}
      savePending={updateIntake.isPending || automation.isLoading}
      saveError={saveError}
      deletePending={deleteIntake.isPending}
      deleteError={actions.deleteError}
      onDelete={actions.removeIntake}
      editAutomationHref={editAutomationHref}
      canvasId={intake.appId}
      runHrefFor={intakeRunHref(organizationId, factoryKey, intake.appId, lineId)}
      agent={
        agent.agentNode
          ? {
              draft: agent.draft ?? undefined,
              isLoading: agent.isLoading || !agent.draft,
              organizationId,
              factoryId,
              factoryKey,
              automationId: intake.appId,
              onSave: agent.save,
            }
          : undefined
      }
      onClose={onClose}
      initialTab={initialTab}
      page={page}
      fixed
    />
  );
}

const DEPENDABOT_SETTINGS_NODE: SuperplaneComponentsNode = {
  id: "dependabot-alerts-settings",
  name: "On Dependabot Alert",
  type: "TYPE_TRIGGER",
  component: "github.onDependabotAlert",
};

const DEPENDABOT_INTAKE_DELETE_COPY: SplitConfigDeleteCopy = {
  menuLabel: INTAKE_SETTINGS_COPY.delete,
  title: INTAKE_SETTINGS_COPY.deleteTitle,
  description: INTAKE_SETTINGS_COPY.deleteDescription,
  confirmLabel: INTAKE_SETTINGS_COPY.deleteConfirm,
  pendingLabel: COLUMN_AUTOMATIONS_COPY.deletingLabel,
  keepLabel: INTAKE_SETTINGS_COPY.deleteCancel,
};

function intakeRunHref(organizationId: string, factoryKey: string, appId: string | undefined, lineId?: string) {
  if (!appId) {
    return undefined;
  }
  return (runId: string) => factoryAppRunPath(organizationId, factoryKey, appId, runId, { from: "lines", lineId });
}

function useIntakeSettingsActions({
  intake,
  updateIntake,
  deleteIntake,
  automationRefetch,
  onClose,
}: {
  intake: ConfiguredLineIntakeSource;
  updateIntake: ReturnType<typeof useUpdateFactoryIntake>;
  deleteIntake: ReturnType<typeof useDeleteFactoryIntake>;
  automationRefetch: () => Promise<unknown>;
  onClose: () => void;
}) {
  const [deleteError, setDeleteError] = useState<string>();

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      await updateIntake.mutateAsync({
        intakeId: intake.intakeId,
        settings: intakeSettingsToApi(next),
        ...datadogServiceRebind(intake, next),
        ...linearProjectRebind(intake, next),
        ...sentryProjectRebind(intake, next),
      });
      await automationRefetch();
    },
    [automationRefetch, intake, updateIntake],
  );

  const removeIntake = useCallback(async () => {
    setDeleteError(undefined);
    try {
      await deleteIntake.mutateAsync(intake.intakeId);
      onClose();
    } catch (error) {
      setDeleteError(getApiErrorMessage(error, INTAKE_SETTINGS_COPY.deleteError));
      throw error;
    }
  }, [deleteIntake, intake.intakeId, onClose]);

  return { deleteError, saveSettings, removeIntake };
}

function datadogServiceRebind(
  intake: ConfiguredLineIntakeSource,
  next: IntakeSourceSettings,
): { integrationId?: string; resourceId?: string } {
  if (intake.source.id !== "datadog") {
    return {};
  }
  const service = next.datadogService.trim();
  const current = intake.resourceId?.trim() ?? "";
  if (!service || service === current || !intake.integrationId) {
    return {};
  }
  return { integrationId: intake.integrationId, resourceId: service };
}

function linearProjectRebind(
  intake: ConfiguredLineIntakeSource,
  next: IntakeSourceSettings,
): { integrationId?: string; resourceId?: string } {
  if (intake.source.id !== "linear-issues") {
    return {};
  }
  const projects = next.linearProjectIds.join(",");
  const current = intake.resourceId?.trim() ?? "";
  if (!projects || projects === current || !intake.integrationId) {
    return {};
  }
  return { integrationId: intake.integrationId, resourceId: projects };
}

function sentryProjectRebind(
  intake: ConfiguredLineIntakeSource,
  next: IntakeSourceSettings,
): { integrationId?: string; resourceId?: string } {
  if (intake.source.id !== "sentry-exceptions") {
    return {};
  }
  const projects = next.sentryProjectIds.join(",");
  const current = intake.resourceId?.trim() ?? "";
  if (!projects || projects === current || !intake.integrationId) {
    return {};
  }
  return { integrationId: intake.integrationId, resourceId: projects };
}
