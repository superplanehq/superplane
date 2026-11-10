import { renderHook } from "@testing-library/react";

import type { OnboardingPageModel } from "./useFirstRunSetupFlow";
import { pageModel } from "./firstRunSetup.test-support";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";

export { pageModel };

export function setupState(): OnboardingSetupApi {
  const { result } = renderHook(() =>
    useOnboardingSetupState("Payments Service", {
      simulateDiscovery: false,
      connected: new Set(["github"]),
      initial: { vcsHost: "github", issuesChoice: "vcs", selectedRepo: "acme/api" },
    }),
  );
  return result.current;
}

export function hostedVcsModel(overrides: Partial<OnboardingPageModel> = {}) {
  return pageModel({
    hostedAgentReady: true,
    hostedModelsAvailable: true,
    setup: { ...setupState(), issuesChoice: "vcs" },
    ...overrides,
  });
}
