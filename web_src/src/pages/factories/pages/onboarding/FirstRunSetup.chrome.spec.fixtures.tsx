import { renderHook } from "@testing-library/react";
import { pageModel } from "./firstRunSetup.test-support";

import type { OnboardingPageModel } from "./useFirstRunSetupFlow";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";

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

export { pageModel } from "./firstRunSetup.test-support";

export function hostedVcsModel(overrides: Partial<OnboardingPageModel> = {}) {
  return pageModel({
    hostedAgentReady: true,
    hostedModelsAvailable: true,
    setup: { ...setupState(), issuesChoice: "vcs" },
    ...overrides,
  });
}
