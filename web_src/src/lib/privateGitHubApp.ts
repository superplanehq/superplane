import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { integrationSetupPath, legacySettingsIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

export const PRIVATE_GITHUB_APP_CONFIG = { privateApp: true } as const;
const GITHUB_SETUP_RETURN_PATH = "setupReturnPath";

/** Label for the customer GitHub App path beside hosted Connect GitHub. */
export const CREATE_PRIVATE_GITHUB_APP_LABEL = "Create your own GitHub App";

export type PrivateGitHubAppConfiguration = {
  privateApp: true;
  host?: string;
  factoryOnboarding?: boolean;
  setupReturnPath?: string;
};

export function privateGitHubAppCreateConfiguration(
  integrationName: string,
  extras?: { host?: string; factoryOnboarding?: boolean; returnTo?: string },
): PrivateGitHubAppConfiguration | undefined {
  if (integrationName !== "github") {
    return undefined;
  }
  const host = extras?.host?.trim();
  return {
    ...PRIVATE_GITHUB_APP_CONFIG,
    ...(host ? { host } : {}),
    ...(extras?.factoryOnboarding ? { factoryOnboarding: true } : {}),
    ...(extras?.returnTo ? { [GITHUB_SETUP_RETURN_PATH]: extras.returnTo } : {}),
  };
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
  host?: string;
  factoryOnboarding?: boolean;
  existingNames: Set<string>;
  connected: OrganizationsIntegration[];
  currentUserId?: string;
  goTo: (path: string) => void;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
  onCreated?: (integrationId: string) => Promise<void> | void;
}): Promise<boolean> {
  const host = args.host?.trim();
  if (args.useWizard && !host) {
    startPrivateGitHubAppSetup(args);
    return true;
  }

  const configuration = privateGitHubAppCreateConfiguration("github", {
    host,
    factoryOnboarding: args.factoryOnboarding,
    returnTo: args.returnTo,
  });
  const { result } = await createWithGeneratedName({
    baseName: "github",
    takenNames: args.existingNames,
    create: (name) =>
      args.create({
        integrationName: "github",
        name,
        configuration,
      }),
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  return openCreatedPrivateGitHubApp({
    organizationId: args.organizationId,
    integrationsBasePath: args.integrationsBasePath,
    useWizard: args.useWizard,
    integration: result.integration,
    goTo: args.goTo,
    onCreated: args.onCreated,
  });
}

async function openCreatedPrivateGitHubApp(args: {
  organizationId: string;
  integrationsBasePath?: string;
  useWizard: boolean;
  integration: OrganizationsCreateIntegrationResponse["integration"];
  goTo: (path: string) => void;
  onCreated?: (integrationId: string) => Promise<void> | void;
}): Promise<boolean> {
  const integrationId = args.integration?.metadata?.id?.trim() ?? "";
  if (integrationId && args.onCreated) {
    await args.onCreated(integrationId);
  }
  if (args.useWizard && integrationId) {
    args.goTo(
      `${githubPrivateAppSetupPath(args.organizationId, args.integrationsBasePath)}?integrationId=${integrationId}`,
    );
    return true;
  }
  const action = args.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The private GitHub App setup page did not open.");
  }
  return followBrowserAction(action);
}
