import { useNavigate } from "react-router";

import type { FactoriesFactory } from "@/api-client";

import { firstFactoryLineId } from "../../lib/factoryPagePaths";
import type { OnboardingRepo } from "./onboardingMocks";
import { afterOnboardingPath } from "./useFinishOnboarding";
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
 * Picks the Finish action for the last setup step. The app provisions the
 * workspace. Provisioning creates a canvas and materializes a template, which
 * the Storybook fixture backend cannot serve, so stories mark the workspace
 * ready through the setup context and open the line board instead.
 */
export function useFinishSetupAction(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
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
    const lineId = firstFactoryLineId(args.factory) ?? args.factory?.onboarding?.provisionedLineId;
    if (!lineId) {
      return;
    }
    navigate(afterOnboardingPath({ organizationId: args.organizationId, factoryKey: args.factoryKey, lineId }), {
      replace: true,
    });
  };
}
