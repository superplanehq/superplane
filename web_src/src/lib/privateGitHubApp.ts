import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { integrationSetupPath, legacySettingsIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

export const PRIVATE_GITHUB_APP_CONFIG = { privateApp: true } as const;
const GITHUB_SETUP_RETURN_PATH = "setupReturnPath";

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

  const { result } = await createWithGeneratedName({
    baseName: "github",
    takenNames: args.existingNames,
    create: (name) =>
      args.create({
        integrationName: "github",
        name,
        configuration: {
          ...PRIVATE_GITHUB_APP_CONFIG,
          ...(args.returnTo ? { [GITHUB_SETUP_RETURN_PATH]: args.returnTo } : {}),
        },
      }),
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The private GitHub App setup page did not open.");
  }
  return followBrowserAction(action);
}
