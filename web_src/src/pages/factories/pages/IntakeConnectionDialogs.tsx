import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";

import { intakeConnectionDefaultName, type IntakeConnectionModel } from "./useIntakeConnection";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function IntakeConnectionDialogs({
  organizationId,
  sourceId,
  connection,
}: {
  organizationId: string;
  sourceId: LineIntakeSourceId;
  connection: IntakeConnectionModel;
}) {
  return (
    <>
      <IntegrationCreateDialog
        open={connection.connectOpen}
        onOpenChange={connection.setConnectOpen}
        integrationDefinition={connection.definition}
        organizationId={organizationId}
        onCreateIntegration={async (payload) => {
          const response = await connection.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={connection.createIntegration.reset}
        defaultName={intakeConnectionDefaultName(sourceId)}
        existingIntegrationNames={connection.existingNames}
        hiddenFieldNames={sourceId === "productive-tasks" ? ["region"] : undefined}
        onCreated={connection.completeConnection}
        setupReturnTo={connection.returnPath}
      />
      <ConfigureIntegrationDialog
        integrationId={connection.configureIntegrationId}
        organizationId={organizationId}
        onClose={connection.closeConfigure}
      />
    </>
  );
}
