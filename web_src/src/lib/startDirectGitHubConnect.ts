import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

export function pendingGitHubBrowserAction(
  connected: OrganizationsIntegration[],
): OrganizationsBrowserAction | undefined {
  const pending = connected.find(
    (item) =>
      item.metadata?.integrationName === "github" &&
      item.status?.state !== "ready" &&
      Boolean(item.status?.browserAction?.url),
  );
  return pending?.status?.browserAction;
}

export async function startDirectGitHubConnect(args: {
  organizationId: string;
  returnTo?: string;
  existingNames: Set<string>;
  connected: OrganizationsIntegration[];
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
}): Promise<boolean> {
  const pendingAction = pendingGitHubBrowserAction(args.connected);
  if (pendingAction) {
    rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
    return followBrowserAction(pendingAction);
  }

  const { result } = await createWithGeneratedName({
    baseName: "github",
    takenNames: args.existingNames,
    create: (name) => args.create({ integrationName: "github", name }),
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  return followBrowserAction(result.integration?.status?.browserAction);
}
