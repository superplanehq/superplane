import { SetupView } from "./SetupView";
import { useIntegrationSetupController } from "./useIntegrationSetupController";
import { useReportPageReady } from "@/hooks/useReportPageReady";

interface IntegrationSetupProps {
  organizationId: string;
}

export function IntegrationSetup({ organizationId }: IntegrationSetupProps) {
  const setup = useIntegrationSetupController(organizationId);

  useReportPageReady(!setup.queries.isAvailableIntegrationsLoading);

  return <SetupView setup={setup} />;
}
