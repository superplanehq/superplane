import { usePermissions } from "@/contexts/usePermissions";
import { Navigate } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryPlanningPath, firstFactoryLineId } from "../lib/factoryPagePaths";

/** Old Planning setup URLs open Planning settings when the user can update the factory. */
function planningSetupRedirectTarget(input: {
  canUpdate: boolean;
  linePresent: boolean;
  boardHref: string;
  settingsHref: string;
}): string {
  if (!input.canUpdate || !input.linePresent) {
    return input.boardHref;
  }
  return input.settingsHref;
}

export function PlanningSetupPage() {
  const { canAct } = usePermissions();
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const lineId = firstFactoryLineId(factory);
  const boardHref = factoryHomePath(organizationId, routeSegment);
  const settingsHref = lineId ? factoryPlanningPath(organizationId, routeSegment) : boardHref;

  return (
    <Navigate
      to={planningSetupRedirectTarget({
        canUpdate: canAct("factories", "update"),
        linePresent: Boolean(lineId),
        boardHref,
        settingsHref,
      })}
      replace
    />
  );
}
