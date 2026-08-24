import { Navigate, Outlet, useLocation } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryHomePath, factorySetupPath } from "../../lib/factoryPagePaths";
import { isFactoryOnboardingComplete } from "./onboardingStatus";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function isWorkspaceSetupRoute(pathname: string) {
  return pathname.endsWith("/setup") || pathname.endsWith("/onboarding");
}

/**
 * Keeps incomplete workspaces on setup while other workspaces stay browsable.
 * Storybook can override the server-backed state with its setup context.
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();

  const storybookPending = onboarding?.pending;
  const isSetupRoute = isWorkspaceSetupRoute(location.pathname);
  const isIncomplete = onboarding ? storybookPending?.workspaceId === factoryId : !isFactoryOnboardingComplete(factory);

  if (!isIncomplete) {
    if (isSetupRoute) {
      return <Navigate to={factoryHomePath(organizationId, factoryKey)} replace />;
    }
    return <Outlet />;
  }

  if (isSetupRoute) {
    return <Outlet />;
  }

  return <Navigate to={factorySetupPath(organizationId, factoryKey)} replace />;
}
