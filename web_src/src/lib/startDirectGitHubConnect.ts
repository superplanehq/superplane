import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { pendingGitHubInstallations } from "@/lib/hostedGitHubInstall";
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

export function pendingGitHubInstallPicker(connected: OrganizationsIntegration[]): { id: string } | undefined {
  const pending = connected.find(
    (item) =>
      item.metadata?.integrationName === "github" &&
      item.status?.state !== "ready" &&
      Boolean(item.metadata?.id) &&
      pendingGitHubInstallations(item.status?.metadata).length >= 2,
  );
  if (!pending?.metadata?.id) {
    return undefined;
  }
  return { id: pending.metadata.id };
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
  goTo?: (path: string) => void;
}): Promise<boolean> {
  const picker = pendingGitHubInstallPicker(args.connected);
  if (picker) {
    rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
    const path = `/${args.organizationId}/settings/integrations/${picker.id}`;
    if (args.goTo) {
      args.goTo(path);
      return true;
    }
    window.location.assign(path);
    return true;
  }

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
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The GitHub App install page did not open.");
  }
  return followBrowserAction(action);
}
