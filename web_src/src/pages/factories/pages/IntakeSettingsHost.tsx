import { useDeleteFactoryIntake, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback, useMemo, useState } from "react";

import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
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
  onClose: () => void;
}

/** Settings for one intake, opened from its row at the foot of Backlog. */
export function IntakeSettingsHost({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  intake,
  repository,
  vcsIntegrationId,
  initialTab = "general",
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
      saveError={
        updateIntake.error
          ? getApiErrorMessage(updateIntake.error, "SuperPlane could not save the intake settings. Try again.")
          : undefined
      }
      deletePending={deleteIntake.isPending}
      deleteError={actions.deleteError}
      onDelete={actions.removeIntake}
      editAutomationHref={editAutomationHref}
      canvasId={intake.appId}
      runHrefFor={
        intake.appId
          ? (runId) => factoryAppRunPath(organizationId, factoryKey, intake.appId, runId, { from: "lines", lineId })
          : undefined
      }
      agent={
        agent.agentNode
          ? {
              draft: agent.draft ?? undefined,
              isLoading: agent.isLoading || !agent.draft,
              organizationId,
              factoryId,
              factoryKey,
              onSave: agent.save,
            }
          : undefined
      }
      onClose={onClose}
      initialTab={initialTab}
      fixed
    />
  );
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
