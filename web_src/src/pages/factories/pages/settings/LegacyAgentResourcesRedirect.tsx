import { Navigate, useSearchParams } from "react-router";

import { legacyAgentResourcesPath } from "./legacyAgentResourcesPath";

export function LegacyAgentResourcesRedirect() {
  const [searchParams] = useSearchParams();
  return <Navigate to={legacyAgentResourcesPath(searchParams)} replace />;
}
