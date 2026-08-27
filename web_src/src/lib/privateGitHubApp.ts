import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

export const PRIVATE_GITHUB_APP_CONFIG = { privateApp: true } as const;

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
