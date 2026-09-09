import type { FactoriesFactory } from "@/api-client";
import { useEffect, useRef } from "react";
import { Navigate, Outlet, useLocation } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryHomePath, factoryOverviewPath, factorySetupPath, firstFactoryLineId } from "../../lib/factoryPagePaths";
import { holdSetupAfterThisVisitCompletes } from "./onboardingGateState";
import { isFactoryOnboardingComplete } from "./onboardingStatus";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function isWorkspaceSetupRoute(pathname: string) {
  return pathname.endsWith("/setup") || pathname.endsWith("/onboarding");
}

/**
 * Where a finished workspace goes when it leaves setup: the line board.
 * Finish now holds this route for analysis, then the board button navigates
 * here. A later visit to setup still uses this path.
 */

function pathAfterSetup(organizationId: string, factoryKey: string, factory: FactoriesFactory | null) {
  const lineId = firstFactoryLineId(factory);
  if (!lineId) {
    return factoryOverviewPath(organizationId, factoryKey);
  }
  return factoryHomePath(organizationId, factoryKey, lineId);
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
  // `complete: true` writes completedAt into the factory cache before
  // FirstRunSetup stores the analysis destination. Hold this visit on setup
  // so that write cannot unmount the analysis screen. A later open of setup
  // still leaves for the board.
  const startedIncompleteFactoryId = useRef(isIncomplete ? factoryId : null);
  useEffect(() => {
    if (isIncomplete) {
      startedIncompleteFactoryId.current = factoryId;
      return;
    }
    if (!isSetupRoute && startedIncompleteFactoryId.current === factoryId) {
      startedIncompleteFactoryId.current = null;
    }
  }, [factoryId, isIncomplete, isSetupRoute]);

  if (!isIncomplete) {
    if (holdSetupAfterThisVisitCompletes(startedIncompleteFactoryId.current === factoryId, isSetupRoute)) {
      return <Outlet />;
    }
    if (isSetupRoute) {
      return <Navigate to={pathAfterSetup(organizationId, factoryKey, factory)} replace />;
    }
    return <Outlet />;
  }

  if (!onboarding && factory?.onboarding?.initial === true) {
    return <Navigate to="/onboarding" replace />;
  }

  if (isSetupRoute) {
    return <Outlet />;
  }

  return <Navigate to={factorySetupPath(organizationId, factoryKey)} replace />;
}
