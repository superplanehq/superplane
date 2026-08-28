import { IntegrationCatalog } from "./IntegrationCatalog";

interface IntegrationsProps {
  organizationId: string;
}

export function Integrations({ organizationId }: IntegrationsProps) {
  return <IntegrationCatalog organizationId={organizationId} appearance="legacy" />;
}
