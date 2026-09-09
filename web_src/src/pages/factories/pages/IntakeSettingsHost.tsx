import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback } from "react";

import { factoryAppConfigurePath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import { intakeSettingsToApi, type IntakeSettingsTab, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";

interface IntakeSettingsHostProps {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  intake: ConfiguredLineIntakeSource;
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
  initialTab = "general",
  onClose,
}: IntakeSettingsHostProps) {
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const agent = useColumnCanvasAgentEditor(organizationId, intake.appId);
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
  const editAutomationHref = intake.appId
    ? factoryAppConfigurePath(organizationId, factoryKey, intake.appId, { from: "lines", lineId })
    : undefined;

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      await updateIntake.mutateAsync({
        intakeId: intake.intakeId,
        name: next.name,
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
      automationGraph={automation.graph}
      automationLoading={automation.isLoading}
      automationError={automation.isError}
      onRetryAutomation={automation.refetch}
      onSave={saveSettings}
      savePending={updateIntake.isPending || automation.isLoading}
      saveError={
        updateIntake.error
          ? getApiErrorMessage(updateIntake.error, "SuperPlane could not save the intake settings. Try again.")
          : undefined
      }
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
      fixed
    />
  );
}
