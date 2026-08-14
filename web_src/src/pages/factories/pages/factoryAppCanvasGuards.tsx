import { Navigate } from "react-router";
import { factoryOverviewPath } from "../lib/factoryPagePaths";

export function FactoryAppCanvasRedirect({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  return <Navigate to={factoryOverviewPath(organizationId, factoryKey)} replace />;
}
