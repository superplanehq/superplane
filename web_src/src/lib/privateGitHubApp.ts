import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

export const PRIVATE_GITHUB_APP_CONFIG = { privateApp: true } as const;

/** Label for the customer GitHub App path beside hosted Connect GitHub. */
export const CREATE_PRIVATE_GITHUB_APP_LABEL = "Create your own GitHub App";

export function privateGitHubAppCreateConfiguration(integrationName: string): { privateApp: true } | undefined {
  if (integrationName !== "github") {
    return undefined;
  }
  return { ...PRIVATE_GITHUB_APP_CONFIG };
}

export function githubPrivateAppSetupPath(organizationId: string): string {
  return `/${organizationId}/settings/integrations/github/setup`;
}

export function startPrivateGitHubAppSetup(args: {
  organizationId: string;
  returnTo?: string;
  goTo: (path: string) => void;
}): void {
  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  args.goTo(githubPrivateAppSetupPath(args.organizationId));
}
