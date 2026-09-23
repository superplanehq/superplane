import { canSaveIntakeConnection, type IntakeConnectionBinding } from "./intakeConnectionModel";
import type { IntakeSettingsConnection } from "./IntakeSourceSettingsPopup";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";
import type { IntakeConnectionModel } from "./useIntakeConnection";

export function intakeSettingsConnectionProps(
  intake: ConfiguredLineIntakeSource,
  connection: IntakeConnectionModel,
  savedBinding: IntakeConnectionBinding,
  integrationsBasePath: string,
): IntakeSettingsConnection | undefined {
  if (!connection.enabled) {
    return undefined;
  }
  return {
    health: intake.health,
    integrationsBasePath,
    binding: connection.binding,
    integrations: connection.integrations,
    integrationsLoading: connection.integrationsLoading,
    projects: connection.projects,
    projectsLoading: connection.projectsLoading,
    projectsError: connection.projectsError,
    connecting: connection.connecting,
    connectError: connection.connectError,
    saveDisabled: !canSaveIntakeConnection(connection.enabled, intake.health, savedBinding, connection.binding),
    onBindingChange: connection.setBinding,
    onConnect: () => void connection.connect(),
    onReconnect: connection.reconnect,
    onRetryProjects: connection.retryProjects,
  };
}
