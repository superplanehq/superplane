import { useState } from "react";
import { FirstRunSetup } from "./FirstRunSetup";
import type { OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";
import { useOnboardingSetupState } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

export function StatefulSetup({ model }: { model: OnboardingPageModel }) {
  const [choice, setChoice] = useState<OnboardingAgentCredentialChoice | null>(null);
  return <FirstRunSetup model={{ ...model, agentCredentialChoice: choice, setAgentCredentialChoice: setChoice }} />;
}

export function NavigableSetup({ model }: { model: OnboardingPageModel }) {
  const setup = useOnboardingSetupState("Payments Service", {
    simulateDiscovery: false,
    connected: new Set(["github"]),
  });
  return <StatefulSetup model={{ ...model, setup }} />;
}
