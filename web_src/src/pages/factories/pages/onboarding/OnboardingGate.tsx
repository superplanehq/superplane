import type { FactoriesFactory } from "@/api-client";
import { Navigate, Outlet, useLocation } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import {
  factoryIntakePath,
  factoryOverviewPath,
  factorySetupPath,
  firstFactoryLineId,
} from "../../lib/factoryPagePaths";
import { isFactoryOnboardingComplete } from "./onboardingStatus";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function isWorkspaceSetupRoute(pathname: string) {
  return pathname.endsWith("/setup") || pathname.endsWith("/onboarding");
}

/**
 * Where a finished workspace goes when it leaves setup: the line board with
 * the Intake drawer open, the same place the Finish action opens. Finish and
 * this redirect run in the same tick, so a different target here would drop
 * the drawer whenever this redirect lands last.
 */
function pathAfterSetup(organizationId: string, factoryKey: string, factory: FactoriesFactory | null) {
  const lineId = firstFactoryLineId(factory);
  if (!lineId) {
    return factoryOverviewPath(organizationId, factoryKey);
  }
  return factoryIntakePath(organizationId, factoryKey, lineId);
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
      return <Navigate to={pathAfterSetup(organizationId, factoryKey, factory)} replace />;
    }
    return <Outlet />;
  }

  if (isSetupRoute) {
    return <Outlet />;
  }

  return <Navigate to={factorySetupPath(organizationId, factoryKey)} replace />;
}
