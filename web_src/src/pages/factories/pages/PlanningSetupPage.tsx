import { Navigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryPlanningPath, firstFactoryLineId } from "../lib/factoryPagePaths";

/** Old Planning setup URLs open Planning settings. */
export function PlanningSetupPage() {
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  const { lineId } = useParams<{ lineId?: string }>();
  const line = factory?.lines?.find((entry) => entry.id === lineId);
  const to = line?.id
    ? factoryPlanningPath(organizationId, factoryKey, line.id)
    : factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory));

  return <Navigate to={to} replace />;
}
