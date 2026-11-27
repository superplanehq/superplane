import { usePermissions } from "@/contexts/usePermissions";
import { Navigate } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryOverviewPath, factoryPublicBoardPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { LineBoardRouteSwitch } from "../mobile/MobileShellSwitch";

/**
 * Workspace index: the line board. A member without work-order access cannot
 * use the board, so the public board takes over at its line URL.
 */
export function FactoryBoardIndex() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { canAct, isLoading } = usePermissions();

  if (isLoading) {
    return <div className="flex h-full min-h-0 w-full" data-testid="factory-board-loading" />;
  }

  if (!canAct("work_orders", "read")) {
    const lineId = firstFactoryLineId(factory);
    const pathname = lineId
      ? factoryPublicBoardPath(organizationId, routeSegment, lineId)
      : factoryOverviewPath(organizationId, routeSegment);
    return <Navigate to={{ pathname, search: "" }} replace />;
  }

  return <LineBoardRouteSwitch />;
}
