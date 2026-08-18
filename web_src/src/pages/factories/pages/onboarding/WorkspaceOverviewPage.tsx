import { useState } from "react";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { OverviewPage } from "../OverviewPage";
import { GettingStartedWireframe } from "./GettingStartedWireframe";
import { dismissWorkspaceGettingStarted, shouldShowWorkspaceGettingStarted } from "./gettingStartedState";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

export function WorkspaceOverviewPage() {
  const { organizationId, factoryId } = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();
  const [showGettingStarted, setShowGettingStarted] = useState(
    () => onboarding?.shouldShowOverviewTips(factoryId) ?? shouldShowWorkspaceGettingStarted(organizationId, factoryId),
  );

  if (!showGettingStarted) {
    return <OverviewPage />;
  }

  return (
    <GettingStartedWireframe
      onDismiss={() => {
        dismissWorkspaceGettingStarted(organizationId, factoryId);
        setShowGettingStarted(false);
      }}
    />
  );
}
