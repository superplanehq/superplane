import { OverviewPage } from "../OverviewPage";
import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { GettingStartedWireframe } from "./GettingStartedWireframe";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/** Storybook overview: Get started checklist when tips are active, else real Overview. */
export function StorybookOverviewPage() {
  const { factoryId } = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();

  if (onboarding?.shouldShowOverviewTips(factoryId)) {
    return <GettingStartedWireframe />;
  }

  return <OverviewPage />;
}
