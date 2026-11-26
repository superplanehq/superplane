import { Navigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { factoryCustomAutomationSetupPath } from "../lib/customAutomationPaths";

/** Redirect old setup URLs to the board modal URL. */
export function CustomAutomationSetupPage() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { automationId } = useParams<{ automationId?: string }>();
  if (!firstFactoryLineId(factory)) {
    return <Navigate to={factoryHomePath(organizationId, routeSegment)} replace />;
  }
  return <Navigate to={factoryCustomAutomationSetupPath(organizationId, routeSegment, automationId)} replace />;
}
