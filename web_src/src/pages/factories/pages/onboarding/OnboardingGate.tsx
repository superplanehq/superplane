import { Navigate, Outlet, useLocation, useParams } from "react-router-dom";

import { factoryOnboardingPath } from "../../lib/factoryPagePaths";
import type { OnboardingNavProgress } from "./onboardingStorybookContextValue";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function isUnlockedFactoryPath(pathname: string, progress: OnboardingNavProgress): boolean {
  if (pathname.endsWith("/onboarding")) return true;

  // Overview is always available during setup.
  if (pathname.endsWith("/overview") || /\/workspaces\/[^/]+\/?$/.test(pathname)) return true;

  if ((pathname.includes("/wiki") || pathname.includes("/velocity")) && progress.repoReady && !progress.analyzingRepo) {
    return true;
  }
  if (pathname.includes("/work-orders") && progress.issuesReady && !progress.analyzingIssues) {
    return true;
  }
  if (
    (pathname.includes("/lines") || pathname.includes("/automations") || pathname.includes("/apps")) &&
    progress.agentReady &&
    !progress.analyzingAgent
  ) {
    return true;
  }

  return false;
}

/**
 * Storybook-only: while onboarding is pending, keep the user on setup unless a
 * nav destination has finished generating and is unlocked.
 */
export function OnboardingGate() {
  const onboarding = useOnboardingStorybook();
  const location = useLocation();
  const { organizationId = "", factoryId = "" } = useParams<{ organizationId: string; factoryId: string }>();

  const pending = onboarding?.pending;
  if (!pending) {
    return <Outlet />;
  }

  const progress = onboarding.setupProgress;
  if (isUnlockedFactoryPath(location.pathname, progress)) {
    return <Outlet />;
  }

  const onboardingPath = factoryOnboardingPath(organizationId, pending.workspaceId || factoryId);
  return <Navigate to={onboardingPath} replace />;
}
