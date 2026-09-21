import { useDeleteFactoryIntake, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback, useMemo, useState } from "react";
import { useLocation } from "react-router";

import { factoryAppConfigurePath, factoryAppRunPath, jiraIntakeIntegrationIdFromSearch } from "../lib/factoryPagePaths";
import { factoryIntakeUpdateInput, intakeConnectionFromSource } from "./intakeConnectionModel";
import { IntakeConnectionDialogs } from "./IntakeConnectionDialogs";
import { intakeSettingsConnectionProps } from "./intakeSettingsConnectionProps";
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
import { useIntakeConnection } from "./useIntakeConnection";

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
  const { search } = useLocation();
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const agent = useColumnCanvasAgentEditor(organizationId, intake.appId);
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
  const deleteIntake = useDeleteFactoryIntake(organizationId, factoryId);
  const integrationId = intake.integrationId;
  const resourceId = intake.resourceId;
  const savedBinding = useMemo(
    () => intakeConnectionFromSource({ integrationId, resourceId }),
    [integrationId, resourceId],
  );
  const connection = useIntakeConnection({
    organizationId,
    factoryKey,
    lineId,
    intakeId: intake.intakeId,
    sourceId: intake.source.id,
    initialBinding: savedBinding,
    returnedIntegrationId: jiraIntakeIntegrationIdFromSearch(search),
  });
  const actions = useIntakeSettingsActions({
    intakeId: intake.intakeId,
    savedBinding,
    connectionBinding: connection.binding,
    connectionEnabled: connection.enabled,
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
    <>
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
        paused={intake.paused}
        pausePending={updateIntake.isPending}
        deletePending={deleteIntake.isPending}
        pauseError={actions.pauseError}
        deleteError={actions.deleteError}
        onPause={actions.pauseIntake}
        onResume={actions.resumeIntake}
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
                onSave: agent.save,
              }
            : undefined
        }
        onClose={onClose}
        initialTab={initialTab}
        connection={intakeSettingsConnectionProps(intake, connection, savedBinding)}
        fixed
      />
      <IntakeConnectionDialogs organizationId={organizationId} sourceId={intake.source.id} connection={connection} />
    </>
  );
}

function useIntakeSettingsActions({
  intakeId,
  savedBinding,
  connectionBinding,
  connectionEnabled,
  updateIntake,
  deleteIntake,
  automationRefetch,
  onClose,
}: {
  intakeId: string;
  savedBinding: ReturnType<typeof intakeConnectionFromSource>;
  connectionBinding: ReturnType<typeof intakeConnectionFromSource>;
  connectionEnabled: boolean;
  updateIntake: ReturnType<typeof useUpdateFactoryIntake>;
  deleteIntake: ReturnType<typeof useDeleteFactoryIntake>;
  automationRefetch: () => Promise<unknown>;
  onClose: () => void;
}) {
  const [pauseError, setPauseError] = useState<string>();
  const [deleteError, setDeleteError] = useState<string>();

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      await updateIntake.mutateAsync(
        factoryIntakeUpdateInput(
          intakeId,
          intakeSettingsToApi(next),
          savedBinding,
          connectionBinding,
          connectionEnabled,
        ),
      );
      await automationRefetch();
    },
    [automationRefetch, connectionBinding, connectionEnabled, intakeId, savedBinding, updateIntake],
  );

  const pauseIntake = useCallback(async () => {
    setPauseError(undefined);
    try {
      await updateIntake.mutateAsync({ intakeId, paused: true });
    } catch (error) {
      setPauseError(getApiErrorMessage(error, INTAKE_SETTINGS_COPY.pauseError));
      throw error;
    }
  }, [intakeId, updateIntake]);

  const resumeIntake = useCallback(async () => {
    setPauseError(undefined);
    try {
      await updateIntake.mutateAsync({ intakeId, paused: false });
    } catch (error) {
      setPauseError(getApiErrorMessage(error, INTAKE_SETTINGS_COPY.pauseError));
      throw error;
    }
  }, [intakeId, updateIntake]);

  const removeIntake = useCallback(async () => {
    setDeleteError(undefined);
    try {
      await deleteIntake.mutateAsync(intakeId);
      onClose();
    } catch (error) {
      setDeleteError(getApiErrorMessage(error, INTAKE_SETTINGS_COPY.deleteError));
      throw error;
    }
  }, [deleteIntake, intakeId, onClose]);

  return { pauseError, deleteError, saveSettings, pauseIntake, resumeIntake, removeIntake };
}
