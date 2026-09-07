import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";

import { followBrowserAction } from "@/lib/browserAction";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  hostedGitHubAppSlug,
  hostedGitHubState,
  pendingGitHubInstallations,
  type PendingGitHubInstallation,
} from "@/lib/hostedGitHubInstall";
import { integrationDetailPath, legacySettingsIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { INTEGRATION_SETUP_STAY_PARAM, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

export const GITHUB_SETUP_RETURN_PATH = "setupReturnPath";

export type PendingGitHubAccountPicker = {
  id: string;
  installations: PendingGitHubInstallation[];
  state: string;
  appSlug: string;
};

function startedByUserID(item: OrganizationsIntegration): string {
  const startedBy = item.status?.metadata?.startedByUserID;
  return typeof startedBy === "string" ? startedBy : "";
}

function isOwnPendingGitHub(item: OrganizationsIntegration, currentUserId?: string): boolean {
  const startedBy = startedByUserID(item);
  if (!currentUserId) {
    return false;
  }

  return startedBy === "" || startedBy === currentUserId;
}

function isOwnPendingGitHubItem(item: OrganizationsIntegration, currentUserId?: string): boolean {
  return (
    item.metadata?.integrationName === "github" &&
    item.status?.state !== "ready" &&
    isOwnPendingGitHub(item, currentUserId)
  );
}

export function pendingGitHubBrowserAction(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): OrganizationsBrowserAction | undefined {
  return pendingOwnGitHubWithAction(connected, currentUserId)?.status?.browserAction;
}

function pendingOwnGitHubWithAction(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): OrganizationsIntegration | undefined {
  return connected.find(
    (item) => isOwnPendingGitHubItem(item, currentUserId) && Boolean(item.status?.browserAction?.url),
  );
}

export function pendingGitHubInstallPicker(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): { id: string } | undefined {
  const picker = pendingGitHubAccountPicker(connected, currentUserId);
  return picker ? { id: picker.id } : undefined;
}

export function pendingGitHubAccountPicker(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
): PendingGitHubAccountPicker | undefined {
  if (!currentUserId) {
    return undefined;
  }

  const pending = connected.find((item) => {
    if (!isOwnPendingGitHubItem(item, currentUserId) || !item.metadata?.id) {
      return false;
    }
    return pendingGitHubInstallations(item.status?.metadata).length >= 1;
  });
  if (!pending?.metadata?.id) {
    return undefined;
  }

  return {
    id: pending.metadata.id,
    installations: pendingGitHubInstallations(pending.status?.metadata),
    state: hostedGitHubState(pending.status?.metadata),
    appSlug: hostedGitHubAppSlug(pending.status?.metadata),
  };
}

export function isOnboardingSetupReturnPath(path: string | undefined): boolean {
  if (!path) {
    return false;
  }

  const pathname = path.split("?")[0] ?? path;
  if (pathname === "/onboarding") {
    return true;
  }

  return pathname.includes("/workspaces/") && pathname.endsWith("/setup");
}

function setupReturnConfiguration(returnTo?: string): Record<string, unknown> | undefined {
  if (!returnTo) {
    return undefined;
  }

  return { [GITHUB_SETUP_RETURN_PATH]: returnTo };
}

/** Queue a Connect click until `/me` settles. Fail when that lookup ends with no user id. */
export function hostedGitHubConnectUserGate(
  currentUserId: string | undefined,
  currentUserResolved: boolean,
): "run" | "queue" | "fail" {
  if (currentUserId) {
    return "run";
  }
  if (!currentUserResolved) {
    return "queue";
  }
  return "fail";
}

export function persistGitHubSetupReturnPath(organizationId: string) {
  return async (payload: { id: string; configuration: Record<string, unknown> }) => {
    await organizationsUpdateIntegration(
      withOrganizationHeader({
        organizationId,
        path: { id: organizationId, integrationId: payload.id },
        body: { configuration: payload.configuration },
      }),
    );
  };
}

type StartDirectGitHubConnectArgs = {
  organizationId: string;
  returnTo?: string;
  integrationsBasePath?: string;
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
  goTo?: (path: string) => void;
};

async function resumePendingGitHubConnect(args: StartDirectGitHubConnectArgs): Promise<boolean> {
  const picker = pendingGitHubAccountPicker(args.connected, args.currentUserId);
  if (picker) {
    rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
    if (isOnboardingSetupReturnPath(args.returnTo)) {
      return true;
    }

    const path = githubInstallPickerPath(args.organizationId, picker.id, args.integrationsBasePath);
    if (args.goTo) {
      args.goTo(path);
      return true;
    }
    window.location.assign(path);
    return true;
  }

  const pending = pendingOwnGitHubWithAction(args.connected, args.currentUserId);
  const pendingAction = pending?.status?.browserAction;
  if (!pendingAction) {
    return false;
  }

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  await persistSetupReturnPath(args.update, pending.metadata?.id, args.returnTo);
  return followBrowserAction(pendingAction);
}

export async function startDirectGitHubConnect(args: StartDirectGitHubConnectArgs): Promise<boolean> {
  if (!args.forceNew && !args.currentUserId) {
    return false;
  }

  if (!args.forceNew && (await resumePendingGitHubConnect(args))) {
    return true;
  }

  const { result } = await createWithGeneratedName({
    baseName: "github",
    takenNames: args.existingNames,
    create: (name) => {
      const configuration = setupReturnConfiguration(args.returnTo);
      return args.create({
        integrationName: "github",
        name,
        ...(configuration ? { configuration } : {}),
      });
    },
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The GitHub App install page did not open.");
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

function githubInstallPickerPath(organizationId: string, integrationId: string, integrationsBasePath?: string) {
  const path = integrationDetailPath(
    integrationsBasePath ?? legacySettingsIntegrationsPath(organizationId),
    integrationId,
  );
  return `${path}?${INTEGRATION_SETUP_STAY_PARAM}=1`;
}
