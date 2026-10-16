import { useCallback, useEffect, useRef, type Dispatch, type SetStateAction } from "react";

import type { OrganizationsIntegration } from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { useOnboardingAgentModel } from "./onboardingAgentModel";
import type { OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";
import { restoreSavedAgentSelection } from "./onboardingStatus";
import { useOnboardingAgentContext } from "./useOnboardingAgentPlan";

type SetSelections = Dispatch<SetStateAction<IntegrationSelections>>;

/** Keep a saved provider key when the connection list syncs during onboarding. */
export function useOnboardingAgentPageState(args: {
  organizationId: string;
  factoryId: string;
  connected: Set<IntegrationId>;
  customProvider: boolean;
  agentCredentialChoice: OnboardingAgentCredentialChoice | null;
  setSelections: SetSelections;
  connectedIntegrations: OrganizationsIntegration[];
  agentIntegrationId: string | undefined;
}) {
  const preferOwnKey = args.agentCredentialChoice === "own-key";
  const [selectedModel, selectModel] = useOnboardingAgentModel(args.factoryId);
  const agent = useOnboardingAgentContext(
    args.organizationId,
    args.connected,
    preferOwnKey,
    args.customProvider,
    selectedModel,
  );
  const onSelectionsChange = useSavedAgentSelectionChange(
    args.setSelections,
    args.connectedIntegrations,
    args.agentIntegrationId,
  );
  return { agent, selectModel, onSelectionsChange };
}

function useSavedAgentSelectionChange(
  setSelections: SetSelections,
  connectedIntegrations: OrganizationsIntegration[],
  agentIntegrationId: string | undefined,
) {
  const connectedIntegrationsRef = useRef(connectedIntegrations);
  connectedIntegrationsRef.current = connectedIntegrations;
  const savedAgentIntegrationIdRef = useRef(agentIntegrationId);
  savedAgentIntegrationIdRef.current = agentIntegrationId;
  const onSelectionsChange = useCallback(
    (next: IntegrationSelections) => {
      setSelections(
        restoreSavedAgentSelection(next, connectedIntegrationsRef.current, savedAgentIntegrationIdRef.current) ?? next,
      );
    },
    [setSelections],
  );
  useEffect(() => {
    setSelections(
      (current) => restoreSavedAgentSelection(current, connectedIntegrations, agentIntegrationId) ?? current,
    );
  }, [agentIntegrationId, connectedIntegrations, setSelections]);
  return onSelectionsChange;
}
