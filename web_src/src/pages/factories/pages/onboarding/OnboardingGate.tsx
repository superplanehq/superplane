import { Navigate, Outlet, useLocation } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryOnboardingPath } from "../../lib/factoryPagePaths";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/**
 * Storybook-only: while setup is pending for the current workspace, keep the
 * user on that workspace's setup route. Other workspaces stay browsable.
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId, factoryId, factoryKey } = useFactoriesLayout();

  const pending = onboarding?.pending;
  if (!pending || pending.workspaceId !== factoryId) {
    return <Outlet />;
  }

  if (location.pathname.endsWith("/onboarding")) {
    return <Outlet />;
  }

  return <Navigate to={factoryOnboardingPath(organizationId, factoryKey)} replace />;
}
