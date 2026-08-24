import { Navigate } from "react-router";
import { factoryHomePath } from "../lib/factoryPagePaths";

export function FactoryAppCanvasRedirect({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  return <Navigate to={factoryHomePath(organizationId, factoryKey)} replace />;
}
