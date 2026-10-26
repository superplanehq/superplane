import { Navigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath } from "../lib/factoryPagePaths";
import { factoryCustomAutomationSetupPath } from "../lib/customAutomationPaths";

/** Redirect old setup URLs to the board modal URL. */
export function CustomAutomationSetupPage() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { lineId, automationId } = useParams<{ lineId?: string; automationId?: string }>();
  const fallbackLineId = factory?.lines?.[0]?.id;
  const targetLineId = lineId || fallbackLineId;
  if (!targetLineId) {
    return <Navigate to={factoryHomePath(organizationId, routeSegment)} replace />;
  }
  return (
    <Navigate to={factoryCustomAutomationSetupPath(organizationId, routeSegment, targetLineId, automationId)} replace />
  );
}
