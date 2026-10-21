import { useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { connectPrivateGitHubApp } from "@/lib/privateGitHubApp";
import { useMemo } from "react";
import { useNavigate } from "react-router";

import { writeOnboardingGithubEnterprise } from "./onboardingGithubEnterprise";

export function useFirstRunGitHubEnterprise(args: {
  organizationId: string;
  factoryId: string;
  onCreated?: (integrationId: string) => Promise<void> | void;
}) {
  const navigate = useNavigate();
  const createIntegration = useCreateIntegration(args.organizationId, "install_wizard");
  const connectedQuery = useConnectedIntegrations(args.organizationId, { enabled: Boolean(args.organizationId) });
  const connected = useMemo(() => connectedQuery.data ?? [], [connectedQuery.data]);
  const existingNames = useMemo(
    () =>
      new Set(
        connected
          .map((integration) => integration.metadata?.name?.trim())
          .filter((name): name is string => Boolean(name)),
      ),
    [connected],
  );

  const connect = async (host: string, returnTo: string) => {
    writeOnboardingGithubEnterprise(args.factoryId, true, host);
    return connectPrivateGitHubApp({
      useWizard: true,
      organizationId: args.organizationId,
      host,
      factoryOnboarding: true,
      returnTo,
      existingNames,
      connected,
      goTo: navigate,
      create: async (payload) => {
        const response = await createIntegration.mutateAsync(payload);
        return response.data;
      },
      onCreated: args.onCreated,
    });
  };

  return { connect };
}
