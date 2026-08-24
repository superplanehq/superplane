import { useNavigate } from "react-router";

import { factoryHomePath } from "../../lib/factoryPagePaths";
import type { OnboardingRepo } from "./onboardingMocks";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function storybookRepos(setup: OnboardingSetupApi): OnboardingRepo[] {
  if (!setup.selectedRepo || !setup.vcsHost) {
    return [];
  }
  const [org, name] = setup.selectedRepo.split("/");
  if (!org || !name) {
    return [];
  }
  return [{ id: `${setup.vcsHost}-${org}-${name}`, name, org, provider: setup.vcsHost }];
}

/**
 * Picks the Start action for the last setup step. The app provisions the
 * workspace. Provisioning creates a canvas and materializes a template, which
 * the Storybook fixture backend cannot serve, so stories mark the workspace
 * ready through the setup context and open the line board instead.
 */
export function useFinishSetupAction(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  setup: OnboardingSetupApi;
  finish: () => void | Promise<void>;
}): () => void | Promise<void> {
  const onboarding = useOnboardingStorybook();
  const navigate = useNavigate();

  if (!onboarding) {
    return args.finish;
  }

  return () => {
    onboarding.completeOnboarding(args.factoryId, storybookRepos(args.setup));
    navigate(factoryHomePath(args.organizationId, args.factoryKey), { replace: true });
  };
}
