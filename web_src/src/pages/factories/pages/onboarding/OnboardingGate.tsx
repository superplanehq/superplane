import { Navigate, Outlet, useLocation, useParams } from "react-router";

import { factoryOnboardingPath } from "../../lib/factoryPagePaths";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/**
 * Storybook-only: while onboarding is pending, keep the user on the setup route.
 * The factories sidebar is hidden until setup finishes.
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId = "", factoryId = "" } = useParams<{ organizationId: string; factoryId: string }>();

  const pending = onboarding?.pending;
  if (!pending) {
    return <Outlet />;
  }

  if (location.pathname.endsWith("/onboarding")) {
    return <Outlet />;
  }

  const onboardingPath = factoryOnboardingPath(organizationId, pending.workspaceId || factoryId);
  return <Navigate to={onboardingPath} replace />;
}
