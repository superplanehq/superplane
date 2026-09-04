import { Navigate, useLocation, useParams } from "react-router";
import { workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";

/**
 * Back-compat for the plural `/work-orders(...)` tree, renamed to `/tasks`.
 * Forwards the child segment (list, `new`, or an id) and the query string so
 * old bookmarks keep working.
 */
export function LegacyWorkOrdersRedirect() {
  const {
    organizationId,
    factoryKey,
    "*": splat,
  } = useParams<{
    organizationId: string;
    factoryKey: string;
    "*": string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey) {
    return <Navigate to="/" replace />;
  }

  const suffix = splat ? `/${splat}` : "";
  return <Navigate to={`${workOrdersPath(organizationId, factoryKey)}${suffix}${location.search}`} replace />;
}

/** Back-compat for the singular `/work-order/:orderNumber` permalink, renamed to `/task/:orderNumber`. */
export function LegacyWorkOrderPermalinkRedirect() {
  const { organizationId, factoryKey, orderNumber } = useParams<{
    organizationId: string;
    factoryKey: string;
    orderNumber: string;
  }>();
  const location = useLocation();

  if (!organizationId || !factoryKey || !orderNumber) {
    return <Navigate to="/" replace />;
  }

  return <Navigate to={`${workOrderDetailPath(organizationId, factoryKey, orderNumber)}${location.search}`} replace />;
}
