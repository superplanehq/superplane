import { useUpdateFactory } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import { PLANNING_SETTINGS_COPY, planningSettingsFromFactory, planningSettingsToApi } from "./planningSettingsModel";
import type { PlanningSetupStep } from "./planningSetupCaption";

export function usePlanningSetup(
  organizationId: string,
  factoryId: string,
  factory: Parameters<typeof planningSettingsFromFactory>[0],
) {
  const initial = planningSettingsFromFactory(factory);
  const [step, setStep] = useState<PlanningSetupStep>("refine");
  const [enabled, setEnabled] = useState(initial.enabled);
  const [clarity, setClarity] = useState(initial.clarity);
  const [confidence, setConfidence] = useState(initial.confidence);
  const [error, setError] = useState<string>();
  const updateFactory = useUpdateFactory(organizationId, factoryId);

  async function finish(): Promise<boolean> {
    setError(undefined);
    try {
      await updateFactory.mutateAsync({
        planning: planningSettingsToApi({ enabled, clarity, confidence }),
      });
      return true;
    } catch (cause) {
      setError(getApiErrorMessage(cause, PLANNING_SETTINGS_COPY.wizardFinishError));
      return false;
    }
  }

  return {
    step,
    setStep,
    enabled,
    setEnabled,
    clarity,
    setClarity,
    confidence,
    setConfidence,
    error,
    saving: updateFactory.isPending,
    finish,
  };
}

export type PlanningSetupModel = ReturnType<typeof usePlanningSetup>;
