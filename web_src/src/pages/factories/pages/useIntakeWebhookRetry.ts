import { useCallback, useEffect, useRef } from "react";

import { useFactoryIntakes, useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { showErrorToast } from "@/lib/toast";

import { intakeSettingsToApi } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";

const WEBHOOK_RETRY_POLL_INTERVAL_MS = 3000;
const WEBHOOK_RETRY_POLL_ATTEMPTS = 20;

/**
 * Retries webhook registration for a failed intake. Registration runs in the
 * background, so health is polled until the setup window ends. A slow failure
 * then shows on the board without a reload.
 */
export function useIntakeWebhookRetry(organizationId: string, factoryId: string) {
  const updateIntake = useUpdateFactoryIntake(organizationId, factoryId);
  const { refetch: refetchIntakes } = useFactoryIntakes(organizationId, factoryId);
  const pollTimer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(pollTimer.current), []);

  const retryIntake = useCallback(
    (intake: ConfiguredLineIntakeSource) => {
      updateIntake.mutate(
        {
          intakeId: intake.intakeId,
          settings: intakeSettingsToApi(intake.settings, intake.source.id),
        },
        {
          onSuccess: () => {
            const poll = (attemptsLeft: number) => {
              if (attemptsLeft === 0) {
                return;
              }
              pollTimer.current = window.setTimeout(() => {
                void refetchIntakes();
                poll(attemptsLeft - 1);
              }, WEBHOOK_RETRY_POLL_INTERVAL_MS);
            };
            window.clearTimeout(pollTimer.current);
            poll(WEBHOOK_RETRY_POLL_ATTEMPTS);
          },
          onError: () => showErrorToast("SuperPlane could not retry the webhook setup. Try again."),
        },
      );
    },
    [refetchIntakes, updateIntake],
  );

  return { retryIntake, retrying: updateIntake.isPending };
}
