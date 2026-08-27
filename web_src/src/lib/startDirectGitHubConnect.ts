import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { pendingGitHubInstallations } from "@/lib/hostedGitHubInstall";
import { INTEGRATION_SETUP_STAY_PARAM, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

function startedByUserID(item: OrganizationsIntegration): string {
  const startedBy = item.status?.metadata?.startedByUserID;
  return typeof startedBy === "string" ? startedBy : "";
}

function isOwnPendingGitHub(item: OrganizationsIntegration, currentUserId?: string): boolean {
  return Boolean(currentUserId) && startedByUserID(item) === currentUserId;
}

export function pendingGitHubBrowserAction(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): OrganizationsBrowserAction | undefined {
  const pending = connected.find(
    (item) =>
      item.metadata?.integrationName === "github" &&
      item.status?.state !== "ready" &&
      Boolean(item.status?.browserAction?.url) &&
      isOwnPendingGitHub(item, currentUserId),
  );
  return pending?.status?.browserAction;
}

export function pendingGitHubInstallPicker(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): { id: string } | undefined {
  const pending = connected.find(
    (item) =>
      item.metadata?.integrationName === "github" &&
      item.status?.state !== "ready" &&
      Boolean(item.metadata?.id) &&
      pendingGitHubInstallations(item.status?.metadata).length >= 2 &&
      isOwnPendingGitHub(item, currentUserId),
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
  currentUserId?: string;
  forceNew?: boolean;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
  goTo?: (path: string) => void;
}): Promise<boolean> {
  if (!args.forceNew) {
    const picker = pendingGitHubInstallPicker(args.connected, args.currentUserId);
    if (picker) {
      rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
      const path = `/${args.organizationId}/settings/integrations/${picker.id}?${INTEGRATION_SETUP_STAY_PARAM}=1`;
      if (args.goTo) {
        args.goTo(path);
        return true;
      }
      window.location.assign(path);
      return true;
    }

    const pendingAction = pendingGitHubBrowserAction(args.connected, args.currentUserId);
    if (pendingAction) {
      rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
      return followBrowserAction(pendingAction);
    }
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
