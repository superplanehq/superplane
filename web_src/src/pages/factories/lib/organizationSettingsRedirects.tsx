import { Navigate, useLocation, useParams } from "react-router";

import { organizationSettingsPath, organizationSettingsSectionPath } from "./factoryPagePaths";

export function LegacyWorkspaceOrganizationSettingsRedirect() {
  const {
    organizationId,
    factoryKey,
    "*": rest,
  } = useParams<{
    organizationId: string;
    factoryKey: string;
    "*": string;
  }>();
  const location = useLocation();

  if (!organizationId) {
    return <Navigate to="/" replace />;
  }

  const suffix = rest ? `/${rest}` : "";
  const previousState =
    location.state && typeof location.state === "object" ? (location.state as Record<string, unknown>) : {};

  return (
    <Navigate
      to={`${organizationSettingsPath(organizationId)}${suffix}${location.search}`}
      replace
      state={{ ...previousState, fromFactoryKey: factoryKey }}
    />
  );
}

export function LegacyLLMSpendRedirect() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const location = useLocation();

  if (!organizationId) {
    return <Navigate to="/" replace />;
  }

  return (
    <Navigate to={`${organizationSettingsSectionPath(organizationId, "workspace-usage")}${location.search}`} replace />
  );
}
