import { Navigate, Outlet, useLocation } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryOnboardingPath, factoryOverviewPath } from "../../lib/factoryPagePaths";
import { isFactoryOnboardingComplete } from "./onboardingStatus";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/**
 * Keeps incomplete workspaces on setup while other workspaces stay browsable.
 * Storybook can override the server-backed state with its onboarding context.
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();

  const storybookPending = onboarding?.pending;
  const isOnboardingRoute = location.pathname.endsWith("/onboarding");
  const isIncomplete = onboarding ? storybookPending?.workspaceId === factoryId : !isFactoryOnboardingComplete(factory);

  if (!isIncomplete) {
    if (isOnboardingRoute) {
      return <Navigate to={factoryOverviewPath(organizationId, factoryKey)} replace />;
    }
    return <Outlet />;
  }

  if (isOnboardingRoute) {
    return <Outlet />;
  }

  return <Navigate to={factoryOnboardingPath(organizationId, factoryKey)} replace />;
}
