import {
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegration,
  useNextIntegrationSetupStep,
} from "@/hooks/useIntegrations";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { openRedirectPrompt } from "@/lib/integrations";
import { connectPrivateGitHubApp, githubPrivateAppSetupPath } from "@/lib/privateGitHubApp";
import { useMemo } from "react";
import { useNavigate } from "react-router";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { writeOnboardingGithubEnterprise } from "./onboardingGithubEnterprise";

const GITHUB_OWNER_TYPE_ORGANIZATION = "Organization";
const GITHUB_LOCATION_ENTERPRISE_CLOUD = "ghe.com";

function githubEnterpriseIntegrationsBasePath(organizationId: string, factoryKey: string) {
  return factorySettingsSectionPath(organizationId, factoryKey, "organization", "integrations");
}

export function useFirstRunGitHubEnterprise(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  integrationId?: string;
  host?: string;
  onCreated?: (integrationId: string) => Promise<void> | void;
}) {
  const navigate = useNavigate();
  const createIntegration = useCreateIntegration(args.organizationId, "install_wizard");
  const nextSetupStep = useNextIntegrationSetupStep(args.organizationId);
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
  const integrationId = args.integrationId?.trim() ?? "";
  const described = useIntegration(args.organizationId, integrationId);
  const instance = described.data ?? connected.find((integration) => integration.metadata?.id === integrationId);
  const ready = instance?.status?.state === "ready";
  const currentStep = instance?.status?.setupState?.currentStep;
  const currentStepName = currentStep?.name?.trim() ?? "";
  const integrationsBasePath = githubEnterpriseIntegrationsBasePath(args.organizationId, args.factoryKey);

  const connect = async (host: string, returnTo: string) => {
    writeOnboardingGithubEnterprise(args.factoryId, true, host);
    return connectPrivateGitHubApp({
      useWizard: true,
      stayAfterCreate: true,
      organizationId: args.organizationId,
      integrationsBasePath,
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

  const submitOwner = async (owner: string, returnTo: string) => {
    if (!integrationId) return false;
    rememberIntegrationSetupReturn(args.organizationId, returnTo);
    const response = await nextSetupStep.mutateAsync({
      integrationId,
      inputs: {
        githubLocation: GITHUB_LOCATION_ENTERPRISE_CLOUD,
        host: args.host?.trim() ?? "",
        ownerType: GITHUB_OWNER_TYPE_ORGANIZATION,
        owner,
      },
    });
    const step = response.data?.integration?.status?.setupState?.currentStep;
    if (!step?.redirectPrompt?.url) {
      throw new Error("The private GitHub App setup page did not open.");
    }
    openRedirectPrompt(step);
    return true;
  };

  const resume = (returnTo: string) => {
    if (!integrationId) {
      return false;
    }
    rememberIntegrationSetupReturn(args.organizationId, returnTo);
    if (currentStepName === "createApp") {
      openRedirectPrompt(currentStep ?? null);
      return Boolean(currentStep?.redirectPrompt?.url);
    }
    if (currentStepName === "selectOwner" || !currentStepName) {
      return false;
    }
    navigate(`${githubPrivateAppSetupPath(args.organizationId, integrationsBasePath)}?integrationId=${integrationId}`);
    return true;
  };

  return { connect, submitOwner, resume, ready, currentStepName };
}
