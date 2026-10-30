import { useDeleteFactoryIntake, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback, useEffect, useMemo, useState } from "react";

import { dependabotRepositoryValue } from "@/lib/dependabotRepository";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
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
  return <SourceIntakeSettings key={props.intake.intakeId} {...props} />;
}

function SourceIntakeSettings({
  organizationId,
  factoryId,
  factoryKey,
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
  const saveError =
    actions.pauseError ??
    (updateIntake.error ? getApiErrorMessage(updateIntake.error, INTAKE_SETTINGS_COPY.saveError) : undefined);
  const watchedRepository =
    intake.source.id === "dependabot-alerts"
      ? dependabotRepositoryValue(
          automation.graph.specNodes?.find((node) => node.component === "github.onDependabotAlert"),
          repository,
        )
      : repository;

  return (
    <IntakeSourceSettingsPopup
      settings={intake.settings}
      paused={actions.paused}
      onPausedChange={intake.source.id === "github-issues" ? actions.changePaused : undefined}
      sourceId={intake.source.id}
      organizationId={organizationId}
      factoryId={factoryId}
      repository={watchedRepository}
      integrationId={intake.integrationId}
      resourceId={intake.resourceId}
      labelOptions={labelOptions}
      labelOptionsLoading={repositoryLabels.isLoading}
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={automation.refetch}
      onSave={actions.saveSettings}
      savePending={updateIntake.isPending}
      saveError={saveError}
      deletePending={deleteIntake.isPending}
      deleteError={actions.deleteError}
      onDelete={actions.removeIntake}
      canvasId={intake.appId}
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
  const [pauseError, setPauseError] = useState<string>();
  const [paused, setPaused] = useState(intake.paused);
  useEffect(() => setPaused(intake.paused), [intake.intakeId, intake.paused]);
  const changePaused = useCallback(
    async (next: boolean) => {
      setPauseError(undefined);
      try {
        await updateIntake.mutateAsync({ intakeId: intake.intakeId, paused: next });
        setPaused(next);
      } catch (error) {
        setPauseError(getApiErrorMessage(error, "SuperPlane could not update the intake. Try again."));
        throw error;
      }
    },
    [intake.intakeId, updateIntake],
  );

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      setPauseError(undefined);
      const normalized = normalizeIntakeSourceSettings(next, intake.source.id);
      await updateIntake.mutateAsync({
        intakeId: intake.intakeId,
        settings: intakeSettingsToApi(normalized, intake.source.id),
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

  return { deleteError, saveSettings, removeIntake, paused, changePaused, pauseError };
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
  // An intake with a team saves its projects as settings. Only an older
  // intake that has projects and no team still binds to them.
  if (intake.source.id !== "linear-issues" || next.linearTeamId) {
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
