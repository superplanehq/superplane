import type { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { showErrorToast } from "@/lib/toast";
import {
  selectReadyIntegrationInstance,
  type useIntegrationConnectDialog,
} from "@/pages/home/useIntegrationConnectDialog";
import { Fragment, createElement, useState } from "react";

import { CodexConnectionSetupDialog } from "../../codex/CodexConnectionSetupDialog";
import { OnboardingConnectDialogs } from "./CustomProviderConnectDialog";
import { AGENT_PROVIDER_IDS } from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";
import type { useOnboardingIntegrationSelections } from "./useOnboardingIntegrationSelections";

export function useOnboardingConnectionDialogs(args: {
  organizationId: string;
  factoryId: string;
  bringYourOwnKey: ReturnType<typeof useBYOKAccess>;
  connect: ReturnType<typeof useIntegrationConnectDialog>;
  integrations: ReturnType<typeof useOnboardingIntegrationSelections>;
  connectedIntegrations: Parameters<typeof selectReadyIntegrationInstance>[0];
  existingIntegrationNames: Set<string>;
}) {
  const { bringYourOwnKey, connect, integrations, connectedIntegrations, existingIntegrationNames } = args;
  const [subscriptionDialogOpen, setSubscriptionDialogOpen] = useState(false);
  const [customProviderDialogOpen, setCustomProviderDialogOpen] = useState(false);
  const requestConnect = (id: IntegrationId) => {
    if (providerConnectionDenied(id, bringYourOwnKey)) return false;
    if (id !== "customLlm" || !bringYourOwnKey.customProviderAllowed) return connect.requestConnect(id);
    const existing = selectReadyIntegrationInstance(connectedIntegrations, integrations.selections, "customLlm");
    if (existing) {
      integrations.setSelections(existing);
      // True means the browser left this page. A saved connection stays here.
      return false;
    }
    setCustomProviderDialogOpen(true);
    return false;
  };

  return {
    requestConnect,
    connectSubscription: () => setSubscriptionDialogOpen(true),
    integrationDialogs: createElement(
      Fragment,
      null,
      createElement(OnboardingConnectDialogs, {
        connectDialogs: connect.dialogs,
        customProviderOpen: customProviderDialogOpen,
        organizationId: args.organizationId,
        existingNames: existingIntegrationNames,
        selections: integrations.selections,
        onSelectionsChange: integrations.setSelections,
        onCloseCustomProvider: () => setCustomProviderDialogOpen(false),
      }),
      createElement(CodexConnectionSetupDialog, {
        organizationId: args.organizationId,
        factoryId: args.factoryId,
        open: bringYourOwnKey.noHostedDefault && subscriptionDialogOpen,
        onOpenChange: setSubscriptionDialogOpen,
        onConnected: () => setSubscriptionDialogOpen(false),
      }),
    ),
  };
}

function providerConnectionDenied(id: IntegrationId, access: ReturnType<typeof useBYOKAccess>) {
  const providerDenied = AGENT_PROVIDER_IDS.some((provider) => provider === id) && !access.allowed;
  if (!providerDenied && (id !== "customLlm" || access.customProviderAllowed)) return false;
  showErrorToast("Your organization does not have access to provider credentials. Contact your administrator.");
  return true;
}
