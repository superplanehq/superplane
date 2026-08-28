import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { useCallback } from "react";

import { factoryAppConfigurePath } from "../lib/factoryPagePaths";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
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
  initialTab = "general",
  onOpenRun,
  onClose,
}: IntakeSettingsHostProps) {
  const automation = useIntakeAutomationCanvas(organizationId, intake.appId);
  const runs = useIntakeAutomationRuns(organizationId, factoryId, intake);
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
      onClose={onClose}
      initialTab={initialTab}
      fixed
    />
  );
}
