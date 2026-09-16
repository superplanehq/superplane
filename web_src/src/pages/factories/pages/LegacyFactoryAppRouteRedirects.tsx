import { Navigate, useLocation, useParams } from "react-router";

import { factoryAppPath, factoryAppSplitRunPath } from "../lib/factoryPagePaths";

/**
 * Back-compat for workspace canvas URLs under `/apps/:appId`, renamed to
 * `/automations/:automationId`.
 */
export function LegacyFactoryAppRedirect() {
  const { organizationId, factoryKey, appId } = useParams<{
    organizationId: string;
    factoryKey: string;
    appId: string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey || !appId) {
    return <Navigate to="/" replace />;
  }

  return <Navigate to={`${factoryAppPath(organizationId, factoryKey, appId)}${location.search}`} replace />;
}

/** Back-compat for `/apps/:appId/split-run`, renamed to `/automations/:automationId/split-run`. */
export function LegacyFactoryAppSplitRunRedirect() {
  const { organizationId, factoryKey, appId } = useParams<{
    organizationId: string;
    factoryKey: string;
    appId: string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey || !appId) {
    return <Navigate to="/" replace />;
  }

  const path = factoryAppSplitRunPath(organizationId, factoryKey, appId);
  return <Navigate to={`${path}${location.search}`} replace />;
}
