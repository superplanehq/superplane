import { useIntegrationResources } from "@/hooks/useIntegrations";

/** Resource type string must match pkg/integrations/grafana resourceTypeDashboard. */
export const GRAFANA_DASHBOARD_RESOURCE_TYPE = "dashboard";

type GrafanaDashboardMetadataLabelProps = {
  organizationId?: string;
  integrationId?: string;
  dashboardUid: string;
};

/**
 * Resolves a stored dashboard UID to the Grafana dashboard title for canvas metadata.
 * Falls back to the UID when the integration is unavailable or the list has not loaded.
 */
export function GrafanaDashboardMetadataLabel({
  organizationId,
  integrationId,
  dashboardUid,
}: GrafanaDashboardMetadataLabelProps) {
  const { data: resources } = useIntegrationResources(
    organizationId ?? "",
    integrationId ?? "",
    GRAFANA_DASHBOARD_RESOURCE_TYPE,
  );

  const name = resources?.find((r) => r.id === dashboardUid)?.name?.trim();
  const display = name || dashboardUid;

  return <span className="truncate">Dashboard: {display}</span>;
}
