import { useMemo } from "react";
import type { FactoriesFactoryAppTemplateAgent, FactoriesFactoryOnboarding } from "@/api-client";
import { workspaceAgentFromOnboarding } from "@/pages/factories/lib/customAutomation";
import { useConnectedIntegrations } from "./useIntegrations";

export function useCustomAutomationEditorAgent(
  organizationId: string,
  onboarding: FactoriesFactoryOnboarding | undefined,
  automationId: string | undefined,
  saved: FactoriesFactoryAppTemplateAgent | undefined,
) {
  const needsConnections = automationId
    ? saved?.component === "runnerOpenRouter" && saved.credentialSource === "integration"
    : !!onboarding &&
      !onboarding.codexSubscription &&
      (!!onboarding.agentIntegrationId || onboarding.agentHarness !== "AGENT_HARNESS_SUPERPLANE");
  const { data: connections, isError } = useConnectedIntegrations(organizationId, { enabled: needsConnections });
  const agent = useMemo(
    () => (automationId ? saved : workspaceAgentFromOnboarding(onboarding, connections)),
    [automationId, saved, onboarding, connections],
  );
  const needsProvider = agent?.component === "runnerOpenRouter" && agent.credentialSource === "integration";
  const connection = needsProvider
    ? connections?.find(({ metadata }) => metadata?.name === agent.credentialIntegrationName)
    : undefined;
  return {
    agent,
    customProvider: connection?.metadata?.integrationName === "customLlm",
    ready: !!agent?.component && (!needsConnections || (!!connections && !isError)) && (!needsProvider || !!connection),
  };
}
