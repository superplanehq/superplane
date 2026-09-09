import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback, useMemo } from "react";

import { factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import {
  intakeSettingsToApi,
  type IntakeAutomationRun,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";
import { useIntakeAutomationRuns } from "./useIntakeAutomationRuns";

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
  onOpenRun: (run: IntakeAutomationRun) => void;
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
  onOpenRun,
  onClose,
}: IntakeSettingsHostProps) {
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const agent = useColumnCanvasAgentEditor(organizationId, intake.appId);
  const runs = useIntakeAutomationRuns(organizationId, factoryId, intake);
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
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

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      await updateIntake.mutateAsync({
        intakeId: intake.intakeId,
        settings: intakeSettingsToApi(next),
      });
      await automation.refetch();
    },
    [automation, intake.intakeId, updateIntake],
  );

  return (
    <IntakeSourceSettingsPopup
      settings={intake.settings}
      sourceId={intake.source.id}
      labelOptions={labelOptions}
      labelOptionsLoading={repositoryLabels.isLoading}
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={automation.refetch}
      runs={runs.runs}
      runsLoading={runs.isLoading}
      runsError={runs.isError}
      onRetryRuns={runs.retry}
      onSave={saveSettings}
      savePending={updateIntake.isPending || automation.isLoading}
      saveError={
        updateIntake.error
          ? getApiErrorMessage(updateIntake.error, "SuperPlane could not save the intake settings. Try again.")
          : undefined
      }
      onOpenRun={onOpenRun}
      editAutomationHref={editAutomationHref}
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
      fixed
    />
  );
}
