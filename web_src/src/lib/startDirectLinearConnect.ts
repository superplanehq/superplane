import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { INTEGRATION_SETUP_RETURN_PATH_KEY, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

type LinearConnectionSummary = Pick<OrganizationsIntegration, "metadata" | "status">;

type StartDirectLinearConnectArgs = {
  organizationId: string;
  returnTo?: string;
  existingNames: Set<string>;
  /** Existing Linear connections in the organization. Used to resume pending OAuth or reuse a ready connection. */
  connected?: LinearConnectionSummary[];
  /** When set, skip reuse and resume and always create a new connection. */
  forceNew?: boolean;
  /** Called when a ready Linear connection already exists instead of opening OAuth. */
  onExistingReady?: (integrationId: string) => void;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
};

function setupReturnConfiguration(returnTo?: string): Record<string, unknown> | undefined {
  if (!returnTo) {
    return undefined;
  }

  return { [INTEGRATION_SETUP_RETURN_PATH_KEY]: returnTo };
}

export function findReadyLinearConnection(connected: LinearConnectionSummary[]): LinearConnectionSummary | undefined {
  return connected.find(
    (item) => item.metadata?.integrationName === "linear" && item.status?.state === "ready" && item.metadata?.id,
  );
}

export function findPendingLinearConnection(connected: LinearConnectionSummary[]): LinearConnectionSummary | undefined {
  return connected.find(
    (item) =>
      item.metadata?.integrationName === "linear" &&
      item.status?.state !== "ready" &&
      Boolean(item.status?.browserAction?.url),
  );
}

async function resumePendingLinearConnect(args: StartDirectLinearConnectArgs): Promise<boolean> {
  if (!args.connected?.length) {
    return false;
  }

  const pending = findPendingLinearConnection(args.connected);
  const action = pending?.status?.browserAction as OrganizationsBrowserAction | undefined;
  if (!action?.url) {
    return false;
  }

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  return followBrowserAction(action);
}

/** Creates a Linear connection and opens Linear authorization in this tab. */
export async function startDirectLinearConnect(args: StartDirectLinearConnectArgs): Promise<boolean> {
  if (!args.forceNew && args.connected?.length) {
    const ready = findReadyLinearConnection(args.connected);
    if (ready?.metadata?.id && args.onExistingReady) {
      args.onExistingReady(ready.metadata.id);
      return false;
    }

    if (await resumePendingLinearConnect(args)) {
      return true;
    }
  }

  const { result } = await createWithGeneratedName({
    baseName: "linear",
    takenNames: args.existingNames,
    create: (name) => {
      const configuration = setupReturnConfiguration(args.returnTo);
      return args.create({
        integrationName: "linear",
        name,
        ...(configuration ? { configuration } : {}),
      });
    },
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The Linear authorization page did not open.");
  }
  return followBrowserAction(action);
}
