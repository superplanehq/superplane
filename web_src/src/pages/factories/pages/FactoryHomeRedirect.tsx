import { Navigate } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryLineDetailPath, factoryOverviewPath, firstFactoryLineId } from "../lib/factoryPagePaths";

/** Sends the workspace index and `/lines` to the Kanban board. */
export function FactoryHomeRedirect() {
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  const lineId = firstFactoryLineId(factory);
  const pathname = lineId
    ? factoryLineDetailPath(organizationId, factoryKey, lineId)
    : factoryOverviewPath(organizationId, factoryKey);
  return <Navigate to={{ pathname, search: "" }} replace />;
}
