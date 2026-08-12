import { Navigate, Outlet, useLocation, useParams } from "react-router-dom";

import { factoryOnboardingPath } from "../../lib/factoryPagePaths";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/**
 * Storybook-only: while onboarding is pending, keep the user on the setup page
 * (settings routes sit outside FactoriesLayout and remain reachable).
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId = "", factoryId = "" } = useParams<{ organizationId: string; factoryId: string }>();

  const pending = onboarding?.pending;
  if (!pending) {
    return <Outlet />;
  }

  const onboardingPath = factoryOnboardingPath(organizationId, pending.workspaceId || factoryId);
  const onOnboardingRoute = location.pathname.endsWith("/onboarding");

  if (!onOnboardingRoute) {
    return <Navigate to={onboardingPath} replace />;
  }

  return <Outlet />;
}
