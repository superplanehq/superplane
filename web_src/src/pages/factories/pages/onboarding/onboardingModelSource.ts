import { useCallback, useState } from "react";

import type { OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";

const STORAGE_PREFIX = "superplane:onboarding-model-source";

function storageKey(factoryId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}`;
}

function isModelSource(value: string | null): value is OnboardingAgentCredentialChoice {
  return value === "own-key" || value === "hosted";
}

export function readOnboardingModelSource(factoryId: string): OnboardingAgentCredentialChoice | null {
  if (!factoryId) return null;
  const value = localStorage.getItem(storageKey(factoryId));
  return isModelSource(value) ? value : null;
}

export function writeOnboardingModelSource(factoryId: string, source: OnboardingAgentCredentialChoice): void {
  if (!factoryId) return;
  localStorage.setItem(storageKey(factoryId), source);
}

/** Saved per workspace, so the choice stays after a provider connect redirect. */
export function useOnboardingModelSource(factoryId: string) {
  const [modelSource, setModelSourceState] = useState(() => readOnboardingModelSource(factoryId));
  const setModelSource = useCallback(
    (source: OnboardingAgentCredentialChoice) => {
      writeOnboardingModelSource(factoryId, source);
      setModelSourceState(source);
    },
    [factoryId],
  );
  return [modelSource, setModelSource] as const;
}
