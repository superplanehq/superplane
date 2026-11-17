import { Navigate, useParams } from "react-router";

import { factoryAgentPath } from "../../lib/factoryPagePaths";

export function LegacyRepositorySettingsRoute() {
  const { organizationId = "", factoryKey = "" } = useParams<{ organizationId: string; factoryKey: string }>();
  return <Navigate to={factoryAgentPath(organizationId, factoryKey)} replace />;
}
