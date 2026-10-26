import { Navigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath } from "../lib/factoryPagePaths";
import { factoryCustomVerificationSetupPath } from "../lib/customVerificationPaths";

/** Redirect old setup URLs to the board modal URL. */
export function CustomVerificationSetupPage() {
  const { organizationId, routeSegment, factory } = useFactoriesLayout();
  const { lineId, automationId } = useParams<{ lineId?: string; automationId?: string }>();
  const fallbackLineId = factory?.lines?.[0]?.id;
  const targetLineId = lineId || fallbackLineId;
  if (!targetLineId) {
    return <Navigate to={factoryHomePath(organizationId, routeSegment)} replace />;
  }
  return (
    <Navigate
      to={factoryCustomVerificationSetupPath(organizationId, routeSegment, targetLineId, automationId)}
      replace
    />
  );
}
