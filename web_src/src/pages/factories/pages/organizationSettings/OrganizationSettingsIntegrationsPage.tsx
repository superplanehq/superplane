import { useParams } from "react-router";
import { usePageTitle } from "@/hooks/usePageTitle";
import { IntegrationCatalog } from "@/pages/organization/settings/IntegrationCatalog";
import { FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsIntegrationsPage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  usePageTitle(["Integrations"]);

  return (
    <FactorySettingsPageFrame title="Integrations" subtitle="Connect external tools and services to extend SuperPlane.">
      <IntegrationCatalog organizationId={organizationId} appearance="factories" />
    </FactorySettingsPageFrame>
  );
}
