import { usePermissions } from "@/contexts/usePermissions";
import { Navigate, useParams } from "react-router";

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
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  const { lineId } = useParams<{ lineId?: string }>();
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const boardHref = factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));
  const settingsHref = line?.id ? factoryPlanningPath(organizationId, factoryKey, line.id) : boardHref;

  return (
    <Navigate
      to={planningSetupRedirectTarget({
        canUpdate: canAct("factories", "update"),
        linePresent: Boolean(line?.id),
        boardHref,
        settingsHref,
      })}
      replace
    />
  );
}
