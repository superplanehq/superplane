import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { persistGitHubSetupReturnPath } from "@/lib/startDirectGitHubConnect";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

const SENTRY_SETUP_RETURN_PATH = "setupReturnPath";

function startedByUserID(item: OrganizationsIntegration): string {
  const startedBy = item.status?.metadata?.startedByUserID;
  return typeof startedBy === "string" ? startedBy : "";
}

function isOwnPendingSentry(item: OrganizationsIntegration, currentUserId?: string): boolean {
  const startedBy = startedByUserID(item);
  if (!currentUserId) {
    return false;
  }

  return startedBy === "" || startedBy === currentUserId;
}

function isOwnPendingSentryItem(item: OrganizationsIntegration, currentUserId?: string): boolean {
  return (
    item.metadata?.integrationName === "sentry" &&
    item.status?.state !== "ready" &&
    isOwnPendingSentry(item, currentUserId)
  );
}

function pendingOwnSentryWithAction(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): OrganizationsIntegration | undefined {
  return connected.find(
    (item) => isOwnPendingSentryItem(item, currentUserId) && Boolean(item.status?.browserAction?.url),
  );
}

export function pendingSentryBrowserAction(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): OrganizationsBrowserAction | undefined {
  return pendingOwnSentryWithAction(connected, currentUserId)?.status?.browserAction;
}

function setupReturnConfiguration(returnTo?: string): Record<string, unknown> | undefined {
  if (!returnTo) {
    return undefined;
  }

  return { [SENTRY_SETUP_RETURN_PATH]: returnTo };
}

export const persistSentrySetupReturnPath = persistGitHubSetupReturnPath;

type StartDirectSentryConnectArgs = {
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
  update?: (payload: { id: string; configuration: Record<string, unknown> }) => Promise<void>;
};

async function resumePendingSentryConnect(args: StartDirectSentryConnectArgs): Promise<boolean> {
  const pending = pendingOwnSentryWithAction(args.connected, args.currentUserId);
  const pendingAction = pending?.status?.browserAction;
  if (!pendingAction) {
    return false;
  }

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  await persistSetupReturnPath(args.update, pending.metadata?.id, args.returnTo);
  return followBrowserAction(pendingAction);
}

export async function startDirectSentryConnect(args: StartDirectSentryConnectArgs): Promise<boolean> {
  if (!args.forceNew && !args.currentUserId) {
    return false;
  }

  if (!args.forceNew && (await resumePendingSentryConnect(args))) {
    return true;
  }

  const { result } = await createWithGeneratedName({
    baseName: "sentry",
    takenNames: args.existingNames,
    create: (name) => {
      const configuration = setupReturnConfiguration(args.returnTo);
      return args.create({
        integrationName: "sentry",
        name,
        ...(configuration ? { configuration } : {}),
      });
    },
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The Sentry install page did not open.");
  }
  return followBrowserAction(action);
}

async function persistSetupReturnPath(
  update: ((payload: { id: string; configuration: Record<string, unknown> }) => Promise<void>) | undefined,
  integrationId: string | undefined,
  returnTo: string | undefined,
): Promise<void> {
  const configuration = setupReturnConfiguration(returnTo);
  if (!update || !integrationId || !configuration) {
    return;
  }

  await update({ id: integrationId, configuration });
}
