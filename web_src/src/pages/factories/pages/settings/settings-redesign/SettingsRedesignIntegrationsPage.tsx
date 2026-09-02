import { usePageTitle } from "@/hooks/usePageTitle";
import { IntegrationCatalog } from "@/pages/organization/settings/IntegrationCatalog";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";

export function SettingsRedesignIntegrationsPage() {
  const { organizationId } = useFactorySettingsLayout();
  usePageTitle(["Integrations"]);

  return (
    <FactorySettingsPageFrame title="Integrations" subtitle="Connect the tools this organization uses.">
      <IntegrationCatalog organizationId={organizationId} appearance="factories" />
    </FactorySettingsPageFrame>
  );
}
