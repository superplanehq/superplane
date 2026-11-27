import { useCallback } from "react";

import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { showErrorToast } from "@/lib/toast";

import { intakeSettingsToApi } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";

export function useIntakeWebhookRetry(organizationId: string, factoryId: string) {
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);

  const retryIntake = useCallback(
    (intake: ConfiguredLineIntakeSource) => {
      updateIntake.mutate(
        {
          intakeId: intake.intakeId,
          settings: intakeSettingsToApi(intake.settings, intake.source.id),
        },
        {
          onError: () => showErrorToast("SuperPlane could not retry the webhook setup. Try again."),
        },
      );
    },
    [updateIntake],
  );

  return { retryIntake, retrying: updateIntake.isPending };
}
