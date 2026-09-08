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
  hostedGitHubAuthorizeURL,
  hostedGitHubInstallRequested,
  hostedGitHubStartedByLogin,
  hostedGitHubState,
  pendingGitHubInstallRequests,
  pendingGitHubInstallations,
  type PendingGitHubInstallRequest,
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
  /** GitHub OAuth authorize URL, to ask again which account to use. */
  authorizeUrl: string;
  /** GitHub login that authorized this connect. Empty when the field is absent. */
  githubLogin: string;
};

export type PendingGitHubRequestConnection = {
  id: string;
  connection: OrganizationsIntegration;
  requests: PendingGitHubInstallRequest[];
};

function accountPickerFromItem(item: OrganizationsIntegration | undefined): PendingGitHubAccountPicker | undefined {
  if (!item?.metadata?.id) {
    return undefined;
  }

  return {
    id: item.metadata.id,
    installations: pendingGitHubInstallations(item.status?.metadata),
    state: hostedGitHubState(item.status?.metadata),
    appSlug: hostedGitHubAppSlug(item.status?.metadata),
    authorizeUrl: hostedGitHubAuthorizeURL(item.status?.metadata),
    githubLogin: hostedGitHubStartedByLogin(item.status?.metadata),
  };
}

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

function isGitHubInstallRequest(item: OrganizationsIntegration): boolean {
  return item.metadata?.integrationName === "github" && hostedGitHubInstallRequested(item.status?.metadata);
}

/** Selects one request connection without combining metadata from different integrations. */
export function pendingGitHubRequestConnection(
  connected: OrganizationsIntegration[],
  currentUserId?: string,
  preferredIntegrationId?: string,
): PendingGitHubRequestConnection | undefined {
  if (!currentUserId) return undefined;

  const requested = connected.filter(isGitHubInstallRequest);
  const preferred = requested.find(
    (item) => item.metadata?.id === preferredIntegrationId && isOwnPendingGitHub(item, currentUserId),
  );
  const owned = preferred ?? requested.find((item) => startedByUserID(item) === currentUserId);
  const legacy = owned ?? (requested.length === 1 && startedByUserID(requested[0]) === "" ? requested[0] : undefined);
  const id = legacy?.metadata?.id;
  if (!legacy || !id) return undefined;

  return { id, connection: legacy, requests: pendingGitHubInstallRequests(legacy.status?.metadata) };
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
  preferredIntegrationId?: string,
): OrganizationsIntegration | undefined {
  const candidates = connected.filter(
    (item) =>
      item.metadata?.integrationName === "github" &&
      item.status?.state !== "ready" &&
      Boolean(item.status?.browserAction?.url),
  );
  const preferred = candidates.find(
    (item) => item.metadata?.id === preferredIntegrationId && isOwnPendingGitHub(item, currentUserId),
  );
  if (preferred) return preferred;

  const owned = candidates.find((item) => startedByUserID(item) === currentUserId);
  if (owned) return owned;

  const legacy = candidates.filter((item) => startedByUserID(item) === "");
  return legacy.length === 1 ? legacy[0] : undefined;
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
  preferredIntegrationId?: string,
): PendingGitHubAccountPicker | undefined {
  if (!currentUserId) {
    return undefined;
  }

  const preferredConnection = connected.find((item) => item.metadata?.id === preferredIntegrationId);
  const preferredPicker = githubAccountPickerFromConnection(preferredConnection, currentUserId);
  if (preferredPicker) return preferredPicker;

  const candidates = connected.filter((item) => {
    if (
      item.metadata?.integrationName !== "github" ||
      item.status?.state === "ready" ||
      !item.metadata?.id ||
      !isOwnPendingGitHub(item, currentUserId)
    ) {
      return false;
    }
    return pendingGitHubInstallations(item.status?.metadata).length >= 1;
  });
  const owned = candidates.find((item) => startedByUserID(item) === currentUserId);
  const legacyCandidates = candidates.filter((item) => startedByUserID(item) === "");
  const pending = owned ?? (legacyCandidates.length === 1 ? legacyCandidates[0] : undefined);
  return accountPickerFromItem(pending);
}

/**
 * The account picker for a connection the account picker already bound. A
 * bound connection keeps its installations and state, so onboarding shows the
 * picker again and the member can move the connection to another account.
 */
export function githubAccountPickerFromConnection(
  connection: OrganizationsIntegration | undefined,
  currentUserId?: string,
): PendingGitHubAccountPicker | undefined {
  if (!connection?.metadata?.id || !currentUserId) {
    return undefined;
  }
  if (connection.metadata.integrationName !== "github" || !isOwnPendingGitHub(connection, currentUserId)) {
    return undefined;
  }

  const picker = accountPickerFromItem(connection);
  if (!picker || picker.installations.length === 0 || picker.state === "") {
    return undefined;
  }

  return picker;
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
  preferredIntegrationId?: string;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
  update?: (payload: { id: string; configuration: Record<string, unknown> }) => Promise<void>;
  goTo?: (path: string) => void;
};

async function resumePendingGitHubConnect(args: StartDirectGitHubConnectArgs): Promise<boolean> {
  const picker = pendingGitHubAccountPicker(args.connected, args.currentUserId, args.preferredIntegrationId);
  if (picker) {
    rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
    if (isOnboardingSetupReturnPath(args.returnTo)) {
      // Onboarding asks again which GitHub account to use on every Connect
      // click, so the click goes to GitHub authorization instead of the
      // stored picker. Without a stored authorize URL the flow falls
      // through and starts a fresh connect, which also opens authorization.
      if (picker.authorizeUrl) {
        return followBrowserAction({ method: "GET", url: picker.authorizeUrl });
      }
      return false;
    }

    const path = githubInstallPickerPath(args.organizationId, picker.id, args.integrationsBasePath);
    if (args.goTo) {
      args.goTo(path);
      return true;
    }
    window.location.assign(path);
    return true;
  }

  const pending = pendingOwnGitHubWithAction(args.connected, args.currentUserId, args.preferredIntegrationId);
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
