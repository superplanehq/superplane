import { useCallback, useEffect, useRef, type Dispatch, type SetStateAction } from "react";

import type { OrganizationsIntegration } from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { codexSubscriptionReady, useFactoryCodexConnection } from "./useFactoryCodexConnection";
import { useOnboardingAgentModel } from "./onboardingAgentModel";
import { AGENT_PROVIDER_IDS, type OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";
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
  subscriptionOffered: boolean;
  setSelections: SetSelections;
  connectedIntegrations: OrganizationsIntegration[];
  agentIntegrationId: string | undefined;
}) {
  const subscriptionSelected = args.subscriptionOffered && args.agentCredentialChoice === "subscription";
  const codex = useFactoryCodexConnection(args.organizationId, args.factoryId, args.subscriptionOffered);
  const subscriptionReady = onboardingSubscriptionReady(codex.query);
  const subscription = subscriptionSelected
    ? {
        ready: subscriptionReady,
        loading: codex.query.isPending,
      }
    : undefined;
  const preferOwnKey = args.agentCredentialChoice === "own-key";
  const [selectedModel, selectModel] = useOnboardingAgentModel(args.factoryId);
  const agent = useOnboardingAgentContext(args.organizationId, args.connected, preferOwnKey, args.customProvider, {
    selectedModel,
    subscription,
  });
  const onSelectionsChange = useSavedAgentSelectionChange(
    args.factoryId,
    args.setSelections,
    args.connectedIntegrations,
    args.agentIntegrationId,
  );
  return { agent, subscriptionReady, selectModel, onSelectionsChange };
}

function useSavedAgentSelectionChange(
  factoryId: string,
  setSelections: SetSelections,
  connectedIntegrations: OrganizationsIntegration[],
  agentIntegrationId: string | undefined,
) {
  const storageKey = `superplane:onboarding-agent-integration:${factoryId}`;
  const connectedIntegrationsRef = useRef(connectedIntegrations);
  connectedIntegrationsRef.current = connectedIntegrations;
  const savedAgentIntegrationIdRef = useRef(agentIntegrationId);
  const rememberedId = localStorage.getItem(storageKey) ?? undefined;
  const rememberedIntegrationReady = connectedIntegrations.some(
    (instance) => instance.metadata?.id === rememberedId && instance.status?.state === "ready",
  );
  savedAgentIntegrationIdRef.current = rememberedIntegrationReady ? rememberedId : agentIntegrationId;
  const onSelectionsChange = useCallback(
    (next: IntegrationSelections) => {
      const provider = AGENT_PROVIDER_IDS.find((id) => next[id]?.ready);
      const selectedId = provider ? next[provider]?.id : undefined;
      if (selectedId) {
        localStorage.setItem(storageKey, selectedId);
        savedAgentIntegrationIdRef.current = selectedId;
      }
      setSelections(
        restoreSavedAgentSelection(next, connectedIntegrationsRef.current, savedAgentIntegrationIdRef.current) ?? next,
      );
    },
    [setSelections, storageKey],
  );
  useEffect(() => {
    setSelections(
      (current) =>
        restoreSavedAgentSelection(current, connectedIntegrations, savedAgentIntegrationIdRef.current) ?? current,
    );
  }, [agentIntegrationId, connectedIntegrations, setSelections, storageKey]);
  return onSelectionsChange;
}

function onboardingSubscriptionReady(query: ReturnType<typeof useFactoryCodexConnection>["query"]): boolean {
  const connection = query.data;
  return (
    !query.isError &&
    connection?.eligible === true &&
    !connection.revocationPending &&
    codexSubscriptionReady(connection) &&
    Boolean(connection.availableModels?.length)
  );
}
