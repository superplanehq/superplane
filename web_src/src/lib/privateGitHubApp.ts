import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";

import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { integrationSetupPath, legacySettingsIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { startDirectGitHubConnect } from "@/lib/startDirectGitHubConnect";

export const PRIVATE_GITHUB_APP_CONFIG = { privateApp: true } as const;

/** Label for the customer GitHub App path beside hosted Connect GitHub. */
export const CREATE_PRIVATE_GITHUB_APP_LABEL = "Create your own GitHub App";

export function privateGitHubAppCreateConfiguration(integrationName: string): { privateApp: true } | undefined {
  if (integrationName !== "github") {
    return undefined;
  }
  return { ...PRIVATE_GITHUB_APP_CONFIG };
}

export function githubPrivateAppSetupPath(organizationId: string, basePath?: string): string {
  return integrationSetupPath(basePath ?? legacySettingsIntegrationsPath(organizationId), "github");
}

export function startPrivateGitHubAppSetup(args: {
  organizationId: string;
  returnTo?: string;
  integrationsBasePath?: string;
  goTo: (path: string) => void;
}): void {
  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  args.goTo(githubPrivateAppSetupPath(args.organizationId, args.integrationsBasePath));
}

export async function connectPrivateGitHubApp(args: {
  useWizard: boolean;
  organizationId: string;
  returnTo?: string;
  integrationsBasePath?: string;
  existingNames: Set<string>;
  connected: OrganizationsIntegration[];
  currentUserId?: string;
  goTo: (path: string) => void;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
}): Promise<boolean> {
  if (args.useWizard) {
    startPrivateGitHubAppSetup(args);
    return true;
  }

  return startDirectGitHubConnect({
    organizationId: args.organizationId,
    returnTo: args.returnTo,
    integrationsBasePath: args.integrationsBasePath,
    existingNames: args.existingNames,
    connected: args.connected,
    currentUserId: args.currentUserId,
    forceNew: true,
    goTo: args.goTo,
    create: (payload) =>
      args.create({
        ...payload,
        configuration: { ...payload.configuration, ...PRIVATE_GITHUB_APP_CONFIG },
      }),
  });
}
