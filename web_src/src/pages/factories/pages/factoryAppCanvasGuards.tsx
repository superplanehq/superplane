import { Navigate } from "react-router";
import { factoryOverviewPath } from "../lib/factoryPagePaths";

export function FactoryAppCanvasRedirect({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  return <Navigate to={factoryOverviewPath(organizationId, factoryId)} replace />;
}
