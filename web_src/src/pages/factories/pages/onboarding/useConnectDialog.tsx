import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { useMemo, useState } from "react";

import type { IntegrationId } from "./onboardingFixtures";
import { integrationDefinitionFor, STORYBOOK_ORG_ID } from "./integrationDefinitions";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

/**
 * Opens the real SuperPlane IntegrationCreateDialog with Storybook stubs
 * so setup connect feels grounded in product UI.
 */
export function useConnectDialog(setup: OnboardingSetupApi) {
  const [pending, setPending] = useState<IntegrationId | null>(null);
  const [open, setOpen] = useState(false);

  const definition = useMemo(() => (pending ? integrationDefinitionFor(pending) : null), [pending]);

  const requestConnect = (id: IntegrationId) => {
    setPending(id);
    setOpen(true);
  };

  const markConnected = (id: IntegrationId) => {
    setup.connectIntegration(id);
    if (id === "linear") setup.setIssuesChoice("linear");
    if (id === "jira") setup.setIssuesChoice("jira");
  };

  const dialog = (
    <IntegrationCreateDialog
      open={open && !!definition}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setPending(null);
      }}
      integrationDefinition={definition}
      organizationId={STORYBOOK_ORG_ID}
      defaultName={definition?.name ? `${definition.name}-1` : ""}
      onCreateIntegration={async (payload) => ({
        integration: {
          metadata: {
            id: `storybook-int-${payload.integrationName}`,
            name: payload.name,
            integrationName: payload.integrationName,
          },
        },
      })}
      onCreated={() => {
        if (pending) markConnected(pending);
        setOpen(false);
        setPending(null);
      }}
      onCapabilitySetupRequired={(_name, _id) => {
        // Wireframe: treat capability hand-off as connected so the flow continues.
        if (pending) markConnected(pending);
        setOpen(false);
        setPending(null);
      }}
    />
  );

  return { requestConnect, requestConfigure: requestConnect, dialog };
}
