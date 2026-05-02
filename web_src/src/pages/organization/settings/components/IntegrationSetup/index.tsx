import { SetupView } from "./SetupView";
import { useIntegrationSetupController } from "./useIntegrationSetupController";

interface IntegrationSetupProps {
  organizationId: string;
}

export function IntegrationSetup({ organizationId }: IntegrationSetupProps) {
  const setup = useIntegrationSetupController(organizationId);

  return <SetupView setup={setup} />;
}
